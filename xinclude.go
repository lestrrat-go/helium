package helium

// XIncludeMarker is a marker node used to bracket XInclude-included content.
type XIncludeMarker struct {
	node
}

// NewXIncludeMarker creates an XInclude marker node (start or end).
func NewXIncludeMarker(doc *Document, etype ElementType, name string) *XIncludeMarker {
	n := &XIncludeMarker{}
	n.etype = etype
	n.name = name
	n.doc = doc
	return n
}

func (n *XIncludeMarker) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *XIncludeMarker) AppendText(b []byte) error {
	return appendText(n, b)
}

func (n *XIncludeMarker) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *XIncludeMarker) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *XIncludeMarker) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}
