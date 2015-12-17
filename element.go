package helium

func newElement(name string) *Element {
	e := Element{}
	e.name = name
	e.typ = ElementNode
	return &e
}

func (n *Element) AddChild(cur Node) error {
	switch cur.Type() {
	case TextNode:
		if lc := n.LastChild(); lc != nil && lc.Type() == TextNode {
			return lc.AddContent(cur.Content())
		}
	}

	return n.node.AddChild(cur)
}

func (n *Element) AddContent(b []byte) error {
	t := newText(b)
	return n.AddChild(t)
}

func (n *Element) SetAttribute(name, value string) {
	attr := Attribute{}
	attr.name = name
	attr.AddContent([]byte(value))

	n.attributes = append(n.attributes, attr)
}