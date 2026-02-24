package helium

// XIncludeNode is a marker node used to bracket XInclude-included content.
type XIncludeNode struct {
	node
}

// NewXIncludeNode creates an XInclude marker node (start or end).
func NewXIncludeNode(doc *Document, etype ElementType, name string) *XIncludeNode {
	n := &XIncludeNode{}
	n.etype = etype
	n.name = name
	n.doc = doc
	return n
}

func (n *XIncludeNode) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *XIncludeNode) AddContent(b []byte) error {
	return addContent(n, b)
}

func (n *XIncludeNode) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *XIncludeNode) Replace(cur Node) {
	replaceNode(n, cur)
}

func (n *XIncludeNode) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}
