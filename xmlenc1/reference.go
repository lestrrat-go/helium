package xmlenc1

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"slices"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// refScope is the same-document resolution context of one decrypt: the subtree
// a reference may name and the id index over it. One scope is built per
// Decrypt/DecryptBytes call and threaded through the parse on parseState, so N
// references cost one document walk, well under N.
//
// It answers what a URI names, and nothing about what has been made of the
// answer: which targets have already been taken as candidates is a property of
// the EncryptedData being parsed, and not of URI resolution, so parseState
// holds it.
//
// It is per-call state and never shared between calls, which is what keeps a
// Decryptor safe to fire from several goroutines: the index below is written
// during resolution.
type refScope struct {
	// root is the subtree a same-document URI may name. It is the TOPMOST
	// ELEMENT ancestor of the EncryptedData, not OwnerDocument's document
	// element: an EncryptedData a caller holds detached from any document
	// still resolves references into its own subtree, which is where its
	// EncryptedKey then has to be. A document element's own walk up ends at
	// itself, so an attached EncryptedData sees the whole document.
	root *helium.Element

	// baseURI is the base URI of the EncryptedData, used ONLY to name the
	// document a refused reference was refused in. It takes no part in any
	// resolution: XMLDSig core §4.4.3.2 defines a same-document reference as
	// URI="" or a bare "#..." fragment, so nothing here is ever resolved
	// RELATIVE to a base, and a URI carrying a path is external however that
	// path compares to this one. The external form does join against a base,
	// but against the one in force at the element carrying the URI, and never
	// this one — resolveExternalCipherReference states why.
	baseURI string

	// index maps every id in root's subtree to every element carrying it. It
	// is nil until the first reference actually needs it and is built exactly
	// once thereafter, so a document whose references are all skipped (a Type
	// this package does not implement) never pays for the walk.
	index map[string][]*helium.Element
}

// newRefScope builds the resolution scope for a decrypt of elem. A nil elem
// yields a nil scope, which resolves nothing; the parse rejects a nil
// EncryptedData before any reference is read.
func newRefScope(elem *helium.Element) *refScope {
	if elem == nil {
		return nil
	}
	root := elem
	for parent := elem.Parent(); parent != nil; parent = parent.Parent() {
		anc, ok := helium.AsNode[*helium.Element](parent)
		if !ok {
			// The Document node above a document element, and nothing else in
			// a well-formed tree. Either way the element below it is the top.
			break
		}
		root = anc
	}
	return &refScope{
		root:    root,
		baseURI: helium.NodeGetBase(elem.OwnerDocument(), elem),
	}
}

// in names the document a reference was refused in, for the two reference
// errors. It is empty when the document has no base URI, which is the ordinary
// case for a document parsed from memory.
func (s *refScope) in() string {
	if s == nil || s.baseURI == "" {
		return ""
	}
	return fmt.Sprintf(" in %q", s.baseURI)
}

// referenceURIForm classifies a URI into the same-document forms XMLDSig core
// defines (§4.4.3.2-3), fail-closed. It reports:
//
//   - id: the bare id to resolve (empty for the whole-document forms);
//   - wholeDoc: the reference selects the root of the document;
//   - ok: the URI is a same-document reference at all (false → the caller
//     fails closed with ErrReferenceNotFound).
//
// The four recognized forms are:
//
//   - URI=""                     → the whole document.
//   - URI="#id"                  → the element with that id.
//   - URI="#xpointer(/)"         → the whole document.
//   - URI="#xpointer(id('id'))"  → the element with that id.
//
// Every other URI — an external reference, or any other #xpointer(...) scheme
// — is not a same-document reference and stays fail-closed. This mirrors
// xmldsig1's classification of the same four forms; it reports no
// comment-inclusion flag, because every consumer here is comment-free —
// [resolveCipherReferenceNodeSet] states why, and how to disprove it.
func referenceURIForm(uri string) (string, bool, bool) {
	if uri == "" {
		return "", true, true
	}
	if !strings.HasPrefix(uri, "#") {
		// External reference: not supported.
		return "", false, false
	}
	frag := uri[1:]
	if !strings.HasPrefix(frag, "xpointer(") {
		// Bare-name "#id" (no XPointer scheme). Any "#name" without a "(" is
		// a bare id.
		if strings.ContainsAny(frag, "()") {
			return "", false, false
		}
		return frag, false, true
	}
	// Full XPointer form: #xpointer(<expr>). Only the two bare-names
	// SHOULD-support schemes are honored.
	if !strings.HasSuffix(frag, ")") {
		return "", false, false
	}
	expr := strings.TrimSpace(frag[len("xpointer(") : len(frag)-1])
	if expr == "/" {
		return "", true, true
	}
	if id, ok := parseXPointerID(expr); ok {
		return id, false, true
	}
	return "", false, false
}

