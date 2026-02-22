package helium

// CDATASection represents a CDATA section node in the DOM tree.
// In libxml2 this is an xmlNode with type XML_CDATA_SECTION_NODE.
type CDATASection struct {
	node
}

func newCDATASection(b []byte) *CDATASection {
	c := CDATASection{}
	c.etype = CDATASectionNode
	c.content = make([]byte, len(b))
	copy(c.content, b)
	c.name = "(CDATA)"
	return &c
}

func (n *CDATASection) AddChild(cur Node) error {
	return ErrInvalidOperation
}

func (n *CDATASection) AddContent(b []byte) error {
	n.content = append(n.content, b...)
	return nil
}

func (n *CDATASection) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n CDATASection) Content() []byte {
	return n.content
}

func (n *CDATASection) Replace(cur Node) {
	replaceNode(n, cur)
}

func (n *CDATASection) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}
