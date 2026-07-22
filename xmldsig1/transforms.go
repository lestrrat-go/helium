package xmldsig1

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/domutil"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlbase64"
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
	// stylesheet carries the XSLT transform's serialized xsl:stylesheet subtree
	// (from the ds:Transform element). It is populated only when
	// algorithm == TransformXSLT.
	stylesheet []byte
}

// xpathFilter is a resolved XPath filter transform: an XPath 1.0 boolean
// expression plus the namespace bindings it is evaluated under.
type xpathFilter struct {
	expr string
	ns   map[string]string
}

// transformPipeline is the resolved, algorithm-agnostic result of interpreting a
// Reference transform list: the effective canonicalization method and its
// inclusive-namespace prefixes, whether an enveloped-signature transform is
// present, any XPath filter transforms (in declared order) that run on the
// node-set before canonicalization, and whether the octet-producing step is the
// base64 decode transform rather than a canonicalization.
//
// base64 and c14nMethod are mutually exclusive: base64 ends the pipeline by
// decoding its input node-set's string-value to octets, so no canonicalization
// runs and c14nMethod stays empty in that case.
type transformPipeline struct {
	c14nMethod   string
	prefixes     []string
	hasEnveloped bool
	xpathFilters []xpathFilter
	base64       bool
	// xslt, when non-nil, is the serialized xsl:stylesheet subtree of an XSLT
	// transform. The XSLT transform is octet-in -> octet-out: it consumes the
	// pre-XSLT octet stream (the canonicalized node-set) and its output is
	// digested. It ends the pipeline, so no transform may follow it.
	xslt []byte
}

// transformSteps converts a ReferenceConfig's typed Transform list into the
// algorithm-agnostic steps consumed by resolveTransformPipeline, so the signing
// preflight and the per-reference digest path interpret a transform list
// identically.
func transformSteps(ref ReferenceConfig) []transformStep {
	steps := make([]transformStep, len(ref.Transforms))
	for i, t := range ref.Transforms {
		step := transformStep{algorithm: t.URI()}
		if exc, ok := t.(excC14NTransform); ok {
			step.prefixes = exc.prefixes
		}
		steps[i] = step
	}
	return steps
}

// preflightSignerTransforms validates every Reference's transform pipeline
// BEFORE any DOM mutation or node moves. Every sign entry point calls this
// first so that a rejected pipeline (an unsupported transform, or a transform
// ordered after canonicalization) returns its error without moving caller
// content into an <Object>, adding a Signature element, or otherwise mutating
// the input tree.
func preflightSignerTransforms(cfg *signerConfig) error {
	for i, ref := range cfg.references {
		pipe, err := resolveTransformPipeline(transformSteps(ref))
		if err != nil {
			// Carry the failing reference's index and URI so a caller signing
			// over a multi-reference configuration can pinpoint the offending
			// Reference, symmetric with the per-reference digest loop. The
			// underlying sentinel (e.g. ErrUnsupportedTransform) stays matchable
			// via errors.Is through ReferenceError.Unwrap.
			return &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: err}
		}
		// The base64 decode transform is verify-only: the signing digest path
		// (processReference) canonicalizes its reference node-set and has no
		// base64 branch, so honoring a base64 transform here would silently digest
		// the canonicalized subtree instead of the decoded octets — a fail-open
		// signature. Reject it fail-closed before any DOM mutation. There is no
		// typed Transform constructor for it, but a caller can implement the
		// exported Transform interface with this URI, so the guard is required.
		if pipe.base64 {
			return &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: fmt.Errorf("%w: base64 transform is not supported for signing", ErrUnsupportedTransform)}
		}
		// The XPath filter transform is verify-only: the signing digest path
		// (signReferenceOctets) reads only c14nMethod/prefixes/hasEnveloped from the
		// pipeline and never applies pipe.xpathFilters, and processReference writes
		// no <XPath> child, so honoring it would silently digest the unfiltered
		// node-set under a <Transform Algorithm="...xpath..."> with no expression — a
		// fail-open signature that no verifier reproduces. There is no typed
		// Transform constructor for it, but a caller can implement the exported
		// Transform interface with the TransformXPath URI, so reject it fail-closed
		// before any DOM mutation.
		if len(pipe.xpathFilters) > 0 {
			return &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: fmt.Errorf("%w: XPath filter transform is not supported for signing", ErrUnsupportedTransform)}
		}
		// The XSLT transform is verify-only, exactly like base64: the signing digest
		// path has no XSLT branch, so honoring an XSLT transform here would silently
		// digest the pre-XSLT octets instead of the transformed output — a fail-open
		// signature. There is no typed Transform constructor for it, but a caller can
		// implement the exported Transform interface with this URI, so the guard is
		// required. resolveTransformPipeline marks XSLT presence with a non-nil
		// pipe.xslt even when no stylesheet was captured.
		if pipe.xslt != nil {
			return &ReferenceError{Op: opSign, Reference: i, URI: ref.URI, Err: fmt.Errorf("%w: XSLT transform is not supported for signing", ErrUnsupportedTransform)}
		}
	}
	return nil
}