// parseXPointerID matches the XPointer id() form id('X') or id("X") and returns
// the quoted id. Anything else (a bare argument, a nested call, an unbalanced or
// mismatched quote) is rejected so only the two SHOULD-support schemes resolve.
func parseXPointerID(expr string) (string, bool) {
	if !strings.HasPrefix(expr, "id(") || !strings.HasSuffix(expr, ")") {
		return "", false
	}
	arg := strings.TrimSpace(expr[len("id(") : len(expr)-1])
	if len(arg) < 2 {
		return "", false
	}
	q := arg[0]
	if (q != '\'' && q != '"') || arg[len(arg)-1] != q {
		return "", false
	}
	inner := arg[1 : len(arg)-1]
	// The id itself must not contain the quote character (no embedded quote /
	// second argument), keeping this a strict single-id match.
	if strings.IndexByte(inner, q) >= 0 {
		return "", false
	}
	return inner, true
}

// resolveSameDocument returns the single element a same-document URI names
// within scope, building scope's id index on first use.
//
// A CipherReference names a NODE-SET, and no single element, so it resolves
// through [resolveCipherReferenceNodeSet] and this function answers only the
// ds:RetrievalMethod path, where xmldsig-core1 §4.4.3 does name a single
// element (an xenc:EncryptedKey) and there is no node-set to convert.
//
// It fails closed in three ways, all with a sentinel the caller can match: a
// URI that is not a same-document reference, and one matching no element, are
// ErrReferenceNotFound; one matching more than one element is
// ErrAmbiguousReference, because taking either of two elements answering to one
// id would let whoever injected the second choose the key.
//
// The index is built through domutil.BuildIDIndex, so the ID-name rule here is
// the same FROZEN rule xmldsig1 resolves references under and the two cannot
// disagree about which elements carry an id. That walk observes ctx once per
// node, per this package's per-node polling rule: its trip count is the size of
// a document the caller did not write.
func resolveSameDocument(ctx context.Context, scope *refScope, uri string) (*helium.Element, error) {
	id, wholeDoc, ok := referenceURIForm(uri)
	if !ok {
		return nil, fmt.Errorf("%w: %q is not a same-document reference", ErrReferenceNotFound, uri)
	}
	if scope == nil || scope.root == nil {
		return nil, fmt.Errorf("%w: %q has no document to resolve against", ErrReferenceNotFound, uri)
	}
	if wholeDoc {
		return scope.root, nil
	}
	if scope.index == nil {
		index, err := domutil.BuildIDIndex(ctx, scope.root)
		if err != nil {
			return nil, err
		}
		scope.index = index
	}
	matches := scope.index[id]
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: no element carries id %q%s", ErrReferenceNotFound, id, scope.in())
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("%w: id %q is carried by %d elements%s", ErrAmbiguousReference, id, len(matches), scope.in())
	}
}

// cipherReferenceBase64Transform is the one ds:Transform algorithm a
// CipherReference may declare here: the XMLDSig base64 transform, which decodes
// its input and evaluates nothing the document wrote.
const cipherReferenceBase64Transform = NamespaceDSig + "base64"

// maxCipherReferenceTransforms bounds how many ds:Transform children one
// xenc:Transforms may declare before the list is refused unread.
//
// The schema puts no bound on the sequence, so the count is whatever an
// attacker-supplied document says, and the list is read while the document is
// parsed — before any key is resolved and before anything the document says has
// been authenticated. Four is far above any interoperable list: a real
// CipherReference carries none or one, and the transform feature is OPTIONAL
// (xmlenc-core1 §3.3.1) in the first place. The cap is applied as the children
// are collected, so it holds ahead of the algorithm validation below and a
// flood of transforms costs one refusal, well short of a walk of them all.
const maxCipherReferenceTransforms = 4

