package xmldsig1

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath1"

	helium "github.com/lestrrat-go/helium"
)

// Transform represents a single step in a reference transform pipeline.
type Transform interface {
	URI() string
}

// envelopedTransform implements the enveloped-signature transform.
type envelopedTransform struct{}

func (envelopedTransform) URI() string { return TransformEnvelopedSignature }

// Enveloped returns the enveloped-signature transform. When applied during
// signing or verification, the ds:Signature element and its descendants are
// omitted from the canonical input. This is done on a deep copy of the
// document, so the caller's live DOM is never mutated.
func Enveloped() Transform { return envelopedTransform{} }

// c14nTransform applies canonicalization.
type c14nTransform struct {
	method string
}

func (t c14nTransform) URI() string { return t.method }

// C14NTransform returns a canonicalization transform for the given method URI.
func C14NTransform(method string) Transform {
	return c14nTransform{method: method}
}

// excC14NTransform applies Exclusive C14N with optional inclusive namespace prefixes.
type excC14NTransform struct {
	prefixes []string
}

func (excC14NTransform) URI() string { return ExcC14N10 }

// Prefixes returns the inclusive namespace prefixes for this transform. The
// returned slice is a copy, so a caller cannot mutate the transform's internal
// prefix list through it.
func (t excC14NTransform) Prefixes() []string { return slices.Clone(t.prefixes) }

// ExcC14NTransform returns an Exclusive C14N transform with optional
// inclusive namespace prefixes. The prefixes are copied, so a later mutation of
// the caller's slice cannot alter the returned transform.
func ExcC14NTransform(prefixes ...string) Transform {
	return excC14NTransform{prefixes: slices.Clone(prefixes)}
}

// cloneReferenceTransforms returns a deep copy of a Reference's transform slice:
// a fresh backing array plus a copy of each mutable transform's internal state.
// Signer.clone uses it so a later caller mutation of the original Transforms
// slice (or of a prefix slice a transform holds) cannot alter an
// already-configured Signer or race with an in-flight signing operation.
func cloneReferenceTransforms(transforms []Transform) []Transform {
	if transforms == nil {
		return nil
	}
	out := make([]Transform, len(transforms))
	for i, t := range transforms {
		out[i] = cloneTransform(t)
	}
	return out
}

// cloneTransform returns a copy of t that shares no mutable state with t. Only
// excC14NTransform carries mutable state (its prefix slice); every other
// Transform is an immutable value and is returned unchanged.
func cloneTransform(t Transform) Transform {
	exc, ok := t.(excC14NTransform)
	if !ok {
		return t
	}
	return excC14NTransform{prefixes: slices.Clone(exc.prefixes)}
}

// transformStep is the algorithm-agnostic view of a single Reference transform,
// shared by the signing (typed Transform) and verification (parsedTransform)
// paths so both interpret a transform list identically.
type transformStep struct {
	algorithm string
	prefixes  []string
	// xpathExpr and xpathNS carry an XPath filter transform's expression and its
	// in-scope namespace bindings (from the ds:Transform/XPath element). They are
	// populated only when algorithm == TransformXPath.
	xpathExpr string
	xpathNS   map[string]string
	// xpathHere is the ds:XPath element bearing the expression, threaded through so
	// the here() function (XMLDSig core §6.6.3.1) resolves to it. It is nil on the
	// signing path (no bearing node), where here() then fails closed. Populated
	// only when algorithm == TransformXPath. Its position matches parsedTransform
	// so a transformStep(parsedTransform) conversion stays valid.
	xpathHere helium.Node
	// stylesheet carries the XSLT transform's serialized xsl:stylesheet subtree
	// (from the ds:Transform element). It is populated only when
	// algorithm == TransformXSLT.
	stylesheet []byte
}

// xpathFilter is a resolved XPath filter transform: an XPath 1.0 boolean
// expression, the namespace bindings it is evaluated under, and the bearing
// ds:XPath element that the here() function resolves to (nil when here() has no
// bearing node, e.g. the signing path).
type xpathFilter struct {
	expr     *xpath1.Expression
	ns       map[string]string
	hereNode helium.Node
}

// canonicalize applies the appropriate c14n mode for the given method URI
// to the document, returning the canonical bytes.
func canonicalize(method string, doc *helium.Document, prefixes []string) ([]byte, error) {
	mode, comments, err := resolveC14NMode(method)
	if err != nil {
		return nil, err
	}
	canon := c14n.NewCanonicalizer(mode)
	if comments {
		canon = canon.Comments()
	}
	if mode == c14n.ExclusiveC14N10 && len(prefixes) > 0 {
		canon = canon.InclusiveNamespaces(prefixes)
	}
	return canon.CanonicalizeTo(doc)
}

// canonicalizeSubtree canonicalizes a single element subtree by canonicalizing
// the node-set of that subtree against its owning document.
func canonicalizeSubtree(method string, elem *helium.Element, prefixes []string) ([]byte, error) {
	return canonicalizeNodeSet(method, collectSubtreeNodes(elem), elem.OwnerDocument(), prefixes)
}

// canonicalizeNodeSet canonicalizes an explicit node-set against doc using the
// given method. It is the shared node-set -> octet stage for a plain subtree
// reference and for a reference whose XPath filter transforms have narrowed the
// node-set. A comment node is emitted only when it is BOTH in the node-set and
// the method is a WithComments variant (see effectiveC14NMethod), so a
// comment-excluding reference form never emits comments regardless of the c14n
// method.
func canonicalizeNodeSet(method string, nodes []helium.Node, doc *helium.Document, prefixes []string) ([]byte, error) {
	mode, comments, err := resolveC14NMode(method)
	if err != nil {
		return nil, err
	}
	canon := c14n.NewCanonicalizer(mode).NodeSet(nodes)
	if comments {
		canon = canon.Comments()
	}
	if mode == c14n.ExclusiveC14N10 && len(prefixes) > 0 {
		canon = canon.InclusiveNamespaces(prefixes)
	}
	return canon.CanonicalizeTo(doc)
}

