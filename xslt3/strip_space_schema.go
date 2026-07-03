package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

// schemaWSMode is the whitespace-stripping verdict a schema type annotation
// imposes on the whitespace-only text-node children of an element, per the
// XSLT 3.0 §4.4.2 rules for a schema-validated source tree. These verdicts take
// precedence over xsl:strip-space / xsl:preserve-space (they apply "regardless"
// of those declarations).
type schemaWSMode int

const (
	// schemaWSNeutral means the schema imposes no verdict (the element is
	// untyped, or its type is the ur-type / an empty content type), so the
	// ordinary xsl:strip-space / xsl:preserve-space / DTD rules decide.
	schemaWSNeutral schemaWSMode = iota
	// schemaWSStrip means the element has an element-only complex content type,
	// so its whitespace-only text-node children are stripped regardless of
	// xsl:preserve-space (whitespace between child elements is insignificant).
	schemaWSStrip
	// schemaWSPreserve means the whitespace is significant and must be kept
	// regardless of xsl:strip-space: the element has simple or mixed content (the
	// whitespace is part of its value), or one of its ancestors was validated
	// against a type that includes an assertion (the assertion may depend on the
	// whitespace, so a processor cannot safely remove it).
	schemaWSPreserve
)

// schemaWSClassifier resolves the schema-derived whitespace verdict for an
// element from the type annotations gathered during source validation. It is
// keyed on the ORIGINAL (validated) source nodes, which is exactly what both the
// strip-space copy (copyAndStrip, whose parent nodes are the originals) and the
// execution-time strip pass see.
type schemaWSClassifier struct {
	annotations map[helium.Node]string
	registry    *schemaRegistry
}

// newSchemaWSClassifier returns a classifier, or nil when no type annotations
// are available (so callers keep the annotation-free fast path untouched).
func newSchemaWSClassifier(annotations map[helium.Node]string, reg *schemaRegistry) *schemaWSClassifier {
	if len(annotations) == 0 {
		return nil
	}
	return &schemaWSClassifier{annotations: annotations, registry: reg}
}

// mode returns the schema whitespace verdict for parent's whitespace-only
// text-node children.
func (c *schemaWSClassifier) mode(parent *helium.Element) schemaWSMode {
	if c == nil || parent == nil {
		return schemaWSNeutral
	}
	// An ancestor validated against a type that includes an assertion preserves
	// all descendant whitespace (the assertion may reference it). This wins over
	// the element's own content-type verdict.
	for n := helium.Node(parent); n != nil; n = n.Parent() {
		e, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		if tn := c.annotations[e]; tn != "" && typeAnnotationHasAssertion(tn, c.registry) {
			return schemaWSPreserve
		}
	}
	tn := c.annotations[parent]
	if tn == "" {
		return schemaWSNeutral
	}
	switch schemaContentCategory(tn, c.registry) {
	case schemaCatElementOnly:
		return schemaWSStrip
	case schemaCatSignificant:
		return schemaWSPreserve
	default:
		return schemaWSNeutral
	}
}

type schemaContentCat int

const (
	schemaCatNeutral schemaContentCat = iota
	schemaCatElementOnly
	schemaCatSignificant // simple or mixed content: whitespace may be part of the value
)

// schemaContentCategory classifies a type annotation (in xsdTypeName format:
// "xs:local", "Q{ns}local", or "Q{}local") by whether its content type makes
// whitespace-only text significant, insignificant (element-only), or unknown.
func schemaContentCategory(typeName string, reg *schemaRegistry) schemaContentCat {
	if strings.HasPrefix(typeName, "xs:") {
		// A built-in xs: type is a simple/atomic type (whitespace is part of the
		// value) EXCEPT the ur-types and the "not validated" sentinel, which carry
		// no content-type verdict. Anonymous element-only types collapse to
		// xs:anyType under xsdTypeName, so xs:anyType stays neutral rather than
		// wrongly claiming element-only.
		switch typeName {
		case "xs:untyped", "xs:anyType", "xs:anySimpleType":
			return schemaCatNeutral
		default:
			return schemaCatSignificant
		}
	}
	if reg == nil {
		return schemaCatNeutral
	}
	td, _, ok := reg.LookupTypeDef(typeName)
	if !ok || td == nil {
		return schemaCatNeutral
	}
	switch td.ContentType {
	case xsd.ContentTypeElementOnly:
		return schemaCatElementOnly
	case xsd.ContentTypeSimple, xsd.ContentTypeMixed:
		return schemaCatSignificant
	default:
		// ContentTypeEmpty (or unclassified): no whitespace-only content to
		// preserve, and stripping is a no-op; leave it to the ordinary rules.
		return schemaCatNeutral
	}
}

// typeAnnotationHasAssertion reports whether the named type (or any type in its
// base chain) carries an XSD 1.1 xs:assert constraint.
func typeAnnotationHasAssertion(typeName string, reg *schemaRegistry) bool {
	if reg == nil || strings.HasPrefix(typeName, "xs:") {
		return false
	}
	td, _, ok := reg.LookupTypeDef(typeName)
	if !ok {
		return false
	}
	// Walk the base chain (bounded to guard against a malformed cyclic schema).
	for cur, depth := td, 0; cur != nil && depth < 128; cur, depth = cur.BaseType, depth+1 {
		if len(cur.Assertions) > 0 {
			return true
		}
	}
	return false
}

// sourceNeedsSchemaStrip reports whether the validated tree rooted at root
// contains any whitespace-only text node that the schema element-only rule would
// strip. It lets the transform skip building a strip copy for a schema-validated
// document that has no element-only whitespace to remove (and no xsl:strip-space
// rules), preserving the no-copy fast path.
func sourceNeedsSchemaStrip(root helium.Node, class *schemaWSClassifier) bool {
	if class == nil || root == nil {
		return false
	}
	stack := []helium.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			switch child.Type() {
			case helium.TextNode, helium.CDATASectionNode:
				parent, ok := child.Parent().(*helium.Element)
				if ok && isWhitespaceOnly(child.Content()) && class.mode(parent) == schemaWSStrip {
					return true
				}
			case helium.ElementNode:
				stack = append(stack, child)
			}
		}
	}
	return false
}

// isWhitespaceOnly reports whether content consists solely of XML whitespace.
func isWhitespaceOnly(content []byte) bool {
	for _, b := range content {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
}
