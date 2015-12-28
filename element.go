package helium

func newElement(name string) *Element {
	e := Element{}
	e.name = name
	e.etype = ElementNode
	return &e
}

func (n *Element) SetAttribute(name, value string) error {
	attr := newAttribute(name, value, nil)

	if n.properties == nil {
		n.properties = attr
	} else {
		for p := n.properties; p != nil; {
			if p.Name() == name {
				return ErrDuplicateAttribute
			}
			if next := n.NextSibling(); next != nil {
				p = next.(*Attribute)
			} else {
				p = nil
			}
		}

		n.properties.AddChild(attr)
	}

	return nil
}

func (n Element) Attributes() []*Attribute {
	attrs := []*Attribute{}
	for attr := n.properties; attr != nil; {
		attrs = append(attrs, attr)
		if a := attr.NextSibling(); a != nil {
			attr = a.(*Attribute)
		} else {
			attr = nil
		}
	}

	return attrs
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