// collectDocumentNodes returns the whole-document node-set: every top-level
// comment and processing-instruction plus the document element's full subtree
// (elements, their in-scope namespace nodes, attributes, and descendants). It is
// the initial node-set for a whole-document reference (URI="" or "#xpointer(/)")
// when a transform needs explicit node membership. The materializer removes
// comments for a comment-excluding Reference form before applying that transform.
func collectDocumentNodes(doc *helium.Document) []helium.Node {
	var nodes []helium.Node
	for c := range helium.Children(doc) {
		switch c.Type() {
		case helium.ElementNode:
			nodes = append(nodes, collectSubtreeNodes(c)...)
		case helium.CommentNode, helium.ProcessingInstructionNode:
			nodes = append(nodes, c)
		}
	}
	return nodes
}

// defaultXPathOpLimit bounds the number of evaluation operations a single XPath
// evaluation may perform, matching libxml2's opLimit mechanism (see
// xpath1.Evaluator.OpLimit). xpath1 already caps recursion depth (5000) and
// node-set length (10M); this additionally bounds total operation count so an
// attacker-supplied XPath filter or XPointer expression cannot stall verification
// with a pathological expression. The limit is generous — far above any realistic
// same-document reference — while still finite, so a legitimate signature is
// never rejected for exceeding it.
const defaultXPathOpLimit = 100_000_000

// hereFunction implements the XMLDSig here() function (core §6.6.3.1): it returns
// a node-set containing the single element that bears the XPath expression — the
// ds:XPath element of an XPath filter transform. The bearing node is threaded in
// at evaluator-construction time.
type hereFunction struct {
	node helium.Node
}

// Eval returns the bearing node as a one-node node-set. here() takes no
// arguments, so a call with any argument fails closed. When no bearing node was
// threaded in (the signing path, or a URI-borne XPointer), here() is unavailable
// and fails closed with ErrHereUnavailable rather than resolving to a wrong node.
func (h hereFunction) Eval(_ context.Context, args []*xpath1.Result) (*xpath1.Result, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("%w: here() takes no arguments", ErrUnsupportedTransform)
	}
	if h.node == nil {
		return nil, ErrHereUnavailable
	}
	return &xpath1.Result{Type: xpath1.NodeSetResult, NodeSet: []helium.Node{h.node}}, nil
}

// newDSigXPathEvaluator builds the single bounded XPath 1.0 evaluator used by
// both the XPath filter transform and the general XPointer resolver, unifying
// namespace handling, the here() function, and the security bound (OpLimit) in
// one place. ns are the prefix->URI bindings the expression is evaluated under;
// hereNode is the bearing element for here() (nil disables here(), which then
// fails closed); opLimit bounds the operation count (0 = unlimited).
func newDSigXPathEvaluator(ns map[string]string, hereNode helium.Node, opLimit int) xpath1.Evaluator {
	eval := xpath1.NewEvaluator()
	if len(ns) > 0 {
		eval = eval.Namespaces(ns)
	}
	if opLimit > 0 {
		eval = eval.OpLimit(opLimit)
	}
	return eval.Function("here", hereFunction{node: hereNode})
}

// compileXPathFilterExpression compiles and statically validates the transform
// expression during the complete-list validation pass. Wrapping it in
// fn:boolean makes this compiled form identical to the one evaluated for each
// input node.
func compileXPathFilterExpression(expr string, eval xpath1.Evaluator) (*xpath1.Expression, error) {
	compiled, err := xpath1.Compile("boolean(" + expr + ")")
	if err != nil {
		return nil, fmt.Errorf("%w: invalid XPath transform expression %q: %v", ErrUnsupportedTransform, expr, err)
	}
	if err := eval.Validate(compiled); err != nil {
		return nil, fmt.Errorf("%w: invalid XPath transform expression %q: %v", ErrUnsupportedTransform, expr, err)
	}
	return compiled, nil
}

// applyXPathFilter implements the XMLDSig XPath filter transform
// (http://www.w3.org/TR/1999/REC-xpath-19991116, core §6.6.3): the expression is
// evaluated once per input node with that node as the context node, under the
// transform's in-scope namespace bindings, and the node is kept when the result
// converts to boolean true. The expression is wrapped in fn:boolean so the XPath
// data-model boolean conversion (a non-empty node-set, a non-zero number, a
// non-empty string) governs membership. Evaluation runs on the shared bounded
// evaluator (namespaces, here(), and the OpLimit security bound). Expressions are
// compiled and statically validated during complete-list validation, before
// execution starts. An evaluation error is fail-closed as
// ErrUnsupportedTransform so a reference never digests an unfiltered node-set.
func applyXPathFilter(ctx context.Context, nodes []helium.Node, f xpathFilter) ([]helium.Node, error) {
	eval := newDSigXPathEvaluator(f.ns, f.hereNode, defaultXPathOpLimit)
	kept := make([]helium.Node, 0, len(nodes))
	for _, n := range nodes {
		r, err := eval.Evaluate(ctx, f.expr, n)
		if err != nil {
			// Preserve the here()-unavailable sentinel as a matchable typed error
			// rather than flattening it into an ErrUnsupportedTransform string, so a
			// caller can tell "here() has no bearing node" from a generic malformed
			// transform. Both are fail-closed.
			if errors.Is(err, ErrHereUnavailable) {
				return nil, err
			}
			return nil, fmt.Errorf("%w: XPath transform evaluation failed: %v", ErrUnsupportedTransform, err)
		}
		if r.Bool {
			kept = append(kept, n)
		}
	}
	return kept, nil
}

