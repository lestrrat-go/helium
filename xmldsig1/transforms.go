package xmldsig1

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/domutil"

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

// Prefixes returns the inclusive namespace prefixes for this transform.
func (t excC14NTransform) Prefixes() []string { return t.prefixes }

// ExcC14NTransform returns an Exclusive C14N transform with optional
// inclusive namespace prefixes.
func ExcC14NTransform(prefixes ...string) Transform {
	return excC14NTransform{prefixes: prefixes}
}

// transformStep is the algorithm-agnostic view of a single Reference transform,
// shared by the signing (typed Transform) and verification (parsedTransform)
// paths so both interpret a transform list identically.
type transformStep struct {
	algorithm string
	prefixes  []string
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
	for _, ref := range cfg.references {
		if _, _, _, err := resolveTransformPipeline(transformSteps(ref)); err != nil {
			return err
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
// converts the node-set to an octet stream. helium supports no
// octet-stream-consuming transform, so once a c14n transform has produced octets
// no further transform — including a second c14n — may run; such a list is
// rejected fail-closed rather than silently honoring only the last c14n.
//
// When no c14n transform is declared, the XMLDSig default node-set->octet
// conversion applies, which is inclusive Canonical XML 1.0 (NOT Exclusive C14N).
func resolveTransformPipeline(steps []transformStep) (string, []string, bool, error) {
	c14nMethod := ""
	var prefixes []string
	hasEnveloped := false
	producedOctets := false
	for _, t := range steps {
		if producedOctets {
			return "", nil, false, fmt.Errorf("%w: transform %s ordered after canonicalization", ErrUnsupportedTransform, t.algorithm)
		}
		switch t.algorithm {
		case C14N10, C14N10Comments, ExcC14N10, ExcC14N10Comments, C14N11URI, C14N11Comments:
			c14nMethod = t.algorithm
			prefixes = t.prefixes
			producedOctets = true
		case TransformEnvelopedSignature:
			hasEnveloped = true
		default:
			return "", nil, false, fmt.Errorf("%w: %s", ErrUnsupportedTransform, t.algorithm)
		}
	}
	if c14nMethod == "" {
		c14nMethod = C14N10
	}
	return c14nMethod, prefixes, hasEnveloped, nil
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

// canonicalizeSubtree canonicalizes a single element subtree. It creates
// a temporary document containing just the subtree for canonicalization.
func canonicalizeSubtree(method string, elem *helium.Element, prefixes []string) ([]byte, error) {
	mode, comments, err := resolveC14NMode(method)
	if err != nil {
		return nil, err
	}
	canon := c14n.NewCanonicalizer(mode).NodeSet(collectSubtreeNodes(elem))
	if comments {
		canon = canon.Comments()
	}
	if mode == c14n.ExclusiveC14N10 && len(prefixes) > 0 {
		canon = canon.InclusiveNamespaces(prefixes)
	}
	return canon.CanonicalizeTo(elem.OwnerDocument())
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
		for child := cur.FirstChild(); child != nil; child = child.NextSibling() {
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

// resolveReference resolves a Reference URI to the target node.
// For URI="" (enveloped), returns the document element.
// For URI="#id", returns the unique element with that ID. If more than one
// element matches the ID, returns ErrAmbiguousReference — this is the
// primary defense against XML Signature Wrapping (XSW) attacks where an
// attacker injects a duplicate-ID element containing malicious content.
func resolveReference(doc *helium.Document, uri string) (*helium.Element, error) {
	if uri == "" {
		return doc.DocumentElement(), nil
	}
	if strings.HasPrefix(uri, "#") {
		id := uri[1:]
		// Walk the tree once and collect every candidate. We accept matches
		// from any of: a DTD/schema-declared ID-typed attribute, xml:id, or
		// the "id" attribute token in the casings "Id", "ID", or "id". We
		// refuse to resolve the reference if more than one element matches.
		matches := findElementsByID(doc, id)
		switch len(matches) {
		case 0:
			return nil, fmt.Errorf("%w: %s", ErrReferenceNotFound, uri)
		case 1:
			return matches[0], nil
		default:
			return nil, fmt.Errorf("%w: %s (matched %d elements)", ErrAmbiguousReference, uri, len(matches))
		}
	}
	return nil, fmt.Errorf("%w: external references not supported: %s", ErrReferenceNotFound, uri)
}

// findElementsByID walks the entire document tree and returns every element
// whose ID matches the given value. The walk is exhaustive — it never
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
func findElementsByID(doc *helium.Document, id string) []*helium.Element {
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
	walk(doc.DocumentElement())
	return matches
}
