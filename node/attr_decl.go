package node

// AttributeDecl represents an attribute declaration
type AttributeDecl struct {
	treeNode
	*prefix
	etype        AttributeType
	def          AttributeDefault
	defaultValue string
	tree         *Enumeration
	elem         string
}

func NewAttributeDecl() *AttributeDecl {
	return &AttributeDecl{}
}

func (n *AttributeDecl) SetType(atype AttributeType) {
	n.etype = atype
}

func (n *AttributeDecl) SetName(name string) {
	n.name = name
}

func (n *AttributeDecl) SetElem(elem string) {
	n.elem = elem
}

func (n *AttributeDecl) SetDef(def AttributeDefault) {
	n.def = def
}

func (n *AttributeDecl) SetTree(tree *Enumeration) {
	n.tree = tree
}

func (n *AttributeDecl) SetDefaultValue(value string) {
	n.defaultValue = value
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

func (n *AttributeDecl) Replace(cur Node) error {
	return replaceNode(n, cur)
}

func (n *AttributeDecl) Type() NodeType {
	return AttributeDeclNodeType
}

func (n *AttributeDecl) LocalName() string {
	return n.elem
}

func (n *AttributeDecl) SetNextSibling(sibling Node) error {
	return setNextSibling(n, sibling)
}

func (n *AttributeDecl) SetPrevSibling(sibling Node) error {
	return setPrevSibling(n, sibling)
}