// removeSignatureNodes drops every node in the enveloped Signature's own subtree
// from a node-set, implementing the enveloped-signature transform on the
// explicit node-set used by the XPath-filter path (the non-XPath path omits the
// Signature via canonicalizeEnveloped's document clone instead).
func removeSignatureNodes(nodes []helium.Node, sigElem *helium.Element) []helium.Node {
	kept := make([]helium.Node, 0, len(nodes))
	for _, n := range nodes {
		if isDescendantOrSelf(n, sigElem) {
			continue
		}
		kept = append(kept, n)
	}
	return kept
}

// canonicalizeDetachedSubtree canonicalizes target, an element that lives inside
// the detached subtree rooted at root — a Signature that has not yet been placed
// in a document. It serves both a reference into an enveloping Signature's own
// <Object> content (URI="#id") and the Signature's own <SignedInfo> in detached
// and enveloping signing, so neither canonicalization ever inserts the Signature
// into the caller's document. The c14n canonicalizer walks from a document root,
// so a detached node set would canonicalize to nothing. We move the LIVE root
// into a private throwaway document for the duration of the canonicalization —
// never touching the caller's document — then move it back out and restore its
// owning document, leaving root detached exactly as it was. Using the live
// nodes (not a copy) keeps the bytes identical to what a verifier canonicalizing
// the same nodes in place would produce.
//
// The throwaway document is rooted at a proxy element that reproduces the FULL
// inherited canonicalization context target will have once the caller places the
// returned Signature under the caller document element, and root is placed under
// that proxy. Canonicalization of a node-set apex inherits two dimensions from
// its omitted ancestors, and the proxy stands in for the caller document element
// (target's nearest omitted element-ancestor once placed) by carrying both:
//
//   - Every in-scope namespace declaration. Inclusive Canonical XML emits every
//     in-scope namespace on the node-set apex, including ones inherited from
//     ancestors and not visibly used, so a bare-rooted throwaway document would
//     drop the caller root's namespaces and produce a digest no verifier can
//     match.
//   - The inherited xml:* attributes, copied per the C14N version so the set
//     matches EXACTLY what helium's own canonicalizer inherits to a node-set
//     apex (see copyInheritedXMLAttrs): Canonical XML 1.0 inherits every
//     xml:*-namespace attribute across an omitted-ancestor gap (including
//     xml:id), while Canonical XML 1.1 inherits only xml:lang/xml:space and
//     lexically joins xml:base (xml:id NOT inherited). Deriving the copied set
//     from the caller root's own xml:* attributes per version keeps sign-time
//     (proxy) and verify-time (real placement) inheritance identical by
//     construction, so no inherited xml:* dimension can be missed.
//
// The proxy is never part of the canonicalized node set (only target's own
// subtree is), so it changes the inherited context only, never the emitted bytes
// of the subtree itself. Exclusive Canonical XML emits only visibly-utilized
// namespaces and performs NO xml:* inheritance, so both an unused inherited
// namespace and any inherited xml:* on the proxy leave its output byte-identical.
func canonicalizeDetachedSubtree(method string, root, target *helium.Element, prefixes []string) ([]byte, error) {
	mode, _, err := resolveC14NMode(method)
	if err != nil {
		return nil, err
	}

	origDoc := root.OwnerDocument()
	tmp := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)

	proxy, err := tmp.CreateElement("proxy")
	if err != nil {
		return nil, err
	}
	if origDoc != nil {
		if docElem := origDoc.DocumentElement(); docElem != nil {
			for prefix, ns := range domutil.InScopeNamespaces(docElem, true) {
				if err := proxy.DeclareNamespace(prefix, ns.URI()); err != nil {
					return nil, err
				}
			}
			if err := copyInheritedXMLAttrs(proxy, docElem, mode); err != nil {
				return nil, err
			}
		}
	}
	if err := tmp.SetDocumentElement(proxy); err != nil {
		return nil, err
	}
	if err := proxy.AddChild(root); err != nil {
		return nil, err
	}

	// root is now grafted into the throwaway document. Restore it on EVERY exit —
	// normal return, error, AND a panic unwinding out of canonicalization: detach
	// root from the throwaway document and give the subtree back its original
	// owning document, leaving root detached exactly as the caller expects. The
	// library must always undo its own temporary mutation, even when a downstream
	// panic (whatever its cause) unwinds through it.
	defer func() {
		helium.UnlinkNode(root)
		root.SetTreeDoc(origDoc)
	}()

	// Propagate the throwaway document onto the whole subtree so canonicalizeSubtree,
	// which reaches the document via target.OwnerDocument(), can walk it.
	//
	// The deferred restore above is complete for any tree built through the guarded
	// APIs: SetTreeDoc's walk (helium setListDoc) can only panic mid-walk on a
	// typed-nil sibling pointer, and every guarded construction path (parser,
	// AddChild/AddSibling/Replace, Create*, SetDocumentElement) rejects nil and
	// typed-nil up front via isNilNode/ErrNilNode, so a well-formed subtree's
	// owner-change walk never panics part-way and origDoc is fully restored. A
	// typed-nil sibling is reachable only through the explicitly-unsafe
	// helium.UnsafeSet* family, whose contract states a misuse leaves the tree
	// inconsistent; a caller that corrupts the subtree that way owns the result, so
	// a partial restore after such a caller-corrupted tree is not a defect here.
	root.SetTreeDoc(tmp)

	return canonicalizeSubtree(method, target, prefixes)
}