// resolveTransformPipeline interprets an ordered XMLDSig Reference transform
// list and returns the effective canonicalization method, its
// inclusive-namespace prefixes, and whether an enveloped-signature transform is
// present.
//
// A Reference's transform output begins as a node-set. The enveloped-signature
// transform maps a node-set to a node-set; a canonicalization (c14n) transform
// and the base64 decode transform each convert the node-set to an octet stream.
// helium supports no octet-stream-consuming transform, so once any transform has
// produced octets no further transform — a second c14n, a base64 after a c14n,
// or anything after a base64 — may run; such a list is rejected fail-closed
// rather than silently honoring only the last octet-producing transform.
//
// When no octet-producing transform is declared, the XMLDSig default
// node-set->octet conversion applies, which is inclusive Canonical XML 1.0 (NOT
// Exclusive C14N). The base64 transform ends the pipeline with decoded octets
// that are digested directly, so no default c14n is applied when it is present.
//
// The XPath filter transform (TransformXPath) maps a node-set to a node-set and
// so may appear before the octet-producing transform; each is recorded in order.
//
// The XSLT transform (TransformXSLT) is the one octet-in -> octet-out transform
// helium interprets: it consumes the pre-XSLT octet stream (the canonicalized
// node-set, filled in with inclusive C14N 1.0 when no c14n transform precedes it)
// and its output is digested. It ends the pipeline like an octet-producing
// transform, so nothing may follow it, and it may not itself follow one (an
// XSLT after a c14n/base64, or a second XSLT, is rejected). Any OTHER transform
// ordered after an octet-producing or octet-ending transform is likewise
// rejected.
//
// pipelineClosed records that an octet-ENDING transform (c14n, base64, or XSLT)
// has run, after which no further transform may appear. Whether octets already
// exist is recovered from p (a non-empty c14nMethod, base64, or xslt), so the
// C14N 1.0 fill-in below covers the "no octet-producing transform" case without a
// separate flag.
func resolveTransformPipeline(steps []transformStep) (transformPipeline, error) {
	var p transformPipeline
	pipelineClosed := false
	for _, t := range steps {
		if pipelineClosed {
			return transformPipeline{}, fmt.Errorf("%w: transform %s ordered after an octet-producing transform", ErrUnsupportedTransform, t.algorithm)
		}
		switch t.algorithm {
		case C14N10, C14N10Comments, ExcC14N10, ExcC14N10Comments, C14N11URI, C14N11Comments:
			p.c14nMethod = t.algorithm
			p.prefixes = t.prefixes
			pipelineClosed = true
		case TransformBase64:
			p.base64 = true
			pipelineClosed = true
		case TransformEnvelopedSignature:
			p.hasEnveloped = true
		case TransformXPath:
			p.xpathFilters = append(p.xpathFilters, xpathFilter{expr: t.xpathExpr, ns: t.xpathNS})
		case TransformXSLT:
			// XSLT consumes octets and produces octets; the pre-XSLT octets are the
			// canonicalized node-set (the default C14N 1.0 fill-in below supplies
			// them for a bare XSLT). It ends the pipeline. p.xslt marks XSLT presence
			// even when no stylesheet was captured (a sign-side caller-implemented
			// Transform with this URI has no stylesheet), so the sign preflight and
			// the verify path both detect it via a non-nil p.xslt; the empty marker
			// is never reached on the verify path, where parseXSLTTransform always
			// captures a non-empty stylesheet.
			p.xslt = t.stylesheet
			if p.xslt == nil {
				p.xslt = []byte{}
			}
			pipelineClosed = true
		default:
			return transformPipeline{}, fmt.Errorf("%w: %s", ErrUnsupportedTransform, t.algorithm)
		}
	}
	// The base64 transform digests its decoded octets directly, so a Reference
	// carrying it needs no canonicalization; only a Reference with no
	// octet-producing transform falls back to the inclusive C14N 1.0 default. An
	// XSLT transform still needs its pre-XSLT octets, so it takes this fill-in too.
	if p.c14nMethod == "" && !p.base64 {
		p.c14nMethod = C14N10
	}
	return p, nil
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

// base64TransformOctets implements the base64 decode transform (XMLDSig core
// §6.6.2) for a same-document node-set input. The transform's input is the
// resolved element's node-set; the spec converts a node-set input to octets by
// taking its XPath 1.0 string-value (equivalently, applying self::text() and
// concatenating), which is the element's concatenated descendant text with all
// element start/end tags, comments, and processing instructions stripped — so
// domutil.TextContent(target) is exactly that value. The concatenated text is
// base64-decoded (XML whitespace within the base64 is ignored by the decoder)
// and the decoded octets are digested directly, with no canonicalization after.
//
// Invalid base64 in the input is fail-closed as ErrInvalidSignature, matching how
// a malformed DigestValue/SignatureValue base64 is reported, so a Reference whose
// base64 content cannot be decoded never digests partial or unintended bytes.
func base64TransformOctets(target *helium.Element) ([]byte, error) {
	decoded, err := xmlbase64.DecodeString(domutil.TextContent(target))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 transform input: %v", ErrInvalidSignature, err)
	}
	return decoded, nil
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
// when an XPath filter transform must run on it. A comment-excluding form still
// drops comments at the c14n stage via effectiveC14NMethod.
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

// applyXPathFilter implements the XMLDSig XPath filter transform
// (http://www.w3.org/TR/1999/REC-xpath-19991116, core §6.6.3): the expression is
// evaluated once per input node with that node as the context node, under the
// transform's in-scope namespace bindings, and the node is kept when the result
// converts to boolean true. The expression is wrapped in fn:boolean so the XPath
// data-model boolean conversion (a non-empty node-set, a non-zero number, a
// non-empty string) governs membership. A malformed expression or an evaluation
// error is fail-closed as ErrUnsupportedTransform so a reference never digests
// an unfiltered node-set.
func applyXPathFilter(ctx context.Context, nodes []helium.Node, f xpathFilter) ([]helium.Node, error) {
	expr, err := xpath1.Compile("boolean(" + f.expr + ")")
	if err != nil {
		return nil, fmt.Errorf("%w: invalid XPath transform expression %q: %v", ErrUnsupportedTransform, f.expr, err)
	}
	eval := xpath1.NewEvaluator()
	if len(f.ns) > 0 {
		eval = eval.Namespaces(f.ns)
	}
	kept := make([]helium.Node, 0, len(nodes))
	for _, n := range nodes {
		r, err := eval.Evaluate(ctx, expr, n)
		if err != nil {
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
