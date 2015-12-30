package helium

func newAttributeDecl() *AttributeDecl {
	attr := &AttributeDecl{}
	attr.etype = AttributeDeclNode
	return attr
}

func (n *AttributeDecl) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *AttributeDecl) AddContent(b []byte) error {
	return addContent(n, b)
}

func (n *AttributeDecl) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *AttributeDecl) Replace(cur Node) {
	replaceNode(n, cur)
}

func newAttribute(name string, ns *Namespace) *Attribute {
	attr := &Attribute{}
	attr.name = name
	attr.ns = ns
	return attr
}

func (n *Attribute) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *Attribute) AddContent(b []byte) error {
	return addContent(n, b)
}

func (n *Attribute) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *Attribute) Replace(cur Node) {
	replaceNode(n, cur)
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