// resolveCipherReference dereferences one xenc:CipherReference into the octets
// its CipherData stands for, charging budget for what it yields.
//
// xmlenc-core1 §3.3.1 makes support for the element REQUIRED and defines no
// dereferencing of its own: it requires "the same URI encoding, dereferencing,
// scheme, and HTTP response codes as that of [XMLDSIG-CORE1]". In that model
// (xmldsig-core1 §4.4.3.2 and §4.4.3.3) the same-document forms are normative
// MUSTs while HTTP dereferencing is RECOMMENDED, so the same-document forms are
// implemented unconditionally here and an external URI needs
// [Decryptor.CipherReferenceResolver].
//
// There are four outcomes, and the first two turn on a distinction easy to
// lose:
//
//   - an ABSENT @URI is ErrMalformedEncrypted, because the schema marks the
//     attribute required;
//   - a PRESENT and EMPTY @URI is the valid null URI naming the whole document,
//     which resolves like any other same-document form;
//   - a same-document URI yields a node-set, converted to octets either by
//     canonicalization (no transform) or by the declared #base64 transform,
//     which takes the named element's character data;
//   - any other URI is external and yields an octet stream from the caller's
//     resolver, or ErrReferenceNotFound when none is configured.
//
// The result is never re-parsed as a document: it goes to the block decryption
// as ciphertext. So a reference naming the EncryptedData that carries it, or
// naming itself, is inert, terminating at whatever those octets decrypt to, and there is no depth to bound.
func resolveCipherReference(ctx context.Context, elem *helium.Element, ps *parseState, budget cipherValueBudget) ([]byte, error) {
	uri, ok := elem.GetAttribute("URI")
	if !ok {
		return nil, abort(ctx, fmt.Errorf("%w: CipherReference has no URI attribute, which the xenc schema requires", ErrMalformedEncrypted))
	}
	steps, err := parseCipherReferenceTransforms(ctx, elem)
	if err != nil {
		return nil, err
	}

	var octets []byte
	_, _, sameDoc := referenceURIForm(uri)
	if sameDoc {
		octets, steps, err = resolveSameDocumentCipherReference(ctx, uri, steps, ps, budget)
	} else {
		octets, err = resolveExternalCipherReference(ctx, elem, uri, ps, budget)
	}
	if err != nil {
		return nil, err
	}

	// Whatever produced the octets, every transform not already consumed
	// applies to them in order. The list is at most maxCipherReferenceTransforms
	// long and every entry has been validated as #base64, so this loop's trip
	// count is fixed by the cap, and never by the document and each pass only
	// shrinks the value it is given.
	for range steps {
		octets, err = decodeBase64Octets(ctx, octets)
		if err != nil {
			return nil, err
		}
	}

	// A CipherReference may legitimately yield zero octets — an empty resource,
	// or an empty base64 value — and parseEncryptedData reads a nil CipherValue
	// as "this EncryptedData carried no CipherData at all". Returning an empty
	// but NON-NIL slice keeps those two apart.
	if octets == nil {
		return []byte{}, nil
	}
	return octets, nil
}

// resolveSameDocumentCipherReference resolves the node-set a same-document
// CipherReference URI names and converts it to octets, returning those octets
// and the transforms still to apply.
//
// The conversion depends on what the first transform expects, exactly as
// XMLDSig core's processing model does. With no transform at all the node-set
// must become octets on its own, which §4.4.3.3 does by canonicalization. With
// a #base64 transform first, the transform consumes the node-set directly and
// takes its STRING-VALUE, which xmldsig-core1 §6.6.2 defines as the
// document-order concatenation of the self::text() nodes — it "strips away the
// start and end tags of the identified element and any of its descendant
// elements", so base64 nested anywhere under the target contributes.
//
// Both conversions read the one node-set resolveCipherReferenceNodeSet built,
// so what a URI selects is decided once, and re-derived per conversion never.
func resolveSameDocumentCipherReference(ctx context.Context, uri string, steps []string, ps *parseState, budget cipherValueBudget) ([]byte, []string, error) {
	set, err := resolveCipherReferenceNodeSet(ctx, ps.refs, uri)
	if err != nil {
		return nil, nil, abort(ctx, err)
	}
	if len(steps) == 0 {
		octets, err := canonicalizeCipherReferenceNodeSet(ctx, set, budget)
		return octets, nil, err
	}
	octets, err := decodeCipherReferenceNodeSet(ctx, set, budget)
	if err != nil {
		return nil, nil, err
	}
	return octets, steps[1:], nil
}

// cipherReferenceNodeSet is what a same-document CipherReference URI names: the
// node-set both conversions above consume, described by its selection and
// materialized as no slice.
//
// It is described this way because the two conversions need
// different things of it — canonicalization needs every namespace and attribute
// node the canonicalizer renders, while the #base64 string-value walk needs only
// character data — and materializing the larger of the two for both would make a
// document pay for nodes its own transform list never reads.
type cipherReferenceNodeSet struct {
	// doc owns target and is what the canonicalization runs against. It is nil
	// for an element held outside any document, which only the canonicalization
	// needs to refuse: a string-value walk reads the subtree itself.
	doc *helium.Document
	// target is the element the selection is rooted at: the named element, or
	// the document element for the two whole-document forms.
	target *helium.Element
	// wholeDoc records that the URI selected the document, and not one
	// element, so the canonicalization can include the top-level processing
	// instructions that sit outside the document element.
	wholeDoc bool
	// includeComments records whether comment nodes are MEMBERS of the set.
	// resolveCipherReferenceNodeSet owns why it is false for every form.
	includeComments bool
}

