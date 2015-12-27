package helium

func newElement(name string) *Element {
	e := Element{}
	e.name = name
	e.etype = ElementNode
	return &e
}

func (n *Element) SetAttribute(name, value string) {
	attr := newAttribute(name, value, nil)

	n.properties = append(n.properties, attr)
}

func (n Element) Attributes() []*Attribute {
	return n.properties
}

func (n *Element) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *Element) AddContent(b []byte) error {
	return addContent(n, b)
}

func (n *Element) Replace(cur Node) {
	replaceNode(n, cur)
}