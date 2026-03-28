package helium

import "github.com/lestrrat-go/helium/enum"

type Enumeration []string

// Attribute represents an XML attribute (libxml2: xmlAttr).
type Attribute struct {
	docnode
	atype       enum.AttributeType
	defaultAttr bool
	ns          *Namespace
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
	next := n.NextSibling()
	if next == nil {
		return nil
	}
	return next.(*Attribute) //nolint:forcetypeassert
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
// i.e. one supplied by the DTD rather than present in the source document.
func (n *Attribute) SetDefault(b bool) {
	n.defaultAttr = b
}

// IsDefault reports whether this attribute was supplied by the DTD as a
// default value rather than being explicitly specified in the source document.
func (n *Attribute) IsDefault() bool {
	return n.defaultAttr
}

// Value returns the attribute's text value as a string.
func (n Attribute) Value() string {
	return string(n.Content())
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