// resolveCipherReferenceNodeSet resolves a same-document CipherReference URI
// into the node-set it names. It is the one construction site for that
// selection: what a URI selects, and which node kinds belong to the selection,
// are decided here and nowhere else.
//
// # Every form here is comment-free
//
// All four same-document forms — URI="", "#id", "#xpointer(/)" and
// "#xpointer(id('id'))" — set includeComments false, the XPointer pair
// included. A CipherReference carries no CanonicalizationMethod, so the only
// node-set-to-octet conversion available to it is the default one
// xmldsig-core1 §4.4.3.2 fixes: plain Canonical XML, which omits comments.
// That section recommends the XPointer forms only "if the application also
// intends to support any canonicalization that preserves comments", which this
// one cannot. Canonical XML 1.0 §2.1 likewise makes with-comments a parameter
// of the METHOD, and no property of the node-set, xmldsig1 converts a
// transform-free reference with plain C14N 1.0 here too, and Santuario guards
// comment output the same way — so a comment-bearing form would diverge from
// both the sibling package and the implementation this package interoperates
// with.
//
// The claim is measurable, and never merely asserted: bisecting
// [Decryptor.MaxCipherValueBytes] recovers the exact canonical octet count of a
// resolution, and adding a comment to the target leaves all four counts
// unchanged — 17 for "#t" and for "#xpointer(id('t'))", 402 for URI="" and 414
// for "#xpointer(/)" on the document TestCipherReference builds. A form that
// admitted comments would recover a longer count with the comment present.
func resolveCipherReferenceNodeSet(ctx context.Context, scope *refScope, uri string) (cipherReferenceNodeSet, error) {
	target, err := resolveSameDocument(ctx, scope, uri)
	if err != nil {
		return cipherReferenceNodeSet{}, err
	}
	_, wholeDoc, _ := referenceURIForm(uri)
	return cipherReferenceNodeSet{doc: target.OwnerDocument(), target: target, wholeDoc: wholeDoc}, nil
}

// nodeSetVisitor consumes one walk of a cipherReferenceNodeSet. The walk owns
// membership and document order; a visitor owns what the nodes it is handed
// become.
//
// enterElement and leaveElement bracket an element's descendants, so a visitor
// that tracks state down the tree (the namespace bindings in scope, say) can
// undo it on the way back up, recomputing nothing per node, which is what
// keeps the walk linear in the size of the subtree.
type nodeSetVisitor interface {
	enterElement(*helium.Element) error
	leaveElement(*helium.Element)
	visitNode(helium.Node) error
}

// walkCipherReferenceNodeSet hands every member of set to v in document order.
//
// The context is observed once per node, per this package's rule in context.go:
// the trip count is the size of a subtree the caller did not write, so a walk
// polling only at its ends would run to the end of an attacker-chosen document
// after the caller cancelled.
//
// Children are enumerated through helium.Children, which is cycle-safe and
// stops at a foreign-owned sibling — so a DTD declaration list cannot spill
// into the set — while still descending through an entity reference into the
// replacement content the canonicalizer reads there. The canonicalizer
// enumerates the same way, so the set and its own walk agree on what the
// subtree is.
//
// The recursion is as deep as the subtree, which is what the canonicalizer's
// own walk costs too, so this adds no bound the canonicalization does not
// already have.
func walkCipherReferenceNodeSet(ctx context.Context, set cipherReferenceNodeSet, v nodeSetVisitor) error {
	return walkNodeSetSubtree(ctx, set, set.target, v)
}

func walkNodeSetSubtree(ctx context.Context, set cipherReferenceNodeSet, cur helium.Node, v nodeSetVisitor) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	elem, isElem := helium.AsNode[*helium.Element](cur)
	switch {
	case isElem:
		if err := v.enterElement(elem); err != nil {
			return abort(ctx, err)
		}
	case isNodeSetMember(set, cur):
		if err := v.visitNode(cur); err != nil {
			return abort(ctx, err)
		}
	}
	for child := range helium.Children(cur) {
		if err := walkNodeSetSubtree(ctx, set, child, v); err != nil {
			return err
		}
	}
	if isElem {
		v.leaveElement(elem)
	}
	return nil
}

// isNodeSetMember reports whether a node that is not an element belongs to set.
// Only a comment is ever excluded, and only because every form
// resolveCipherReferenceNodeSet builds is comment-free.
func isNodeSetMember(set cipherReferenceNodeSet, n helium.Node) bool {
	if set.includeComments {
		return true
	}
	_, isComment := helium.AsNode[*helium.Comment](n)
	return !isComment
}

// leafVisitor supplies the do-nothing element bracketing that a visitor reading
// only leaf content embeds, so such a visitor states the node kinds it cares
// about and nothing else.
type leafVisitor struct{}

func (leafVisitor) enterElement(*helium.Element) error { return nil }

func (leafVisitor) leaveElement(*helium.Element) {}

