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

type ElementContentType int

const (
	ElementContentPCDATA ElementContentType = iota + 1
	ElementContentElement
	ElementContentSeq
	ElementContentOr
)

type ElementContentOccur int

const (
	ElementContentOnce ElementContentOccur = iota + 1
	ElementContentOpt
	ElementContentMult
	ElementContentPlus
)

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

func (n *ElementDecl) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *ElementDecl) AppendText(b []byte) error {
	return appendText(n, b)
}

func (n *ElementDecl) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *ElementDecl) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *ElementDecl) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}