// copyInheritedXMLAttrs copies the caller document element's inherited xml:*
// attributes onto the proxy so that, as target's nearest omitted
// element-ancestor, the proxy contributes exactly the xml:* values helium's own
// canonicalizer inherits to a node-set apex under the given C14N mode. The set is
// derived programmatically from the document element's xml-namespace attributes
// per the version rule (inheritedUnderMode) rather than a hardcoded name list, so
// an unusual or future xml:* attribute is never missed. The document element is
// the root, so its ancestor-or-self inherited context is exactly its own xml:*
// attributes. The xml namespace is predeclared and never emitted, so a fresh xml
// Namespace on each copied attribute is sufficient.
func copyInheritedXMLAttrs(proxy, docElem *helium.Element, mode c14n.Mode) error {
	for _, attr := range docElem.Attributes() {
		if attr.URI() != lexicon.NamespaceXML {
			continue
		}
		if !inheritedUnderMode(mode, attr.LocalName()) {
			continue
		}
		xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
		if err := proxy.SetAttributeNS(attr.LocalName(), attr.Value(), xmlNS); err != nil {
			return err
		}
	}
	return nil
}

// inheritedUnderMode reports whether an xml:<localName> attribute on an omitted
// ancestor is inherited to the node-set apex under the given C14N mode, matching
// helium's own canonicalizer (canonicalizer.go):
//
//   - Canonical XML 1.0 (inheritXMLAttrs10) inherits EVERY xml:*-namespace
//     attribute across an omitted-ancestor gap, including xml:id.
//   - Canonical XML 1.1 inherits only xml:lang and xml:space
//     (processSimpleInheritable11) and lexically joins xml:base
//     (processXMLBase11); xml:id is NOT inherited.
//   - Exclusive Canonical XML performs no xml:* inheritance, so nothing is copied
//     (its output is byte-identical regardless of the proxy's xml:* attributes).
func inheritedUnderMode(mode c14n.Mode, localName string) bool {
	switch mode {
	case c14n.C14N10:
		return true
	case c14n.C14N11:
		return localName == "base" || localName == "lang" || localName == "space"
	default:
		return false
	}
}

// isDescendantOrSelf reports whether n is root itself or lives inside root's
// subtree, walking n's ancestor chain by parent pointers.
func isDescendantOrSelf(n helium.Node, root *helium.Element) bool {
	for cur := n; cur != nil; cur = cur.Parent() {
		if e, ok := helium.AsNode[*helium.Element](cur); ok && e == root {
			return true
		}
	}
	return false
}

// canonicalizeEnveloped computes the canonical bytes for an enveloped
// signature reference WITHOUT mutating the caller's document. The
// enveloped-signature transform is defined as canonicalizing the reference
// content with the ds:Signature element and its descendants omitted; rather
// than unlinking the live Signature (which races with concurrent readers and
// risks leaving the caller's DOM corrupted if a restore fails), we deep-copy
// the document, remove the Signature from the copy, and canonicalize the copy.
//
// doc is the caller's (unmodified) document and sigElem is the live Signature
// element to omit. When wholeDoc is true the whole copied document is
// canonicalized (URI=""); otherwise the cloned subtree corresponding to the
// live target element is canonicalized (URI="#id"). The returned bytes are
// byte-identical to canonicalizing the same tree with the Signature physically
// detached.
func canonicalizeEnveloped(method string, doc *helium.Document, target, sigElem *helium.Element, wholeDoc bool, prefixes []string) ([]byte, error) {
	clone, err := helium.CopyDoc(doc)
	if err != nil {
		return nil, err
	}

	// Resolve the Signature's twin in the clone by replaying the child-index
	// path from the document down to the live Signature. CopyDoc preserves
	// child order, so the path is stable.
	//
	// If the Signature is not attached to the document (e.g. an enveloped
	// transform requested on a detached/enveloping signature that lives outside
	// the tree), there is nothing in the canonical input to omit, so we
	// canonicalize the copy unchanged — matching the pre-clone behavior where
	// unlinking a detached node was a no-op.
	var cloneSigMut helium.MutableNode
	if sigPath := childIndexPath(sigElem); sigPath != nil {
		cloneSig := nodeAtPath(clone, sigPath)
		mut, ok := cloneSig.(helium.MutableNode)
		if !ok {
			return nil, fmt.Errorf("xmldsig1: could not locate Signature element in canonicalization copy")
		}
		cloneSigMut = mut
	}

	// Resolve the cloned target BEFORE unlinking the cloned Signature. Both
	// paths are computed against the live (un-unlinked) tree, so they must be
	// applied to the clone while it still mirrors that structure. If we unlinked
	// first, a Signature that precedes the target as a sibling would shift the
	// target's child index and nodeAtPath would resolve the wrong subtree.
	var cloneTarget *helium.Element
	if !wholeDoc {
		targetPath := childIndexPath(target)
		if targetPath == nil {
			return nil, fmt.Errorf("xmldsig1: could not locate reference target for enveloped transform")
		}
		t, ok := helium.AsNode[*helium.Element](nodeAtPath(clone, targetPath))
		if !ok {
			return nil, fmt.Errorf("xmldsig1: reference target in canonicalization copy is not an element")
		}
		cloneTarget = t
	}

	// Now it is safe to unlink the cloned Signature: the cloneTarget pointer is
	// already held and survives the structural change.
	if cloneSigMut != nil {
		helium.UnlinkNode(cloneSigMut)
	}

	// Whole-document reference: canonicalize the entire copy.
	if wholeDoc {
		return canonicalize(method, clone, prefixes)
	}

	// Fragment reference: canonicalize the cloned subtree corresponding to the
	// live target element.
	return canonicalizeSubtree(method, cloneTarget, prefixes)
}