// canonicalizeCipherReferenceNodeSet converts the node-set a same-document
// CipherReference names into octets by Canonical XML 1.0 without comments,
// which is the conversion xmldsig-core1 §4.4.3.3 requires of a node-set that
// has to become an octet stream.
//
// The canonical form is written into a budgetWriter, and never into a plain
// buffer, so an attacker-chosen subtree is stopped at the first byte past the
// remaining allowance instead of being canonicalized in full and rejected
// afterwards. The budget itself is charged once, with the final total, only
// after the write completed.
//
// budgetWriter also polls ctx on every Write, which is what stops a
// cancelled or expired caller here: the budget bounds OUTPUT OCTETS, not
// work, and this walk's cost is elements times in-scope namespace
// declarations, a shape that can run long while emitting almost nothing. For
// a whole-document form (below) there is no node-set collector ahead of this
// call at all, so this write loop is the only place in the whole resolution
// that ever observes cancellation.
//
// A whole-document form (URI="" or "#xpointer(/)") naming the document element
// canonicalizes the DOCUMENT, so the top-level processing instructions that sit
// outside that element are included, as §4.4.3.2's "every non-comment node of
// the XML document" requires. Every other form names one element and
// canonicalizes that element's subtree as an explicit node-set.
func canonicalizeCipherReferenceNodeSet(ctx context.Context, set cipherReferenceNodeSet, budget cipherValueBudget) ([]byte, error) {
	if set.doc == nil {
		return nil, abort(ctx, fmt.Errorf("%w: CipherReference names an element with no owning document to canonicalize against", ErrMalformedEncrypted))
	}
	canon := c14n.NewCanonicalizer(c14n.C14N10)
	if set.includeComments {
		canon = canon.Comments()
	}
	if !set.wholeDoc || set.doc.DocumentElement() != set.target {
		collector := canonicalNodeSetCollector{budget: budget}
		if err := walkCipherReferenceNodeSet(ctx, set, &collector); err != nil {
			return nil, abort(ctx, err)
		}
		canon = canon.NodeSet(collector.nodes)
	}
	var buf bytes.Buffer
	if err := canon.Canonicalize(set.doc, newBudgetWriter(ctx, &buf, budget)); err != nil {
		return nil, abort(ctx, err)
	}
	if budget != nil {
		if err := budget.charge(buf.Len()); err != nil {
			return nil, abort(ctx, err)
		}
	}
	return buf.Bytes(), nil
}

// canonicalNodeSetCollector materializes the node-set the canonicalizer selects
// on: each node of the subtree, and for an element the attribute nodes and the
// namespace nodes it needs, since c14n renders only the namespace and attribute
// nodes the set holds.
//
// # Why the namespace membership is not the in-scope axis
//
// The namespace AXIS of an element holds every binding in scope on it, so
// materializing it per element costs one wrapper per (element × in-scope
// declaration) — quadratic in a document that declares many prefixes high up,
// and bounded by nothing the output writer can see, since a declaration
// inherited by a descendant is rendered once at the top and costs no output
// bytes at all. A 210 KB document with 50k elements under 500 declarations
// reaches several gigabytes that way.
//
// The canonicalizer's node-set rule makes that unnecessary. It renders a
// namespace node on an element only when the NEAREST VISIBLE ANCESTOR does not
// carry the same prefix at the same URI, so an inherited binding is suppressed
// wherever it appears, and the only bindings that can ever be rendered are the
// selection root's in-scope axis plus, on each descendant, the declarations
// that actually CHANGE a binding. Those are what this collects, which is linear
// in the subtree. The one binding kept beyond them is the in-scope DEFAULT
// namespace, carried on every element that has one: c14n reads its absence from
// a set as an xmlns="" undeclaration, so dropping it where it is merely
// inherited would emit a reset the document never wrote.
//
// A comment node is never a member, which costs nothing today — the
// canonicalizer is comment-free for every form resolveCipherReferenceNodeSet
// builds — and keeps membership right by construction if one ever is not.
type canonicalNodeSetCollector struct {
	// nodes is the materialized set, in document order.
	nodes []helium.Node
	// inScope is the running prefix→binding map at the element being visited.
	// It is nil until the selection root seeds it with that element's whole
	// in-scope axis, and is mutated and restored as the walk descends and
	// returns, so the whole walk pays for one map, and never one per element.
	inScope map[string]*helium.Namespace
	// undo holds, per open element, the bindings inScope carried before that
	// element's declarations were applied, innermost frame last.
	undo [][]nsBinding
	// elements counts the elements admitted so far, which budget bounds.
	elements int
	budget   cipherValueBudget
}

// nsBinding is one entry of inScope as it stood before an element overwrote it.
// bound distinguishes a prefix that was previously unbound from one bound to a
// nil namespace, which cannot occur but would otherwise be indistinguishable.
type nsBinding struct {
	prefix string
	ns     *helium.Namespace
	bound  bool
}

