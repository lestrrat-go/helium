package helium

func newAttribute(name, value string, ns *Namespace) *Attribute {
	attr := &Attribute{}
	attr.name = name
	attr.AddContent([]byte(value))
	attr.ns = ns
	return attr
}

func (n *Attribute) SetDefault(b bool) {
	n.defaultAttr = b
}

func (n *Attribute) IsDefault() bool {
	return n.defaultAttr
}

func (n Attribute) Value() string {
	return string(n.Content())
}

func (n Attribute) Prefix() string {
	return n.ns.Prefix()
}

func (n Attribute) URI() string {
	return n.ns.URI()
}
