package xmldsig1

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/internal/xpath1/lexer"
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

// transformSnapshot is the immutable representation of a caller-defined
// transform. Signing observes only the transform URI, so retaining the
// caller's implementation would let later mutations change a configured
// Signer.
type transformSnapshot struct {
	uri string
}

func (t transformSnapshot) URI() string { return t.uri }

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

// cloneTransform returns a copy of t that shares no mutable state with t.
func cloneTransform(t Transform) Transform {
	switch t := t.(type) {
	case nil:
		return nil
	case excC14NTransform:
		return excC14NTransform{prefixes: slices.Clone(t.prefixes)}
	case envelopedTransform, c14nTransform:
		return t
	default:
		return transformSnapshot{uri: t.URI()}
	}
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
// the node-set of that subtree against its owning document. The node set goes
// straight to c14n, so it is built with the reduced, mode-aware namespace
// membership (see collectCanonicalizationNodes).
func canonicalizeSubtree(ctx context.Context, method string, elem *helium.Element, prefixes []string) ([]byte, error) {
	mode, comments, err := resolveC14NMode(method)
	if err != nil {
		return nil, err
	}
	nodes, err := collectCanonicalizationNodes(ctx, elem, mode)
	if err != nil {
		return nil, err
	}
	return canonicalizeNodeSetMode(mode, comments, nodes, elem.OwnerDocument(), prefixes)
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
	return canonicalizeNodeSetMode(mode, comments, nodes, doc, prefixes)
}

// canonicalizeNodeSetMode is the shared node-set -> octet call for a method URI
// whose c14n mode is already resolved, so a caller that needed the mode to build
// the node set does not resolve it twice.
func canonicalizeNodeSetMode(mode c14n.Mode, comments bool, nodes []helium.Node, doc *helium.Document, prefixes []string) ([]byte, error) {
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
func collectDocumentNodes(ctx context.Context, doc *helium.Document) ([]helium.Node, error) {
	var set nodeSet
	for c := range helium.Children(doc) {
		switch c.Type() {
		case helium.ElementNode:
			sub, err := collectSubtreeNodes(ctx, c)
			if err != nil {
				return nil, err
			}
			if err := set.addAll(ctx, sub); err != nil {
				return nil, err
			}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			if err := set.add(ctx, c); err != nil {
				return nil, err
			}
		}
	}
	return set.nodes, nil
}

// collectConvertedDocumentNodes builds the whole-document node-set for a
// document parsed at an octet boundary, sized to what the transform consuming it
// can observe. A canonicalization transform takes the set whole, so the reduced,
// mode-aware namespace membership yields identical octets while staying linear in
// the document (see collectCanonicalizationNodes); the XPath filter transform is
// evaluated once per node and may drop an interior element while keeping its
// descendants, so it keeps the complete axis.
//
// Those two are the only node-set consumers a parse can feed: validateTransformSteps
// rejects an enveloped-signature transform after an octet boundary, and base64
// consumes a node-set through its own text-only conversion.
func collectConvertedDocumentNodes(ctx context.Context, doc *helium.Document, consumerAlgorithm string) ([]helium.Node, error) {
	mode, _, err := resolveC14NMode(consumerAlgorithm)
	if err != nil {
		// Not one of the six canonicalization URIs, so the consumer is the XPath
		// filter and the set it is evaluated over must carry every namespace node.
		return collectDocumentNodes(ctx, doc)
	}
	return collectCanonicalizationDocumentNodes(ctx, doc, mode)
}

// collectCanonicalizationDocumentNodes is collectDocumentNodes with the document
// element's subtree built for canonicalization under mode, in place of the
// complete namespace axis. The document element is the set's apex and its whole
// subtree stays a member, which is what collectCanonicalizationNodes requires.
func collectCanonicalizationDocumentNodes(ctx context.Context, doc *helium.Document, mode c14n.Mode) ([]helium.Node, error) {
	var set nodeSet
	for c := range helium.Children(doc) {
		switch c.Type() {
		case helium.ElementNode:
			elem, ok := helium.AsNode[*helium.Element](c)
			if !ok {
				continue
			}
			sub, err := collectCanonicalizationNodes(ctx, elem, mode)
			if err != nil {
				return nil, err
			}
			if err := set.addAll(ctx, sub); err != nil {
				return nil, err
			}
		case helium.CommentNode, helium.ProcessingInstructionNode:
			if err := set.add(ctx, c); err != nil {
				return nil, err
			}
		}
	}
	return set.nodes, nil
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
// and fails closed with ErrHereUnavailable, resolving to no wrong node.
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

// maxErrorExpressionBytes bounds how much of an XPath filter expression an error
// message names. The expression is attacker-controlled and bounded only by
// maxXPathFilterExpressionBytes, so quoting a whole one would make the error
// string as large as the expression for whatever a caller logs. 128 bytes names
// every realistic filter in full — the W3C defCan-1 interop vector is 75
// characters — and identifies a longer one by its opening.
const maxErrorExpressionBytes = 128

// truncatedExpression returns as much of expr as an error message may name,
// marking a shortened one so a reader can tell a prefix from a whole
// expression.
func truncatedExpression(expr string) string {
	if len(expr) <= maxErrorExpressionBytes {
		return expr
	}
	// Cut on a rune boundary so a multi-byte character in a name test or a string
	// literal is not split into escaped bytes when the prefix is quoted.
	cut := maxErrorExpressionBytes
	for cut > 0 && !utf8.RuneStart(expr[cut]) {
		cut--
	}
	return expr[:cut] + "..."
}

// compileXPathFilterExpression compiles and statically validates the transform
// expression during the complete-list validation pass. Wrapping it in
// fn:boolean makes this compiled form identical to the one evaluated for each
// input node.
func compileXPathFilterExpression(expr string, eval xpath1.Evaluator) (*xpath1.Expression, error) {
	compiled, err := xpath1.Compile("boolean(" + expr + ")")
	if err != nil {
		return nil, fmt.Errorf("%w: invalid XPath transform expression %q: %v", ErrUnsupportedTransform, truncatedExpression(expr), err)
	}
	if err := eval.Validate(compiled); err != nil {
		return nil, fmt.Errorf("%w: invalid XPath transform expression %q: %v", ErrUnsupportedTransform, truncatedExpression(expr), err)
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
// ErrUnsupportedTransform while retaining the evaluator error, so a reference
// never digests an unfiltered node-set and callers can still detect cancellation.
func applyXPathFilter(ctx context.Context, nodes []helium.Node, f xpathFilter) ([]helium.Node, error) {
	eval := newDSigXPathEvaluator(f.ns, f.hereNode, defaultXPathOpLimit)
	kept := make([]helium.Node, 0, len(nodes))
	for _, n := range nodes {
		r, err := eval.Evaluate(ctx, f.expr, n)
		if err != nil {
			// Preserve the here()-unavailable sentinel as a matchable typed error,
			// flattening it into no ErrUnsupportedTransform string, so a
			// caller can tell "here() has no bearing node" from a generic malformed
			// transform. Both are fail-closed.
			if errors.Is(err, ErrHereUnavailable) {
				return nil, err
			}
			return nil, fmt.Errorf("%w: XPath transform evaluation failed: %w", ErrUnsupportedTransform, err)
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
//
// A dropped node is charged like a kept one: testing it walks its ancestor
// chain, and a set that lies entirely inside the Signature keeps nothing at all,
// so charging only what survives would read a whole node set without ever
// polling.
func removeSignatureNodes(ctx context.Context, nodes []helium.Node, sigElem *helium.Element) ([]helium.Node, error) {
	kept := nodeSet{nodes: make([]helium.Node, 0, len(nodes))}
	for _, n := range nodes {
		if isDescendantOrSelf(n, sigElem) {
			if err := kept.charge(ctx, 1); err != nil {
				return nil, err
			}
			continue
		}
		if err := kept.add(ctx, n); err != nil {
			return nil, err
		}
	}
	return kept.nodes, nil
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
func canonicalizeDetachedSubtree(ctx context.Context, method string, root, target *helium.Element, prefixes []string) ([]byte, error) {
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
	// typed-nil sibling is not reachable through helium's public API at all: the
	// raw link setters are package-private, and this module's own corrupt-tree test
	// fixtures reach them only through internal/nodelink. A caller that somehow
	// corrupts the subtree that way owns the result, so a partial restore after such
	// a corrupted tree is not a defect here.
	root.SetTreeDoc(tmp)

	return canonicalizeSubtree(ctx, method, target, prefixes)
}

// copyInheritedXMLAttrs copies the caller document element's inherited xml:*
// attributes onto the proxy so that, as target's nearest omitted
// element-ancestor, the proxy contributes exactly the xml:* values helium's own
// canonicalizer inherits to a node-set apex under the given C14N mode. The set is
// derived programmatically from the document element's xml-namespace attributes
// per the version rule (inheritedUnderMode), and from no hardcoded name list, so
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
func canonicalizeEnveloped(ctx context.Context, method string, doc *helium.Document, target, sigElem *helium.Element, wholeDoc bool, prefixes []string) ([]byte, error) {
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
	return canonicalizeSubtree(ctx, method, cloneTarget, prefixes)
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

// collectSubtreeNodes returns all nodes in the subtree rooted at n (including n
// itself) in document order, giving every element the COMPLETE XPath in-scope
// namespace axis.
//
// That axis costs one namespace node per (element x in-scope declaration), so it
// is quadratic in the document by construction. It is used only where the node
// set is an XPath filter transform's input: the filter is evaluated once per
// node — namespace nodes included — and it may drop an interior element while
// keeping that element's descendants, which makes every element's own axis
// observable. Narrowing the axis there would change which nodes the filter sees
// and which namespaces the survivors can still render, so the set stays complete
// and the walk polls ctx instead.
//
// A node set that goes straight to c14n unfiltered is built by
// collectCanonicalizationNodes, which is linear in the document.
func collectSubtreeNodes(ctx context.Context, n helium.Node) ([]helium.Node, error) {
	c := &subtreeCollector{fullAxis: true}
	if err := c.collect(ctx, n); err != nil {
		return nil, err
	}
	return c.nodes, nil
}

// collectCanonicalizationNodes returns the node set for canonicalizing the
// subtree rooted at elem under mode, in document order. Every element,
// attribute, and character node of the subtree is a member, exactly as in
// collectSubtreeNodes; only the namespace nodes are reduced, to the ones that
// can still change a byte of the canonical output. The result is therefore
// linear in the document — one namespace node per declaration actually written
// plus at most one per element — well under quadratic in (elements x ancestor
// declarations).
//
// It is valid ONLY for a set handed to c14n whole: every element of the subtree
// stays a member, so each element's nearest rendered ancestor is its own parent.
// Do not use it where an XPath filter may narrow the set (see
// collectSubtreeNodes).
//
// The two canonicalization families need DIFFERENT namespace sets, because they
// decide what to render from different inputs:
//
//   - Inclusive (Canonical XML 1.0 and 1.1) renders a namespace node when the
//     nearest rendered ancestor's node set does NOT already carry that prefix
//     with the same URI (c14n renderNamespacesNodeSet / nsRenderedByAncestor).
//     Membership is compared against the ANCESTOR'S SET, so a binding an element
//     inherits unchanged must be ABSENT from that element's set: keeping it
//     while the parent's set has dropped it would emit a redundant declaration.
//     What must stay is every binding the element CHANGES relative to its
//     parent — that is what inclusive C14N emits — plus the in-scope default
//     namespace, which the ancestor lookup also uses to decide whether to emit
//     an xmlns="" undeclaration.
//
//   - Exclusive C14N renders only the namespaces an element VISIBLY UTILIZES
//     (its own prefix, its attributes' prefixes) plus any prefix in an
//     InclusiveNamespaces PrefixList, and suppresses one an output ancestor
//     already rendered by consulting the rendered-namespace stack, never
//     the ancestor's node set (renderNamespacesExclusiveNodeSet). A visibly
//     utilized binding that is missing from the element's own set is not
//     rendered AT ALL, so applying the inclusive reduction here would emit
//     element and attribute names whose prefixes are never declared. Exclusive
//     therefore keeps the element's own prefix and every attribute prefix in
//     addition to the changed bindings.
//
// A PrefixList needs no per-element handling of its own: the subtree root
// carries the complete axis, so a listed prefix in scope there is rendered at
// the root and suppressed by the rendered-namespace stack at every descendant
// that does not rebind it — and a descendant that does rebind it has that
// binding as a change.
func collectCanonicalizationNodes(ctx context.Context, elem *helium.Element, mode c14n.Mode) ([]helium.Node, error) {
	c := &subtreeCollector{exclusive: mode == c14n.ExclusiveC14N10}
	if err := c.collect(ctx, elem); err != nil {
		return nil, err
	}
	return c.nodes, nil
}

// subtreeCollector walks a subtree once, tracking the in-scope namespace
// bindings incrementally so no element recomputes them from its ancestor chain.
// fullAxis selects the complete namespace axis on every element; otherwise the
// reduced, mode-aware membership described on collectCanonicalizationNodes
// applies, with exclusive naming the canonicalization family.
type subtreeCollector struct {
	nodeSet
	fullAxis  bool
	exclusive bool
	scope     map[string]*helium.Namespace
}

// nsBinding records the binding one element replaced for a prefix, so the walk
// can restore its parent's scope on the way out and can tell an inherited
// binding from one the element changed.
type nsBinding struct {
	prefix string
	prev   *helium.Namespace
	had    bool
}

// ctxPollInterval is how many units of work pass between context checks. The
// work these walks do is allocating and reading node-set members, so a cancelled
// context must be noticed inside a walk, not only at its end; polling every unit
// would charge a context read per member for no extra promptness.
const ctxPollInterval = 256

// ctxPoll counts work against the poll interval on behalf of one walk or one
// node set.
type ctxPoll struct {
	since int
}

// charge accounts n units of work against the poll interval and polls ctx when
// it elapses.
func (p *ctxPoll) charge(ctx context.Context, n int) error {
	p.since += n
	if p.since < ctxPollInterval {
		return nil
	}
	p.since = 0
	return ctx.Err()
}

// nodeSet is a node set under construction. add and addAll are the ONLY
// operations that grow it, and both charge what they added, so a site that grows
// a node set cannot leave that growth unpolled: there is no uncharged append to
// write. The members are what the cost is made of — one element can contribute
// as many namespace nodes as the document has bindings — so charging anything
// coarser, the tree nodes walked or the sets appended, would leave a whole
// element's or a whole subtree's worth of work inside one unpolled span.
type nodeSet struct {
	ctxPoll
	nodes []helium.Node
}

func (s *nodeSet) add(ctx context.Context, n helium.Node) error {
	s.nodes = append(s.nodes, n)
	return s.charge(ctx, 1)
}

// addAll appends nodes a poll interval at a time, so the span between two polls
// is bounded by that interval and not by the length of the slice handed in — the
// subtree collections that feed this method return the whole document's node set
// on a whole-document reference. The capacity for the entire slice is taken in
// one step first: chunked appends alone would re-grow the backing array
// geometrically, which costs about five times a single bulk append at a million
// members, while growing once leaves the copy within a percent of it at every
// size.
func (s *nodeSet) addAll(ctx context.Context, nodes []helium.Node) error {
	s.nodes = slices.Grow(s.nodes, len(nodes))
	for len(nodes) > 0 {
		n := min(len(nodes), ctxPollInterval)
		s.nodes = append(s.nodes, nodes[:n]...)
		if err := s.charge(ctx, n); err != nil {
			return err
		}
		nodes = nodes[n:]
	}
	return nil
}

func (c *subtreeCollector) collect(ctx context.Context, n helium.Node) error {
	// A context already cancelled when the walk starts stops it before any node
	// is collected; the periodic poll below covers one cancelled while it runs.
	if err := ctx.Err(); err != nil {
		return err
	}
	c.scope = make(map[string]*helium.Namespace)
	if elem, ok := helium.AsNode[*helium.Element](n); ok {
		// Seed the scope with the subtree root's own in-scope axis, so bindings
		// declared ABOVE the root are carried in. Every deeper element updates
		// this map incrementally instead of walking its ancestors again. The
		// seeding is charged like an emission: the root goes on to emit one
		// namespace node per binding seeded here, so the two are the same size.
		for prefix, ns := range domutil.InScopeNamespaces(elem, true) {
			c.scope[prefix] = ns
			if err := c.charge(ctx, 1); err != nil {
				return err
			}
		}
	}
	return c.walk(ctx, n, true)
}

func (c *subtreeCollector) walk(ctx context.Context, n helium.Node, root bool) error {
	if err := c.add(ctx, n); err != nil {
		return err
	}

	elem, isElem := helium.AsNode[*helium.Element](n)
	var changed []nsBinding
	if isElem {
		if !root {
			changed = c.enter(elem)
		}
		if err := c.appendNamespaceNodes(ctx, elem, root, changed); err != nil {
			return err
		}
		for _, attr := range elem.Attributes() {
			if err := c.add(ctx, attr); err != nil {
				return err
			}
		}
	}

	// Enumerate owned children via helium.Children, which stops at a
	// foreign-owned child (an entity reference's shared Entity node is owned by
	// the DTD, whose sibling pointers thread into the DTD declaration list) and
	// is cycle-safe. A raw FirstChild/NextSibling walk would spill DTD
	// declaration nodes into the c14n node set. This matches the c14n
	// canonicalizer itself, which enumerates element children and expands an
	// entity reference through helium.Children, so the node set holds only the
	// owned subtree.
	for child := range helium.Children(n) {
		if err := c.walk(ctx, child, false); err != nil {
			return err
		}
	}

	if isElem && !root {
		c.leave(changed)
	}
	return nil
}

// prefixIndexThreshold is how many prefixes a set must hold before it indexes
// them. Below it a linear scan beats a map and the allocation is pure loss, and
// the sets an ordinary element builds — the prefixes it changed, the prefixes it
// has emitted — never reach it.
const prefixIndexThreshold = 8

// prefixSet answers "does this element's set already hold this prefix" for one
// element. It owns both representations and switches between them ITSELF, on
// what it has actually been given: no caller states a capacity, so no caller can
// state one that a later call site outgrows. That is the whole point of the
// type. Membership is exact string equality either way, so which representation
// answers never changes an answer — only what the answer costs.
//
// A set is built per element and dropped with it, deliberately: a
// collector-wide map cleared between elements would keep the bucket array the
// largest element grew, so every later element would pay for that one — the
// document-wide cost the index exists to remove.
type prefixSet struct {
	list  []string
	index map[string]struct{}
}

// contains reports whether prefix is already in the set, scanning the ordered
// list until the set has grown large enough to index itself.
func (s *prefixSet) contains(prefix string) bool {
	if s.index != nil {
		_, ok := s.index[prefix]
		return ok
	}
	return slices.Contains(s.list, prefix)
}

// add puts prefix in the set, building the index the moment the set reaches the
// size at which scanning it stops being the cheaper answer.
func (s *prefixSet) add(prefix string) {
	s.list = append(s.list, prefix)
	if s.index != nil {
		s.index[prefix] = struct{}{}
		return
	}
	if len(s.list) < prefixIndexThreshold {
		return
	}
	s.index = make(map[string]struct{}, len(s.list))
	for _, p := range s.list {
		s.index[p] = struct{}{}
	}
}

// enter applies elem's namespace declarations to the running scope and returns
// the bindings it replaced, one entry per prefix touched. The rule matches
// domutil.InScopeNamespaces exactly — declarations first, then the element's own
// active namespace when nothing already binds its prefix — so the incremental
// scope equals the ancestor-chain walk that function performs.
func (c *subtreeCollector) enter(elem *helium.Element) []nsBinding {
	var seen prefixSet
	var changed []nsBinding
	for _, ns := range elem.Namespaces() {
		changed = c.bind(ns, changed, &seen)
	}
	if ns := elem.Namespace(); ns != nil {
		if _, ok := c.scope[ns.Prefix()]; !ok {
			changed = c.bind(ns, changed, &seen)
		}
	}
	return changed
}

// bind records the binding prefix had before ns replaced it, unless this element
// already recorded one for that prefix, and installs ns in the running scope.
// seen holds the prefixes changed already carries.
func (c *subtreeCollector) bind(ns *helium.Namespace, changed []nsBinding, seen *prefixSet) []nsBinding {
	prefix := ns.Prefix()
	if !seen.contains(prefix) {
		prev, had := c.scope[prefix]
		changed = append(changed, nsBinding{prefix: prefix, prev: prev, had: had})
		seen.add(prefix)
	}
	c.scope[prefix] = ns
	return changed
}

// leave restores the scope the element's parent had.
func (c *subtreeCollector) leave(changed []nsBinding) {
	for _, b := range changed {
		if b.had {
			c.scope[b.prefix] = b.prev
			continue
		}
		delete(c.scope, b.prefix)
	}
}

// appendNamespaceNodes emits elem's namespace nodes. c14n keys namespace nodes
// by their parent element, so each wrapper is parented to elem. The implicit xml
// namespace is never emitted — C14N never declares it explicitly.
//
// It stops and reports the context's error when a poll falls due inside the
// emission and the context is done. Nothing it has emitted is used then: the
// caller abandons the whole node set, so aborting part way through an element
// cannot change what a completed walk collects.
func (c *subtreeCollector) appendNamespaceNodes(ctx context.Context, elem *helium.Element, root bool, changed []nsBinding) error {
	// The subtree root has no rendered ancestor inside the node set, so it must
	// carry the whole axis: inclusive C14N emits every in-scope binding there,
	// and exclusive C14N resolves a PrefixList against it.
	if c.fullAxis || root {
		for prefix, ns := range c.scope {
			if prefix == lexicon.PrefixXML {
				continue
			}
			if err := c.add(ctx, helium.NewNamespaceNodeWrapper(ns, elem)); err != nil {
				return err
			}
		}
		return nil
	}

	// The candidates are the changed bindings, the default namespace, the
	// element's own prefix, and every attribute prefix. Nobody counts them: the
	// set sizes itself from what it is given, so an element whose candidates are
	// mostly attribute prefixes — which no count of changed bindings predicts —
	// indexes them exactly as an element that declared them would.
	var emitted prefixSet
	for _, b := range changed {
		ns := c.scope[b.prefix]
		if b.had && b.prev.URI() == ns.URI() {
			// A redeclaration of the binding already in scope changes nothing.
			continue
		}
		if err := c.emitNamespace(ctx, elem, b.prefix, &emitted); err != nil {
			return err
		}
	}

	// The in-scope default namespace stays on every element: the inclusive
	// ancestor lookup emits an xmlns="" undeclaration for an element whose set
	// has no default-namespace node while its nearest rendered ancestor's set
	// does, so dropping an unchanged default would undeclare it spuriously.
	if err := c.emitNamespace(ctx, elem, "", &emitted); err != nil {
		return err
	}

	if !c.exclusive {
		return nil
	}
	// Exclusive C14N renders what an element visibly utilizes, and renders
	// nothing for a prefix missing from the element's own set, so the element's
	// own prefix and its attributes' prefixes stay members even when inherited
	// unchanged.
	if ns := elem.Namespace(); ns != nil {
		if err := c.emitNamespace(ctx, elem, ns.Prefix(), &emitted); err != nil {
			return err
		}
	}
	for _, attr := range elem.Attributes() {
		if prefix := attr.Prefix(); prefix != "" {
			if err := c.emitNamespace(ctx, elem, prefix, &emitted); err != nil {
				return err
			}
		}
	}
	return nil
}

// emitNamespace appends the namespace node for prefix when that prefix is in
// scope and emitted does not already hold it, recording it there. emitted is
// carried by pointer and has nothing to rethread, so a call site cannot drop the
// record and make the element emit a duplicate namespace node — which would
// change the canonical octets. A prefix it does emit is charged against the poll
// interval, so it reports the context's error when that poll falls due on a done
// context.
func (c *subtreeCollector) emitNamespace(ctx context.Context, elem *helium.Element, prefix string, emitted *prefixSet) error {
	if prefix == lexicon.PrefixXML {
		return nil
	}
	ns, ok := c.scope[prefix]
	if !ok || emitted.contains(prefix) {
		return nil
	}
	emitted.add(prefix)
	return c.add(ctx, helium.NewNamespaceNodeWrapper(ns, elem))
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
	// XPath S contains only space, tab, carriage return, and line feed.
	expr := strings.Trim(frag[len("xpointer("):len(frag)-1], " \t\r\n")
	if expr == "/" {
		return "", true, true, true
	}
	if id, ok := parseXPointerID(expr); ok {
		return id, false, true, true
	}
	return "", false, false, false
}

// parseXPointerID matches the XPointer id() form id('X') or id("X"), allowing
// XML S before the opening parenthesis, and returns the quoted id. Anything else
// (a bare argument, a nested call, an unbalanced or mismatched quote) is rejected
// so only the two SHOULD-support schemes resolve.
func parseXPointerID(expr string) (string, bool) {
	if !strings.HasPrefix(expr, "id") {
		return "", false
	}
	rest := strings.TrimLeft(expr[len("id"):], " \t\r\n")
	if !strings.HasPrefix(rest, "(") || !strings.HasSuffix(rest, ")") {
		return "", false
	}
	// XPath S contains only space, tab, carriage return, and line feed.
	arg := strings.Trim(rest[1:len(rest)-1], " \t\r\n")
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
			// xpointer(), binding nothing out of order; the URI then stays
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
			// XPath S contains only space, tab, carriage return, and line feed.
			expr = strings.Trim(unescapeXPointerData(data), " \t\r\n")
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
// the remaining string after the closing ")". Leading XPath S whitespace between
// parts is skipped. It fails (ok=false) on a missing/empty scheme name, a scheme
// name carrying XPath S whitespace or parens, or unbalanced parentheses.
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
// prefix and namespace URI. A missing "=" or an invalid NCName prefix is malformed.
func parseXmlnsPart(data string) (string, string, bool) {
	rawPrefix, rawNS, ok := strings.Cut(data, "=")
	if !ok {
		return "", "", false
	}
	// XPointer's xmlns() scheme uses XML S, not all Unicode whitespace.
	prefix := strings.Trim(rawPrefix, " \t\r\n")
	if !xmlchar.IsValidNCName(prefix) {
		return "", "", false
	}
	return prefix, strings.Trim(rawNS, " \t\r\n"), true
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

type preparedGeneralXPointer struct {
	id         string
	idSelector bool
	compiled   *xpath1.Expression
	eval       xpath1.Evaluator
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

// prepareGeneralXPointer validates and compiles a general XPointer expression
// for later resolution to a single element apex.
//
// An id() selector NEVER reaches xpath1's built-in id(): the built-in resolves
// through Document.GetElementByID, whose ID table overwrites on collision so a
// duplicate id silently resolves to a single element (an XML Signature Wrapping
// bypass). Instead, an expression whose whole value is an id('X') selector — in
// ANY XPath S whitespace spelling (id('X'), id ('X'), id( "X" )) — resolves
// through the duplicate-detecting domutil.FindElementsByID. ANY other use of
// id() (a parenthesized or embedded id() call the selector parser cannot reduce
// to a single literal id) is rejected fail-closed and never reaches the built-in.
// Every remaining expression is statically validated with the merged namespace
// context, the shared operation limit, and here() disabled (nil bearing node).
func prepareGeneralXPointer(doc *helium.Document, overrides map[string]string, expr string) (*preparedGeneralXPointer, error) {
	if containsNonXMLSWhitespaceOutsideLiteral(expr) {
		return nil, fmt.Errorf("%w: invalid XPointer expression %q",
			ErrReferenceNotFound, lexer.DiagnosticExcerpt(expr))
	}
	if id, isIDCall, ok := parseXPointerIDSelector(expr); isIDCall {
		if !ok {
			return nil, fmt.Errorf("%w: unsupported XPointer id() selector %q",
				ErrReferenceNotFound, lexer.DiagnosticExcerpt(expr))
		}
		return &preparedGeneralXPointer{id: id, idSelector: true}, nil
	}
	if expressionReferencesID(expr) {
		// id() appears somewhere other than as the whole-expression selector
		// handled above (a wrapping paren, a predicate, a path step). xpath1's
		// built-in id() cannot be trusted under duplicate ids, so fail closed.
		return nil, fmt.Errorf("%w: unsupported XPointer id() use %q",
			ErrReferenceNotFound, lexer.DiagnosticExcerpt(expr))
	}

	compiled, err := xpath1.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid XPointer expression %q: %v",
			ErrReferenceNotFound, lexer.DiagnosticExcerpt(expr), err)
	}
	eval := newDSigXPathEvaluator(xpointerNamespaces(doc, overrides), nil, defaultXPathOpLimit)
	if err := eval.Validate(compiled); err != nil {
		return nil, fmt.Errorf("%w: invalid XPointer expression %q: %v",
			ErrReferenceNotFound, lexer.DiagnosticExcerpt(expr), err)
	}
	return &preparedGeneralXPointer{compiled: compiled, eval: eval}, nil
}

// containsNonXMLSWhitespaceOutsideLiteral reports whitespace that xpath1's
// lexer would accept even though XPath 1.0 permits only XML S between tokens.
// Whitespace inside a quoted string is data and remains untouched.
func containsNonXMLSWhitespaceOutsideLiteral(expr string) bool {
	var quote rune
	for _, r := range expr {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if unicode.IsSpace(r) && r != ' ' && r != '\t' && r != '\r' && r != '\n' {
			return true
		}
	}
	return false
}

func resolvePreparedGeneralXPointerTarget(ctx context.Context, doc *helium.Document, prepared *preparedGeneralXPointer) (*helium.Element, error) {
	if prepared.idSelector {
		matches := domutil.FindElementsByID(doc.DocumentElement(), prepared.id)
		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("%w: xpointer(id(%q))", ErrReferenceNotFound, prepared.id)
		case 1:
			return matches[0], nil
		default:
			return nil, fmt.Errorf("%w: xpointer id %q matched %d elements", ErrAmbiguousReference, prepared.id, len(matches))
		}
	}

	nodes, err := prepared.eval.Find(ctx, prepared.compiled, doc.DocumentElement())
	if err != nil {
		// Preserve the here()-unavailable sentinel (a URI-borne XPointer has no
		// ds:XPath bearing node) as a matchable typed error, flattening
		// it into no ErrReferenceNotFound. Both remain fail-closed.
		if errors.Is(err, ErrHereUnavailable) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: XPointer evaluation failed: %w", ErrReferenceNotFound, err)
	}
	return singleElementApex(nodes)
}

// parseXPointerIDSelector recognizes a whole-expression id() selector in any
// XPath S whitespace spelling — id('X'), id ("X"), id( 'X' ), with optional
// surrounding XPath S whitespace. isIDCall reports that the trimmed expression
// IS a top-level id(...) call; ok additionally reports that it cleanly reduces
// to a single quoted id literal, returned in the first result. A general-XPointer id()
// selector is ALWAYS routed through the duplicate-detecting domutil.FindElementsByID
// (never xpath1's built-in id()), so an id() call that is not a clean single
// literal is reported as isIDCall && !ok for the caller to reject fail-closed.
func parseXPointerIDSelector(expr string) (string, bool, bool) {
	// XPath S contains only space, tab, carriage return, and line feed.
	s := strings.Trim(expr, " \t\r\n")
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
	arg := strings.Trim(rest[1:len(rest)-1], " \t\r\n")
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
// followed (modulo XPath S whitespace) by "(". The general-XPointer resolver
// uses it to fail closed on any id() use it does not itself resolve through the
// duplicate-detecting domutil.FindElementsByID, since xpath1's built-in id()
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
// extraRoots. Overlapping search roots count each element once by pointer
// identity. An enveloping signature passes its own (detached) Signature element
// as an extra root so a reference into its own <Object> content resolves before
// the Signature is placed in a document. If more than one distinct element
// matches the id — in either tree, or one in each — returns
// ErrAmbiguousReference. This is the primary defense against XML Signature
// Wrapping (XSW) attacks where an attacker injects a duplicate-ID element
// containing malicious content, and it also rejects an id that collides between
// the document and the Signature's own Object content. Any unsupported URI (an
// external reference, or an unrecognized #xpointer(...) scheme) stays
// fail-closed as ErrReferenceNotFound.
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
	// the reference if more than one distinct element matches.
	var matches []*helium.Element
	seen := make(map[*helium.Element]struct{})
	collect := func(root helium.Node) {
		for _, match := range domutil.FindElementsByID(root, id) {
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			matches = append(matches, match)
		}
	}
	collect(doc.DocumentElement())
	for _, root := range extraRoots {
		collect(root)
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
