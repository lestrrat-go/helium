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