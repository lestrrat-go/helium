package xmlenc1

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
)

// refScope is the same-document resolution context of one decrypt: the subtree
// a reference may name and the id index over it. One scope is built per
// Decrypt/DecryptBytes call and threaded through the parse on parseState, so N
// references cost one document walk rather than N.
//
// It answers what a URI names, and nothing about what has been made of the
// answer: which targets have already been taken as candidates is a property of
// the EncryptedData being parsed rather than of URI resolution, so parseState
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

	// baseURI is the base URI of the EncryptedData, used to name the document
	// a refused reference was refused in. It takes no part in resolution:
	// XMLDSig core §4.4.3.2 defines a same-document reference as URI="" or a
	// bare "#..." fragment, so nothing here is ever resolved RELATIVE to a
	// base, and a URI carrying a path is external however that path compares
	// to this one.
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

// base returns the base URI an external CipherReference URI is joined against,
// and the empty string when there is no scope or the document has none. A nil
// scope is answered rather than dereferenced, so the external path behaves the
// same as the same-document one for an EncryptedData with no document.
func (s *refScope) base() string {
	if s == nil {
		return ""
	}
	return s.baseURI
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
// comment-inclusion flag, because a ds:RetrievalMethod resolves to a single
// ELEMENT rather than to a node-set, so the comment semantics that distinguish
// the bare-name forms from the XPointer ones make no difference here.
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
// its input rather than evaluating anything the document wrote.
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
// flood of transforms costs one refusal rather than a walk of them all.
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
// naming itself, is inert rather than recursive and there is no depth to bound.
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
	_, wholeDoc, sameDoc := referenceURIForm(uri)
	if sameDoc {
		octets, steps, err = resolveSameDocumentCipherReference(ctx, uri, wholeDoc, steps, ps, budget)
	} else {
		octets, err = resolveExternalCipherReference(ctx, uri, ps, budget)
	}
	if err != nil {
		return nil, err
	}

	// Whatever produced the octets, every transform not already consumed
	// applies to them in order. The list is at most maxCipherReferenceTransforms
	// long and every entry has been validated as #base64, so this loop's trip
	// count is fixed by the cap rather than by the document and each pass only
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
// takes the named element's character data, which is what decodeCipherValue
// already reads for an xenc:CipherValue — the same bounded, never-joined walk,
// charging the same budget.
func resolveSameDocumentCipherReference(ctx context.Context, uri string, wholeDoc bool, steps []string, ps *parseState, budget cipherValueBudget) ([]byte, []string, error) {
	target, err := resolveSameDocument(ctx, ps.refs, uri)
	if err != nil {
		return nil, nil, abort(ctx, err)
	}
	if len(steps) == 0 {
		octets, err := canonicalizeCipherReference(ctx, target, wholeDoc, budget)
		return octets, nil, err
	}
	octets, err := decodeCipherValue(ctx, target, budget)
	if err != nil {
		return nil, nil, err
	}
	return octets, steps[1:], nil
}

// resolveExternalCipherReference dereferences a URI that is not a
// same-document reference through the caller's resolver, and fails closed with
// ErrReferenceNotFound when there is none.
//
// The URI is joined against the document's base URI first, so a relative URI
// resolves as any other relative reference in that document does. The resolver
// is then asked for at most the budget's remaining allowance plus one byte, and
// the returned octets are charged in full: a resolver that can bound its own
// read never buffers an over-budget resource, and one that cannot is still held
// to the budget before anything is built from what it returned.
func resolveExternalCipherReference(ctx context.Context, uri string, ps *parseState, budget cipherValueBudget) ([]byte, error) {
	resolver := ps.cfg.cipherReferenceResolver
	if resolver == nil {
		return nil, abort(ctx, fmt.Errorf("%w: CipherReference URI %q is not a same-document reference and no Decryptor.CipherReferenceResolver is configured%s", ErrReferenceNotFound, uri, ps.refs.in()))
	}
	joined, err := joinReferenceURI(ps.refs.base(), uri)
	if err != nil {
		return nil, abort(ctx, err)
	}

	limit := -1
	if budget != nil {
		limit = budget.remaining()
	}
	var octets []byte
	if bounded, ok := resolver.(boundedReferenceResolver); ok {
		octets, err = bounded.resolveReferenceWithLimit(ctx, joined, limit)
	} else {
		octets, err = resolver.ResolveReference(ctx, joined)
	}
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

// canonicalizeCipherReference converts the node-set a same-document
// CipherReference names into octets by Canonical XML 1.0 without comments,
// which is the conversion xmldsig-core1 §4.4.3.3 requires of a node-set that
// has to become an octet stream.
//
// The canonical form is written into a budgetWriter rather than into a plain
// buffer, so an attacker-chosen subtree is stopped at the first byte past the
// remaining allowance instead of being canonicalized in full and rejected
// afterwards. The budget itself is charged once, with the final total, only
// after the write completed.
//
// A whole-document form (URI="" or "#xpointer(/)") naming the document element
// canonicalizes the DOCUMENT, so the top-level processing instructions that sit
// outside that element are included, as §4.4.3.2's "every non-comment node of
// the XML document" requires. Every other form names one element and
// canonicalizes that element's subtree as a node-set.
func canonicalizeCipherReference(ctx context.Context, target *helium.Element, wholeDoc bool, budget cipherValueBudget) ([]byte, error) {
	doc := target.OwnerDocument()
	if doc == nil {
		return nil, abort(ctx, fmt.Errorf("%w: CipherReference names an element with no owning document to canonicalize against", ErrMalformedEncrypted))
	}
	canon := c14n.NewCanonicalizer(c14n.C14N10)
	if !wholeDoc || doc.DocumentElement() != target {
		canon = canon.NodeSet(appendSubtreeNodes(nil, target))
	}
	var buf bytes.Buffer
	if err := canon.Canonicalize(doc, newBudgetWriter(&buf, budget)); err != nil {
		return nil, abort(ctx, err)
	}
	if budget != nil {
		if err := budget.charge(buf.Len()); err != nil {
			return nil, abort(ctx, err)
		}
	}
	return buf.Bytes(), nil
}

// appendSubtreeNodes appends cur's whole subtree to nodes as the node-set the
// canonicalizer selects on: each node, and for an element also its in-scope
// namespace axis and its attribute axis, since c14n renders only the namespace
// and attribute nodes the set holds.
//
// Children are enumerated through helium.Children, which stops at a
// foreign-owned child — an entity reference's shared Entity node is owned by
// the DTD, whose sibling pointers thread into the DTD declaration list — and is
// cycle-safe, so no DTD declaration spills into the set. The canonicalizer
// enumerates the same way, so the set and the walk agree on what the subtree is.
//
// The recursion is as deep as the subtree, which is what the canonicalizer's
// own walk costs too, so this adds no bound the canonicalization does not
// already have.
func appendSubtreeNodes(nodes []helium.Node, cur helium.Node) []helium.Node {
	nodes = append(nodes, cur)
	if elem, ok := helium.AsNode[*helium.Element](cur); ok {
		for _, ns := range domutil.InScopeNamespaces(elem, true) {
			nodes = append(nodes, helium.NewNamespaceNodeWrapper(ns, elem))
		}
		for _, attr := range elem.Attributes() {
			nodes = append(nodes, attr)
		}
	}
	for child := range helium.Children(cur) {
		nodes = appendSubtreeNodes(nodes, child)
	}
	return nodes
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
// with maxOccurs 1, so a second one is schema-invalid and refused rather than
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
			return nil
		}
		if seenTransforms {
			return fmt.Errorf("%w: CipherReference allows at most one xenc:Transforms", ErrMalformedEncrypted)
		}
		seenTransforms = true
		return eachChildElement(ctx, e, func(transform *helium.Element) error {
			if !isDSigElem(transform, "Transform") {
				return nil
			}
			if len(algorithms) == maxCipherReferenceTransforms {
				return fmt.Errorf("%w: CipherReference declares more than the %d transforms this package reads", ErrMalformedEncrypted, maxCipherReferenceTransforms)
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