// childIndexPath returns the sequence of child indices that locate n starting
// from its document's children (index 0 = document's first child). It returns
// nil if n is not reachable from the document root. The path indexes every node
// type (text, comment, PI, element), so it survives a faithful deep copy that
// preserves child ordering.
func childIndexPath(n helium.Node) []int {
	var rev []int
	for cur := n; cur != nil; cur = cur.Parent() {
		if _, ok := helium.AsNode[*helium.Document](cur); ok {
			// Reached the document node: the accumulated indices form a valid
			// path. Reverse to root-to-node order.
			slices.Reverse(rev)
			return rev
		}
		parent := cur.Parent()
		if parent == nil {
			// Detached from the document before reaching it.
			return nil
		}
		idx := 0
		found := false
		for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
			if c == cur {
				found = true
				break
			}
			idx++
		}
		if !found {
			return nil
		}
		rev = append(rev, idx)
	}
	return nil
}

// nodeAtPath walks the child-index path produced by childIndexPath, starting
// from doc, and returns the node found there (or nil if the path does not
// resolve).
func nodeAtPath(doc *helium.Document, path []int) helium.Node {
	var cur helium.Node = doc
	for _, idx := range path {
		child := cur.FirstChild()
		for i := 0; i < idx && child != nil; i++ {
			child = child.NextSibling()
		}
		if child == nil {
			return nil
		}
		cur = child
	}
	return cur
}

func resolveC14NMode(method string) (c14n.Mode, bool, error) {
	switch method {
	case C14N10:
		return c14n.C14N10, false, nil
	case C14N10Comments:
		return c14n.C14N10, true, nil
	case ExcC14N10:
		return c14n.ExclusiveC14N10, false, nil
	case ExcC14N10Comments:
		return c14n.ExclusiveC14N10, true, nil
	case C14N11URI:
		return c14n.C14N11, false, nil
	case C14N11Comments:
		return c14n.C14N11, true, nil
	default:
		return 0, false, fmt.Errorf("%w: %s", ErrUnsupportedAlgorithm, method)
	}
}

// collectSubtreeNodes returns all nodes in the subtree rooted at n
// (including n itself) in document order.
//
// For each element it also emits one namespace node per in-scope namespace
// (walking ancestors so that bindings declared above the subtree root are
// carried in). The c14n package in node-set mode only renders namespaces that
// are explicitly present in the node set, so without these the canonical bytes
// of a namespace-qualified subtree would drop their xmlns declarations,
// producing non-W3C output that breaks signature interop.
func collectSubtreeNodes(n helium.Node) []helium.Node {
	var nodes []helium.Node
	var walk func(helium.Node)
	walk = func(cur helium.Node) {
		nodes = append(nodes, cur)
		if elem, ok := helium.AsNode[*helium.Element](cur); ok {
			// In-scope namespace axis. c14n keys namespace nodes by their
			// parent element, so each wrapper is parented to this element.
			for _, ns := range inScopeNamespaces(elem) {
				nodes = append(nodes, helium.NewNamespaceNodeWrapper(ns, elem))
			}
			for _, attr := range elem.Attributes() {
				nodes = append(nodes, attr)
			}
		}
		// Enumerate owned children via helium.Children, which stops at a
		// foreign-owned child (an entity reference's shared Entity node is owned
		// by the DTD, whose sibling pointers thread into the DTD declaration
		// list) and is cycle-safe. A raw FirstChild/NextSibling walk would spill
		// DTD declaration nodes into the c14n node set. This matches the c14n
		// canonicalizer itself, which enumerates element children and expands an
		// entity reference through helium.Children, so the node set holds only
		// the owned subtree.
		for child := range helium.Children(cur) {
			walk(child)
		}
	}
	walk(n)
	return nodes
}

// inScopeNamespaces returns the namespaces in scope for elem, walking from the
// document root down so that closer (inner) declarations override outer ones.
// The implicit xml namespace is excluded — C14N never declares it explicitly.
func inScopeNamespaces(elem *helium.Element) []*helium.Namespace {
	byPrefix := domutil.InScopeNamespaces(elem, true)
	result := make([]*helium.Namespace, 0, len(byPrefix))
	for _, ns := range byPrefix {
		result = append(result, ns)
	}
	return result
}

