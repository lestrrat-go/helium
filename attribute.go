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
	return next.(*Attribute)
}

func (n *Attribute) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *Attribute) AppendText(b []byte) error {
	return appendText(n, b)
}

func (n *Attribute) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *Attribute) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

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

func (n *Attribute) SetDefault(b bool) {
	n.defaultAttr = b
}

func (n *Attribute) IsDefault() bool {
	return n.defaultAttr
}

func (n Attribute) Value() string {
	return string(n.Content())
}

func (n Attribute) Name() string {
	if n.ns != nil {
		if p := n.ns.Prefix(); p != "" {
			return p + ":" + n.docnode.Name()
		}
	}
	return n.docnode.Name()
}

func (n Attribute) Prefix() string {
	if n.ns == nil {
		return ""
	}
	return n.ns.Prefix()
}

func (n Attribute) URI() string {
	if n.ns == nil {
		return ""
	}
	return n.ns.URI()
}
