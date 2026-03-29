package xmldsig1

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/c14n"

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
// signing or verification, the Signature element is temporarily detached
// from the document before canonicalization.
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
func collectSubtreeNodes(n helium.Node) []helium.Node {
	var nodes []helium.Node
	var walk func(helium.Node)
	walk = func(cur helium.Node) {
		nodes = append(nodes, cur)
		// Include attribute nodes for elements.
		// Namespace nodes are handled internally by the c14n package.
		if elem, ok := cur.(*helium.Element); ok {
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

// resolveReference resolves a Reference URI to the target node(s).
// For URI="" (enveloped), returns the document element.
// For URI="#id", returns the element with that ID.
func resolveReference(doc *helium.Document, uri string) (*helium.Element, error) {
	if uri == "" {
		return doc.DocumentElement(), nil
	}
	if strings.HasPrefix(uri, "#") {
		id := uri[1:]
		// Try the standard GetElementByID first (DTD-declared + xml:id).
		elem := doc.GetElementByID(id)
		if elem != nil {
			return elem, nil
		}
		// Fallback: search for common Id/ID attributes used by XMLDSig/SAML.
		elem = findElementByIDAttr(doc, id)
		if elem != nil {
			return elem, nil
		}
		return nil, fmt.Errorf("%w: %s", ErrReferenceNotFound, uri)
	}
	return nil, fmt.Errorf("%w: external references not supported: %s", ErrReferenceNotFound, uri)
}

// findElementByIDAttr walks the document tree looking for an element with
// an "Id" or "ID" attribute matching the given value. This handles the common
// case where documents use Id attributes without DTD declarations (e.g., SAML).
func findElementByIDAttr(doc *helium.Document, id string) *helium.Element {
	var found *helium.Element
	var walk func(helium.Node)
	walk = func(n helium.Node) {
		if found != nil {
			return
		}
		elem, ok := n.(*helium.Element)
		if !ok {
			return
		}
		for _, attr := range elem.Attributes() {
			name := attr.Name()
			if (name == "Id" || name == "ID") && attr.Value() == id {
				found = elem
				return
			}
		}
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			walk(child)
		}
	}
	walk(doc.DocumentElement())
	return found
}