func (c *canonicalNodeSetCollector) enterElement(e *helium.Element) error {
	c.elements++
	if err := c.chargeElements(); err != nil {
		return err
	}
	c.nodes = append(c.nodes, e)
	if c.inScope == nil {
		c.pushSelectionRoot(e)
	} else {
		c.pushDeclarations(e)
	}
	for _, attr := range e.Attributes() {
		c.nodes = append(c.nodes, attr)
	}
	return nil
}

func (c *canonicalNodeSetCollector) leaveElement(*helium.Element) {
	frame := c.undo[len(c.undo)-1]
	c.undo = c.undo[:len(c.undo)-1]
	// Innermost first, so an element that bound one prefix twice restores the
	// binding it found, and never the one it made in between.
	for _, binding := range slices.Backward(frame) {
		if !binding.bound {
			delete(c.inScope, binding.prefix)
			continue
		}
		c.inScope[binding.prefix] = binding.ns
	}
}

// chargeElements refuses a selection whose element count alone cannot fit the
// budget. Every element in the set contributes at least its own start and end
// tags to the canonical form, so a count past the remaining allowance is over
// budget however small the document's text is — and knowing that here stops the
// set from being built at all, which the output writer downstream cannot do
// because it never sees the nodes that were materialized to feed it.
//
// The refusal is the budget's OWN over-limit error, produced by a charge sized
// to a count that is guaranteed to hit its error branch and never mutate its
// state, exactly as budgetWriter does.
func (c *canonicalNodeSetCollector) chargeElements() error {
	if c.budget == nil {
		return nil
	}
	limit := c.budget.remaining()
	if limit < 0 || c.elements <= limit {
		return nil
	}
	return c.budget.charge(c.elements)
}

// pushSelectionRoot seeds the running bindings with the selection root's whole
// in-scope axis and makes every one of them a member, which is what the root
// has to declare: its own ancestors are outside the set, so the canonicalizer
// finds no visible ancestor to suppress an inherited binding against.
func (c *canonicalNodeSetCollector) pushSelectionRoot(e *helium.Element) {
	c.inScope = domutil.InScopeNamespaces(e, true)
	c.undo = append(c.undo, nil)
	for _, ns := range c.inScope {
		c.nodes = append(c.nodes, helium.NewNamespaceNodeWrapper(ns, e))
	}
}

// pushDeclarations applies e's own namespace declarations to the running
// bindings and makes members of the ones that CHANGE a binding, plus the
// in-scope default namespace.
//
// A declaration that merely repeats the binding already in scope changes
// nothing and is not a member: the canonicalizer suppresses such a
// redeclaration against the nearest visible ancestor, so admitting it would
// have to be suppressed again, and never rendered.
func (c *canonicalNodeSetCollector) pushDeclarations(e *helium.Element) {
	var frame []nsBinding
	var boundDefault bool
	for _, ns := range e.Namespaces() {
		prefix := ns.Prefix()
		if prefix == lexicon.PrefixXML {
			// Never explicitly declared in canonical output, and dropped from
			// the root's axis above, so it is not a member anywhere.
			continue
		}
		prev, bound := c.inScope[prefix]
		frame = append(frame, nsBinding{prefix: prefix, ns: prev, bound: bound})
		c.inScope[prefix] = ns
		if bound && prev.URI() == ns.URI() {
			continue
		}
		c.nodes = append(c.nodes, helium.NewNamespaceNodeWrapper(ns, e))
		boundDefault = boundDefault || prefix == ""
	}
	// An element may carry an ACTIVE namespace whose prefix nothing in scope
	// declares — a programmatically built DOM can, and domutil.InScopeNamespaces
	// reads such a namespace as a binding, so the two must agree on what is in
	// scope or a descendant would compare against a binding the root never had.
	if ns := e.Namespace(); ns != nil && ns.Prefix() != lexicon.PrefixXML {
		if _, bound := c.inScope[ns.Prefix()]; !bound {
			frame = append(frame, nsBinding{prefix: ns.Prefix()})
			c.inScope[ns.Prefix()] = ns
			c.nodes = append(c.nodes, helium.NewNamespaceNodeWrapper(ns, e))
			boundDefault = boundDefault || ns.Prefix() == ""
		}
	}
	c.undo = append(c.undo, frame)
	if boundDefault {
		return
	}
	if ns, ok := c.inScope[""]; ok {
		c.nodes = append(c.nodes, helium.NewNamespaceNodeWrapper(ns, e))
	}
}

func (c *canonicalNodeSetCollector) visitNode(n helium.Node) error {
	c.nodes = append(c.nodes, n)
	return nil
}

