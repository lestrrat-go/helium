package helium

import "github.com/lestrrat-go/helium/enum"

// ElementDecl is an xml element declaration from DTD.
type ElementDecl struct {
	docnode
	decltype   enum.ElementType
	content    *ElementContent
	attributes *AttributeDecl
	prefix     string
}

// DeclType returns the element content type declared in the DTD
// (e.g., ElementElementType for element-only content).
func (e *ElementDecl) DeclType() enum.ElementType {
	return e.decltype
}

// ElementContentType describes the kind of node in an [ElementContent] tree.
type ElementContentType int

const (
	// ElementContentPCDATA indicates a #PCDATA leaf.
	ElementContentPCDATA ElementContentType = iota + 1
	// ElementContentElement indicates a named element reference leaf.
	ElementContentElement
	// ElementContentSeq indicates a sequence (,) node with two children.
	ElementContentSeq
	// ElementContentOr indicates a choice (|) node with two children.
	ElementContentOr
)

// ElementContentOccur describes the occurrence constraint on an
// [ElementContent] node.
type ElementContentOccur int

const (
	// ElementContentOnce means the content must appear exactly once.
	ElementContentOnce ElementContentOccur = iota + 1
	// ElementContentOpt means the content is optional (?).
	ElementContentOpt
	// ElementContentMult means the content may appear zero or more times (*).
	ElementContentMult
	// ElementContentPlus means the content must appear one or more times (+).
	ElementContentPlus
)

// ElementContent represents the content model of a DTD element declaration
// as a binary tree. Leaf nodes are either #PCDATA or named element
// references; interior nodes are sequence (,) or choice (|) operators.
// Each node carries an occurrence constraint (once, ?, *, +).
//
// For example, the declaration <!ELEMENT doc (a, (b | c)+)> produces a
// sequence node at the root with element "a" as c1 and a choice node
// (b | c, occurrence +) as c2.
type ElementContent struct {
	ctype  ElementContentType
	coccur ElementContentOccur
	name   string
	prefix string
	c1     *ElementContent
	c2     *ElementContent
	parent *ElementContent
}

func newElementDecl() *ElementDecl {
	e := ElementDecl{}
	e.etype = ElementDeclNode
	return &e
}

// AddChild adds cur as a child of this element declaration node.
func (n *ElementDecl) AddChild(cur Node) error {
	return addChild(n, cur)
}

// AppendText appends the given bytes to this element declaration's
// text content.
func (n *ElementDecl) AppendText(b []byte) error {
	return appendText(n, b)
}

// AddSibling inserts cur as the next sibling of this element
// declaration in the DTD's declaration list.
func (n *ElementDecl) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

// Replace replaces this element declaration node with the given nodes.
func (n *ElementDecl) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

// SetTreeDoc recursively sets the owner document for this element
// declaration and all of its children.
func (n *ElementDecl) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}