// referenceURIForm classifies a same-document Reference URI into the node-set
// forms XMLDSig core supports (§4.3.3.2-3), fail-closed. It reports:
//
//   - id: the bare id to resolve (empty for whole-document forms);
//   - wholeDoc: the reference selects the whole document root;
//   - includeComments: comment nodes are part of the selected node-set;
//   - ok: the URI is one we support at all (false → the caller fails closed).
//
// The supported same-document forms and their comment semantics are:
//
//   - URI=""                     → whole document, comments EXCLUDED.
//   - URI="#id"                  → the element with that id, comments EXCLUDED.
//   - URI="#xpointer(/)"         → whole document, comments INCLUDED.
//   - URI="#xpointer(id('id'))"  → the element with that id, comments INCLUDED.
//
// The bare-name ("#id") and empty ("") forms produce a node-set WITHOUT comment
// nodes; the two full-XPointer forms (the bare-names SHOULD-support set) produce
// a node-set WITH comment nodes. Every other URI — an external reference, or any
// other #xpointer(...) scheme — is unsupported and stays fail-closed so a
// verifier never silently digests bytes the signer did not intend.
func referenceURIForm(uri string) (string, bool, bool, bool) {
	if uri == "" {
		return "", true, false, true
	}
	if !strings.HasPrefix(uri, "#") {
		// External reference: not supported.
		return "", false, false, false
	}
	frag := uri[1:]
	if !strings.HasPrefix(frag, "xpointer(") {
		// Bare-name "#id" (no XPointer scheme). Any "#name" without a "(" is a
		// bare id; comments are excluded.
		if strings.ContainsAny(frag, "()") {
			return "", false, false, false
		}
		return frag, false, false, true
	}
	// Full XPointer form: #xpointer(<expr>). Only the two bare-names
	// SHOULD-support schemes are honored; both include comment nodes.
	if !strings.HasSuffix(frag, ")") {
		return "", false, false, false
	}
	expr := strings.TrimSpace(frag[len("xpointer(") : len(frag)-1])
	if expr == "/" {
		return "", true, true, true
	}
	if id, ok := parseXPointerID(expr); ok {
		return id, false, true, true
	}
	return "", false, false, false
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

// parseGeneralXPointer recognizes a general XPointer URI of the XPointer
// framework form: a "#" followed by zero or more xmlns(prefix=uri) scheme parts
// and then exactly one xpointer(<expr>) scheme part. It returns the prefix->URI
// overrides declared by the xmlns() parts, the (paren-unescaped) XPath expression
// from the xpointer() part, and whether the URI matched this shape at all. A URI
// that is not "#"-prefixed, carries an unsupported scheme (element(), xpath1(),
// ...), is malformed (unbalanced parens, an xmlns part without "="), places an
// xmlns() part after the xpointer() part (the framework grammar requires every
// xmlns() to precede xpointer()), or lacks an xpointer part does NOT match — the
// caller then keeps its existing fail-closed handling. The four fast-path forms
// handled by referenceURIForm never reach here, so they stay byte-identical.
func parseGeneralXPointer(uri string) (map[string]string, string, bool) {
	if !strings.HasPrefix(uri, "#") {
		return nil, "", false
	}
	rest := uri[1:]
	if rest == "" {
		return nil, "", false
	}
	overrides := make(map[string]string)
	var expr string
	var haveXPointer bool
	for len(rest) > 0 {
		scheme, data, remainder, ok := nextSchemePart(rest)
		if !ok {
			return nil, "", false
		}
		switch scheme {
		case "xmlns":
			// The XPointer framework grammar requires every xmlns() scheme part to
			// PRECEDE the xpointer() part. Reject an xmlns() that appears after
			// xpointer() rather than binding it out of order; the URI then stays
			// fail-closed (an external reference) exactly as any other unmatched
			// shape.
			if haveXPointer {
				return nil, "", false
			}
			prefix, ns, ok := parseXmlnsPart(data)
			if !ok {
				return nil, "", false
			}
			overrides[prefix] = ns
		case "xpointer":
			if haveXPointer {
				// Only a single xpointer() part is supported.
				return nil, "", false
			}
			haveXPointer = true
			expr = strings.TrimSpace(unescapeXPointerData(data))
		default:
			// Any other scheme (element(), xpath1(), ...) is unsupported.
			return nil, "", false
		}
		rest = remainder
	}
	if !haveXPointer || expr == "" {
		return nil, "", false
	}
	return overrides, expr, true
}

// nextSchemePart reads one "scheme(data)" pointer part from the front of s,
// respecting the XPointer framework's balanced-parenthesis and "^" escape rules
// inside the data, and returns the scheme name, the raw (still-escaped) data, and
// the remaining string after the closing ")". Leading whitespace between parts is
// skipped. It fails (ok=false) on a missing/empty scheme name, a scheme name
// carrying whitespace or parens, or unbalanced parentheses.
func nextSchemePart(s string) (string, string, string, bool) {
	s = strings.TrimLeft(s, " \t\r\n")
	open := strings.IndexByte(s, '(')
	if open <= 0 {
		return "", "", "", false
	}
	scheme := s[:open]
	if strings.ContainsAny(scheme, " \t\r\n()") {
		return "", "", "", false
	}
	depth := 0
	for i := open; i < len(s); i++ {
		c := s[i]
		// "^(", "^)", and "^^" are escapes: the caret and the next byte are data.
		if c == '^' && i+1 < len(s) {
			if n := s[i+1]; n == '(' || n == ')' || n == '^' {
				i++
				continue
			}
		}
		switch c {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return scheme, s[open+1 : i], s[i+1:], true
			}
		}
	}
	return "", "", "", false
}

// parseXmlnsPart splits an xmlns() scheme part's data "prefix=uri" into its
// prefix and namespace URI. A missing "=" or an empty prefix is malformed.
func parseXmlnsPart(data string) (string, string, bool) {
	rawPrefix, rawNS, ok := strings.Cut(data, "=")
	if !ok {
		return "", "", false
	}
	prefix := strings.TrimSpace(rawPrefix)
	if prefix == "" {
		return "", "", false
	}
	return prefix, strings.TrimSpace(rawNS), true
}

// unescapeXPointerData reverses the XPointer framework circumflex escaping in a
// scheme part's data: "^(" -> "(", "^)" -> ")", "^^" -> "^". A caret not followed
// by one of those is left as-is.
func unescapeXPointerData(data string) string {
	if !strings.ContainsRune(data, '^') {
		return data
	}
	var b strings.Builder
	b.Grow(len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == '^' && i+1 < len(data) {
			if n := data[i+1]; n == '(' || n == ')' || n == '^' {
				b.WriteByte(n)
				i++
				continue
			}
		}
		b.WriteByte(data[i])
	}
	return b.String()
}

// xpointerNamespaces builds the prefix->URI namespace context for a general
// XPointer expression: the document element's in-scope bindings, with the
// xmlns() overrides layered on top. The default (empty-prefix) binding is
// dropped — XPath 1.0 has no default element namespace, so an unprefixed name
// test matches only no-namespace nodes.
func xpointerNamespaces(doc *helium.Document, overrides map[string]string) map[string]string {
	ns := make(map[string]string)
	if root := doc.DocumentElement(); root != nil {
		for prefix, n := range domutil.InScopeNamespaces(root, true) {
			if prefix == "" {
				continue
			}
			ns[prefix] = n.URI()
		}
	}
	maps.Copy(ns, overrides)
	return ns
}