// decodeCipherReferenceNodeSet applies the XMLDSig #base64 transform to a
// same-document node-set: it takes the set's string-value and decodes it.
//
// xmldsig-core1 §6.6.2 defines that input as the self::text() nodes of the
// selection in document order, concatenated — the markup between them is
// stripped, so base64 written across a grandchild, or split at an element
// boundary, decodes exactly as the same characters written directly under the
// target would.
//
// The value is counted before it is built, exactly as decodeCipherValue counts
// an xenc:CipherValue: xs:base64Binary permits whitespace between characters
// and the text may arrive as any number of nodes, so the lexical length an
// attacker controls has no relation to the decoded bytes the budget charges.
// One pass counts through an xmlbase64.Counter, which carries the counting
// state across each node boundary because a base64 quantum (and padding itself)
// may be split between them; only then is the charge made, and only then are
// the characters the decoder sees built. What is held is therefore bounded by
// the budget, and never by the size of the selection.
func decodeCipherReferenceNodeSet(ctx context.Context, set cipherReferenceNodeSet, budget cipherValueBudget) ([]byte, error) {
	var counting nodeSetBase64Counter
	if err := walkCipherReferenceNodeSet(ctx, set, &counting); err != nil {
		return nil, err
	}
	if budget != nil {
		if err := budget.charge(counting.counter.DecodedLen()); err != nil {
			return nil, abort(ctx, err)
		}
	}
	collecting := nodeSetBase64Chars{chars: make([]byte, 0, counting.counter.Chars())}
	if err := walkCipherReferenceNodeSet(ctx, set, &collecting); err != nil {
		return nil, err
	}
	// The characters are already stripped, so this is the decode
	// xmlbase64.DecodeString performs minus the copy that would convert them to
	// a string, sized at the count the budget charged, and never at
	// encoding/base64's own padding-blind DecodedLen.
	decoded := make([]byte, counting.counter.DecodedLen())
	n, err := base64.StdEncoding.Decode(decoded, collecting.chars)
	if err != nil {
		return nil, abort(ctx, fmt.Errorf("%w: invalid base64 in the CipherReference target: %v", ErrMalformedEncrypted, err))
	}
	return decoded[:n], nil
}

// nodeSetCharacterData returns what one node contributes to a node-set's
// string-value: the content of a text or CDATA node, and nothing for any other
// kind.
//
// A comment and a processing instruction contribute nothing because they are
// not text nodes — reading them would splice their text straight into the
// base64 stream. An entity reference contributes nothing ITSELF because the
// walk descends into its replacement content, whose text nodes are handed over
// individually; taking the reference's own content as well would count the
// replacement twice.
func nodeSetCharacterData(n helium.Node) []byte {
	if t, ok := helium.AsNode[*helium.Text](n); ok {
		return t.Content()
	}
	if c, ok := helium.AsNode[*helium.CDATASection](n); ok {
		return c.Content()
	}
	return nil
}

// nodeSetBase64Counter is the counting pass of decodeCipherReferenceNodeSet: it
// measures the value without building it.
type nodeSetBase64Counter struct {
	leafVisitor
	counter xmlbase64.Counter
}

func (v *nodeSetBase64Counter) visitNode(n helium.Node) error {
	v.counter.Add(nodeSetCharacterData(n))
	return nil
}

// nodeSetBase64Chars is the building pass: it appends just the characters the
// decoder will see, into a buffer sized at the count the budget already
// admitted.
type nodeSetBase64Chars struct {
	leafVisitor
	chars []byte
}

func (v *nodeSetBase64Chars) visitNode(n helium.Node) error {
	v.chars = xmlbase64.AppendStripped(v.chars, nodeSetCharacterData(n))
	return nil
}

// resolveExternalCipherReference dereferences a URI that is not a
// same-document reference through the caller's resolver, and fails closed with
// ErrReferenceNotFound when there is none.
//
// The URI is joined against the base URI in force at elem — the
// xenc:CipherReference element that WROTE the URI — so it resolves as any other
// relative reference written at that spot does. The base is computed here,
// and never taken once from the EncryptedData, because xml:base applies per
// element: a CipherReference, or anything between it and the EncryptedData, may
// narrow the base the URI is written in, and a document read from a file
// carries that file's own URL as the outermost base.
//
// What the resolver returns is a stream, and this package reads it: at most the
// budget's remaining allowance plus one probe byte, abandoned when ctx is done,
// and closed on every path. The octets are then charged in full, so nothing is
// built from a resource the budget would not admit. ReferenceResolver owns the
// whole of that division and the limit of what it can promise.
func resolveExternalCipherReference(ctx context.Context, elem *helium.Element, uri string, ps *parseState, budget cipherValueBudget) ([]byte, error) {
	resolver := ps.cfg.cipherReferenceResolver
	if resolver == nil {
		return nil, abort(ctx, fmt.Errorf("%w: CipherReference URI %q is not a same-document reference and no Decryptor.CipherReferenceResolver is configured%s", ErrReferenceNotFound, uri, ps.refs.in()))
	}
	joined, err := joinReferenceURI(helium.NodeGetBase(elem.OwnerDocument(), elem), uri)
	if err != nil {
		return nil, abort(ctx, err)
	}

	limit := -1
	if budget != nil {
		limit = budget.remaining()
	}
	rc, err := resolver.ResolveReference(ctx, joined)
	if err != nil {
		// A resolver reporting an error may still have handed over a live
		// stream, and closing it here is what keeps that shape from leaking:
		// the error is the resolver's own, and rc.Close's is discarded rather
		// than folded into it, so the caller sees the resolver's error
		// unchanged.
		if rc != nil {
			rc.Close()
		}
		return nil, abort(ctx, err)
	}
	// A resolver reporting neither a stream nor an error is refused, and never
	// dereferenced: the alternative is a panic raised inside a decrypt of an
	// unauthenticated document, and this is a third-party implementation the
	// package cannot check any other way.
	if rc == nil {
		return nil, abort(ctx, fmt.Errorf("%w: the configured Decryptor.CipherReferenceResolver returned no stream for %q", ErrReferenceNotFound, joined))
	}
	octets, err := readBoundedReference(ctx, rc, joined, limit)
	if err != nil {
		return nil, abort(ctx, err)
	}
	if budget != nil {
		if err := budget.charge(len(octets)); err != nil {
			return nil, abort(ctx, err)
		}
	}
	return octets, nil
}

