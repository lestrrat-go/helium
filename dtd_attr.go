package helium

import "github.com/lestrrat-go/helium/enum"

// AttributeDecl is an xml attribute declaration from DTD.
type AttributeDecl struct {
	docnode
	atype    enum.AttributeType    // attribute type
	def      enum.AttributeDefault // default
	defvalue string                // ... or the default value
	tree     Enumeration           // ... or the enumeration tree, if any
	prefix   string                // the namespace prefix, if any
	elem     string                // name of the element holding the attribute
}

func newAttributeDecl() *AttributeDecl {
	attr := &AttributeDecl{}
	attr.etype = AttributeDeclNode
	return attr
}

func (n *AttributeDecl) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *AttributeDecl) AppendText(b []byte) error {
	return appendText(n, b)
}

func (n *AttributeDecl) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *AttributeDecl) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *AttributeDecl) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

// AType returns the attribute type (e.g. enum.AttrID, enum.AttrCDATA).
func (n *AttributeDecl) AType() enum.AttributeType {
	return n.atype
}

// Elem returns the element name this attribute declaration belongs to.
func (n *AttributeDecl) Elem() string {
	return n.elem
}

func lookupAttributeDecl(doc *Document, name, prefix, elem string) *AttributeDecl {
	if doc == nil {
		return nil
	}
	if dtd := doc.IntSubset(); dtd != nil {
		if decl, ok := dtd.LookupAttribute(name, prefix, elem); ok {
			return decl
		}
	}
	if dtd := doc.ExtSubset(); dtd != nil {
		if decl, ok := dtd.LookupAttribute(name, prefix, elem); ok {
			return decl
		}
	}
	return nil
}