// singleElementApex enforces the XML Signature Wrapping defense for a general
// XPointer node-set: it must identify a SINGLE element apex. An empty node-set is
// ErrReferenceNotFound; a node-set carrying a non-element principal node, or more
// than one distinct element, is ErrAmbiguousReference. Only when exactly one
// element (and nothing else) is selected does the reference resolve — that single
// element is a proper subtree apex, which the caller feeds into the existing
// subtree canonicalization path.
func singleElementApex(nodes []helium.Node) (*helium.Element, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("%w: XPointer selected an empty node-set", ErrReferenceNotFound)
	}
	var apex *helium.Element
	for _, n := range nodes {
		e, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil, fmt.Errorf("%w: XPointer selected a non-element node", ErrAmbiguousReference)
		}
		if apex == nil {
			apex = e
			continue
		}
		if e != apex {
			return nil, fmt.Errorf("%w: XPointer selected %d distinct elements", ErrAmbiguousReference, 2)
		}
	}
	return apex, nil
}

// resolveGeneralXPointerTarget resolves a general XPointer expression to its
// single element apex.
//
// An id() selector NEVER reaches xpath1's built-in id(): the built-in resolves
// through Document.GetElementByID, whose ID table overwrites on collision so a
// duplicate id silently resolves to a single element (an XML Signature Wrapping
// bypass). Instead, an expression whose whole value is an id('X') selector — in
// ANY whitespace spelling (id('X'), id ('X'), id( "X" )) — resolves through the
// duplicate-detecting findElementsByIDUnder, and ANY other use of id() (a
// parenthesized or embedded id() call the selector parser cannot reduce to a
// single literal id) is rejected fail-closed rather than handed to the built-in.
// Every remaining expression is evaluated on the shared bounded evaluator with
// the merged namespace context, here() disabled (nil bearing node), and the
// single-apex constraint enforced on the result.
func resolveGeneralXPointerTarget(ctx context.Context, doc *helium.Document, overrides map[string]string, expr string) (*helium.Element, error) {
	if id, isIDCall, ok := parseXPointerIDSelector(expr); isIDCall {
		if !ok {
			return nil, fmt.Errorf("%w: unsupported XPointer id() selector %q", ErrReferenceNotFound, expr)
		}
		matches := findElementsByIDUnder(doc.DocumentElement(), id)
		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("%w: xpointer(id(%q))", ErrReferenceNotFound, id)
		case 1:
			return matches[0], nil
		default:
			return nil, fmt.Errorf("%w: xpointer id %q matched %d elements", ErrAmbiguousReference, id, len(matches))
		}
	}
	if expressionReferencesID(expr) {
		// id() appears somewhere other than as the whole-expression selector
		// handled above (a wrapping paren, a predicate, a path step). xpath1's
		// built-in id() cannot be trusted under duplicate ids, so fail closed.
		return nil, fmt.Errorf("%w: unsupported XPointer id() use %q", ErrReferenceNotFound, expr)
	}

	compiled, err := xpath1.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid XPointer expression %q: %v", ErrReferenceNotFound, expr, err)
	}
	eval := newDSigXPathEvaluator(xpointerNamespaces(doc, overrides), nil, defaultXPathOpLimit)
	nodes, err := eval.Find(ctx, compiled, doc.DocumentElement())
	if err != nil {
		// Preserve the here()-unavailable sentinel (a URI-borne XPointer has no
		// ds:XPath bearing node) as a matchable typed error rather than flattening
		// it into ErrReferenceNotFound. Both remain fail-closed.
		if errors.Is(err, ErrHereUnavailable) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: XPointer evaluation failed: %v", ErrReferenceNotFound, err)
	}
	return singleElementApex(nodes)
}

// parseXPointerIDSelector recognizes a whole-expression id() selector in any
// whitespace spelling — id('X'), id ("X"), id( 'X' ), with optional surrounding
// whitespace. isIDCall reports that the trimmed expression IS a top-level
// id(...) call; ok additionally reports that it cleanly reduces to a single
// quoted id literal, returned in the first result. A general-XPointer id()
// selector is ALWAYS routed through the duplicate-detecting findElementsByIDUnder
// (never xpath1's built-in id()), so an id() call that is not a clean single
// literal is reported as isIDCall && !ok for the caller to reject fail-closed.
func parseXPointerIDSelector(expr string) (string, bool, bool) {
	s := strings.TrimSpace(expr)
	if !strings.HasPrefix(s, "id") {
		return "", false, false
	}
	rest := strings.TrimLeft(s[len("id"):], " \t\r\n")
	if !strings.HasPrefix(rest, "(") {
		// "id" is a name prefix of something else (idref, identity(), ...), not an
		// id() call.
		return "", false, false
	}
	if !strings.HasSuffix(rest, ")") {
		// An id() call that does not close the whole expression: id('x')/foo,
		// id('x')[1]. It IS an id() call, but not a clean selector.
		return "", true, false
	}
	arg := strings.TrimSpace(rest[1 : len(rest)-1])
	if len(arg) < 2 {
		return "", true, false
	}
	q := arg[0]
	if (q != '\'' && q != '"') || arg[len(arg)-1] != q {
		return "", true, false
	}
	inner := arg[1 : len(arg)-1]
	// The id must not contain the quote character (no embedded quote / second
	// argument), keeping this a strict single-id match.
	if strings.IndexByte(inner, q) >= 0 {
		return "", true, false
	}
	return inner, true, true
}

