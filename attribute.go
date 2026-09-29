package helium

import (
	"slices"
	"strings"

	"github.com/lestrrat-go/helium/enum"
)

// Enumeration is the list of allowed values recorded for a DTD attribute
// declared with an enumerated or NOTATION type (for example the tokens in
// <!ATTLIST e a (x|y|z) #IMPLIED>).
type Enumeration []string

// Attribute represents an XML attribute (libxml2: xmlAttr).
type Attribute struct {
	docnode
	atype       enum.AttributeType
	defaultAttr bool
	// syntheticBase is set only on the xml:base attribute the parser injects onto
	// the top-level elements of an external parsed entity (parser_entity_decl.go)
	// to record the entity's base URI. It is not present in the source, so DTD
	// validation exempts it from the "attribute must be declared" VC. An AUTHORED
	// xml:base is never marked and is validated normally.
	syntheticBase bool
	ns            *Namespace
}

func newAttribute(name string, ns *Namespace) *Attribute {
	attr := &Attribute{}
	attr.etype = AttributeNode
	attr.name = name
	attr.ns = ns
	return attr
}

// NextAttribute is a thin wrapper around NextSibling() so that the
// caller does not have to constantly type assert
func (n *Attribute) NextAttribute() *Attribute {
	attr, _ := AsNode[*Attribute](n.NextSibling())
	return attr
}

// AddChild adds cur as a child of this attribute node. For attributes
// the child is typically a text node holding the attribute value.
func (n *Attribute) AddChild(cur Node) error {
	return addChild(n, cur)
}

// AppendText appends the given bytes to the attribute's text content.
func (n *Attribute) AppendText(b []byte) error {
	return appendText(n, b)
}

// AddSibling inserts cur as the next sibling of this attribute,
// effectively appending another attribute to the owning element's
// attribute list.
func (n *Attribute) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

// Replace replaces this attribute node in its parent's attribute list
// with the given nodes.
func (n *Attribute) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

// SetTreeDoc recursively sets the owner document for this attribute
// and all of its children (e.g. its text-node value).
func (n *Attribute) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

// AType returns the attribute type (e.g. enum.AttrID, enum.AttrCDATA).
func (n *Attribute) AType() enum.AttributeType {
	return n.atype
}

// SetAType sets the attribute type.
func (n *Attribute) SetAType(v enum.AttributeType) {
	n.atype = v
}

// SetDefault marks (or unmarks) this attribute as a default attribute,
// i.e. one the DTD supplied and the source document never wrote.
func (n *Attribute) SetDefault(b bool) {
	n.defaultAttr = b
}

// IsDefault reports whether this attribute was supplied by the DTD as a
// default value, with no explicit specification in the source document.
func (n *Attribute) IsDefault() bool {
	return n.defaultAttr
}

// Value returns the attribute's value as a string, with every entity reference
// expanded (libxml2: xmlGetProp / xmlNodeGetContent on an attribute).
//
// An attribute whose value is a single Text node, the common case, converts
// that node's bytes directly with one allocation for the string. Any other
// shape, such as the Text/EntityRef list a SubstituteEntities(false) parse
// builds, is expanded by appendAttributeValue, so the result is the same
// string a SubstituteEntities(true) parse stores.
func (n *Attribute) Value() string {
	if n.firstChild == nil {
		return ""
	}
	if t, ok := n.firstChild.(*Text); ok && t.next == nil {
		return string(t.rawContent())
	}
	var b strings.Builder
	appendAttributeValue(&b, &n.docnode, nil)
	return b.String()
}

// appendAttributeValue appends the expanded value of owner's children to b,
// following libxml2's xmlBufGetChildContent: a Text or CDATA child contributes
// its text, an EntityRef child contributes its entity's expanded value, any
// other child contributes the value of its own children, and a child with no
// text (a comment or PI) contributes nothing.
//
// The walk follows only owner's own children (nextOwnedSibling) and stops on a
// cyclic sibling list. active holds the containers and entities being expanded
// on the current path; a node already on it is a reference cycle and is
// skipped. It plays the role of libxml2's XML_ENT_EXPANDING flag without
// writing to the shared Entity, so concurrent readers of one document do not
// race.
func appendAttributeValue(b *strings.Builder, owner *docnode, active []*docnode) {
	var g siblingCycleGuard
	for child := owner.firstChild; child != nil; {
		cdn := child.baseDocNode()
		if g.step(cdn) {
			return
		}
		switch c := child.(type) {
		case *Text:
			_, _ = b.Write(c.rawContent())
		case *CDATASection:
			_, _ = b.Write(c.rawContent())
		case *EntityRef:
			appendEntityRefValue(b, c, active)
		case *Comment, *ProcessingInstruction, *NamespaceNodeWrapper:
		default:
			if !slices.Contains(active, cdn) {
				appendAttributeValue(b, cdn, append(active, cdn))
			}
		}
		child = nextOwnedSibling(owner, cdn)
	}
}

// appendEntityRefValue appends the expanded value of the entity ref names
// (libxml2: xmlBufGetEntityRefContent). The entity is ref's Entity child when
// the reference is bound, otherwise the document's declaration of that name. A
// predefined entity contributes its character; any other entity contributes
// the expanded value of its parsed children. An entity whose replacement text
// was never parsed into children contributes that text as stored.
func appendEntityRefValue(b *strings.Builder, ref *EntityRef, active []*docnode) {
	ent, ok := ref.firstChild.(*Entity)
	if !ok {
		ent = lookupReferencedEntity(ref)
		if ent == nil {
			return
		}
	}
	if ent.entityType == enum.InternalPredefinedEntity {
		_, _ = b.WriteString(ent.content)
		return
	}
	edn := &ent.docnode
	if slices.Contains(active, edn) {
		return
	}
	if ent.firstChild == nil {
		_, _ = b.WriteString(ent.content)
		return
	}
	appendAttributeValue(b, edn, append(active, edn))
}

// lookupReferencedEntity resolves an unbound entity reference by name: a
// predefined entity first, then the owning document's declarations
// (libxml2: xmlGetDocEntity).
func lookupReferencedEntity(ref *EntityRef) *Entity {
	if ent, err := resolvePredefinedEntity(ref.name); err == nil {
		return ent
	}
	if ref.doc == nil {
		return nil
	}
	ent, ok := ref.doc.GetEntity(ref.name)
	if !ok {
		return nil
	}
	return ent
}

// Name returns the qualified (prefixed) name of the attribute.
// If the attribute belongs to a namespace with a non-empty prefix,
// the result is "prefix:localname"; otherwise it is just the local name.
func (n Attribute) Name() string {
	if n.ns != nil {
		if p := n.ns.Prefix(); p != "" {
			return p + ":" + n.docnode.Name()
		}
	}
	return n.docnode.Name()
}

// Prefix returns the namespace prefix of the attribute, or an empty
// string if the attribute is not in a namespace.
func (n Attribute) Prefix() string {
	if n.ns == nil {
		return ""
	}
	return n.ns.Prefix()
}

// URI returns the namespace URI of the attribute, or an empty string
// if the attribute is not in a namespace.
func (n Attribute) URI() string {
	if n.ns == nil {
		return ""
	}
	return n.ns.URI()
}