// decodeBase64Octets applies the XMLDSig base64 transform to an octet stream,
// which is the form every transform past the first one sees.
//
// No budget is charged: base64 decoding only ever shrinks its input, and that
// input was charged when it was produced, so the value this returns is within
// whatever allowance already admitted the value it was given.
func decodeBase64Octets(ctx context.Context, octets []byte) ([]byte, error) {
	chars := xmlbase64.AppendStripped(make([]byte, 0, len(octets)), octets)
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(chars)))
	n, err := base64.StdEncoding.Decode(decoded, chars)
	if err != nil {
		return nil, abort(ctx, fmt.Errorf("%w: invalid base64 in the CipherReference result: %v", ErrMalformedEncrypted, err))
	}
	return decoded[:n], nil
}

// parseCipherReferenceTransforms reads the ds:Transform algorithms one
// xenc:CipherReference declares, and returns them only if EVERY one of them is
// supported.
//
// The wrapper is xenc:Transforms — the xenc namespace, not the ds one, which
// the xenc schema declares as a local element of type xenc:TransformsType —
// while its children are ds:Transform. CipherReferenceType declares the wrapper
// with maxOccurs 1, so a second one is schema-invalid and refused, and never
// merged.
//
// Validating the whole list before returning it is the point: a supported
// algorithm standing ahead of an unsupported one must not be executed first,
// because that would spend the work of a transform on a list that was never
// going to complete. Only the XMLDSig #base64 transform is accepted, and the
// refusal is conforming: xmlenc-core1 §3.3.1 marks both the Transform feature
// and the particular transform algorithms OPTIONAL. XPath and XSLT above all
// stay out — either one evaluates an expression the document chose over a
// document nothing has authenticated, which is unbounded compute bought with a
// few bytes of markup.
func parseCipherReferenceTransforms(ctx context.Context, elem *helium.Element) ([]string, error) {
	var algorithms []string
	var seenTransforms bool
	if err := eachChildElement(ctx, elem, func(e *helium.Element) error {
		if !isXMLEncElem(e, "Transforms") {
			return fmt.Errorf("%w: CipherReference holds a %s element, which xenc:CipherReferenceType does not allow", ErrMalformedEncrypted, e.Name())
		}
		if seenTransforms {
			return fmt.Errorf("%w: CipherReference allows at most one xenc:Transforms", ErrMalformedEncrypted)
		}
		seenTransforms = true
		var children int
		return eachChildElement(ctx, e, func(transform *helium.Element) error {
			children++
			if children > maxCipherReferenceTransforms {
				return fmt.Errorf("%w: CipherReference declares more than the %d transforms this package reads", ErrMalformedEncrypted, maxCipherReferenceTransforms)
			}
			if !isDSigElem(transform, "Transform") {
				return fmt.Errorf("%w: xenc:Transforms holds a %s element, which xenc:TransformsType does not allow", ErrMalformedEncrypted, transform.Name())
			}
			algorithm, _ := transform.GetAttribute("Algorithm")
			algorithms = append(algorithms, algorithm)
			return nil
		})
	}); err != nil {
		return nil, err
	}
	// This loop is over a list already bounded at maxCipherReferenceTransforms,
	// not over anything the document sizes, so it is not an eachSibling site;
	// its error return still runs through abort.
	for _, algorithm := range algorithms {
		if algorithm != cipherReferenceBase64Transform {
			return nil, abort(ctx, fmt.Errorf("%w: %w", ErrMalformedEncrypted, &UnsupportedAlgorithmError{Parameter: paramCipherReferenceTransform, Algorithm: algorithm}))
		}
	}
	return algorithms, nil
}