// expressionReferencesID reports whether an XPath expression invokes the id()
// function anywhere outside a string literal — an id name token immediately
// followed (modulo whitespace) by "(". The general-XPointer resolver uses it to
// fail closed on any id() use it does not itself resolve through the
// duplicate-detecting findElementsByIDUnder, since xpath1's built-in id()
// (Document.GetElementByID) resolves a duplicate id to a single element.
func expressionReferencesID(expr string) bool {
	var quote byte
	for i := range len(expr) {
		c := expr[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if c != 'i' || i+1 >= len(expr) || expr[i+1] != 'd' {
			continue
		}
		if i > 0 && isXPathNameByte(expr[i-1]) {
			continue // tail of a longer name (grid, uuid, ...)
		}
		j := i + len("id")
		for j < len(expr) && (expr[j] == ' ' || expr[j] == '\t' || expr[j] == '\r' || expr[j] == '\n') {
			j++
		}
		if j < len(expr) && expr[j] == '(' {
			return true
		}
	}
	return false
}

// isXPathNameByte reports whether b can appear within an XPath name (NCName plus
// the ":" prefix separator), used to reject a false "id" match that is the tail
// of a longer name.
func isXPathNameByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '_' || b == '-' || b == '.' || b == ':':
		return true
	default:
		return false
	}
}

// effectiveC14NMethod adjusts a canonicalization method for the comment
// membership of the reference's node-set. A C14N WithComments algorithm only
// emits comment nodes that are present in the node-set, so when the reference
// form excludes comments (URI="" or a bare "#id") a WithComments method is
// downgraded to its plain variant — equivalently, no comment node is in the set
// to emit. When the reference form includes comments the method is unchanged (a
// plain method still emits none, which is correct). This is the single point
// where reference-form comment semantics meet the c14n stage.
func effectiveC14NMethod(method string, includeComments bool) string {
	if includeComments {
		return method
	}
	switch method {
	case C14N10Comments:
		return C14N10
	case ExcC14N10Comments:
		return ExcC14N10
	case C14N11Comments:
		return C14N11URI
	}
	return method
}

// resolveReference resolves a Reference URI to the target node.
// For a whole-document form (URI="" or "#xpointer(/)"), returns the document
// element. For an element form ("#id" or "#xpointer(id('id'))"), returns the
// unique element with that id, searched across the document tree and any
// extraRoots. An enveloping signature passes its own (detached) Signature
// element as an extra root so a reference into its own <Object> content resolves
// before the Signature is placed in a document. If more than one element matches
// the id — in either tree, or one in each — returns ErrAmbiguousReference. This
// is the primary defense against XML Signature Wrapping (XSW) attacks where an
// attacker injects a duplicate-ID element containing malicious content, and it
// also rejects an id that collides between the document and the Signature's own
// Object content. Any unsupported URI (an external reference, or an unrecognized
// #xpointer(...) scheme) stays fail-closed as ErrReferenceNotFound.
func resolveReference(doc *helium.Document, uri string, extraRoots ...helium.Node) (*helium.Element, error) {
	id, wholeDoc, _, ok := referenceURIForm(uri)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported reference URI: %s", ErrReferenceNotFound, uri)
	}
	if wholeDoc {
		return doc.DocumentElement(), nil
	}
	// Walk each tree once and collect every candidate. We accept matches from
	// any of: a DTD/schema-declared ID-typed attribute, xml:id, or the "id"
	// attribute token in the casings "Id", "ID", or "id". We refuse to resolve
	// the reference if more than one element matches.
	matches := findElementsByIDUnder(doc.DocumentElement(), id)
	for _, root := range extraRoots {
		matches = append(matches, findElementsByIDUnder(root, id)...)
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: %s", ErrReferenceNotFound, uri)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("%w: %s (matched %d elements)", ErrAmbiguousReference, uri, len(matches))
	}
}

// findElementsByIDUnder walks the subtree rooted at root (root included) and
// returns every element whose ID matches the given value. root may be nil, in
// which case it returns no matches. The walk is exhaustive — it never
// short-circuits — so that duplicate IDs are surfaced to the caller rather
// than silently masked. We do NOT consult Document.GetElementByID: its
// underlying ID table is keyed by ID value and Document.RegisterID
// overwrites on collision, which would hide the duplicate-xml:id case that
// XSW hardening relies on.
//
// An attribute is treated as an ID when it is any of:
//   - declared ID-typed by a DTD or schema (AType == enum.AttrID);
//   - xml:id (ID-typed by the W3C xml:id Recommendation);
//   - the "id" attribute token in the casings "Id", "ID", or "id".
//
// This name set is FROZEN: it recognizes the "id" identifier token in the
// three casings above plus xml:id, and MUST NOT grow to distinct convention
// tokens such as "wsu:Id" or "AssertionID". Those are not universal ID names
// — they are ID-typed only by their own schemas — so a document that relies
// on them must declare that typing (DTD/schema, or by marking the attribute
// AType == enum.AttrID) rather than have this heuristic guess.
func findElementsByIDUnder(root helium.Node, id string) []*helium.Element {
	var matches []*helium.Element
	var walk func(helium.Node)
	walk = func(n helium.Node) {
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return
		}
		for _, attr := range elem.Attributes() {
			name := attr.Name()
			isIDAttr := name == "Id" || name == "ID" || name == "id" || name == "xml:id" || attr.AType() == enum.AttrID
			if !isIDAttr {
				continue
			}
			// xs:ID derives from xs:NCName, which collapses whitespace;
			// match libxml2/helium normalization for xml:id.
			if strings.TrimSpace(attr.Value()) == id {
				matches = append(matches, elem)
				break
			}
		}
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			walk(child)
		}
	}
	walk(root)
	return matches
}
