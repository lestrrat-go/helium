package node

type Attribute struct {
	treeNode
	name        string
	ns          *Namespace
	defaultAttr bool
}

var _ Node = (*Attribute)(nil)

func newAttribute(name string, ns *Namespace) *Attribute {
	return &Attribute{
		name: name,
		ns:   ns,
	}
}

func (Attribute) Type() NodeType {
	return AttributeNodeType
}

func (n *Attribute) Name() string {
	if n.ns == nil {
		return n.name
	}
	return n.ns.Prefix() + ":" + n.name
}

func (n *Attribute) LocalName() string {
	return n.name
}

// NextAttribute is a thin wrapper around NextSibling() so that the
// caller does not have to constantly type assert
func (n *Attribute) NextAttribute() *Attribute {
	next := n.NextSibling()
	if next == nil {
		return nil
	}
	return next.(*Attribute)
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

func (n *Attribute) Replace(cur Node) error {
	return replaceNode(n, cur)
}

func (e *Attribute) SetNextSibling(sibling Node) error {
	return setNextSibling(e, sibling)
}

func (e *Attribute) SetPrevSibling(sibling Node) error {
	return setPrevSibling(e, sibling)
}

func (n *Attribute) SetDefault(b bool) {
	n.defaultAttr = b
}

func (n *Attribute) IsDefault() bool {
	return n.defaultAttr
}

func (n *Attribute) Value() string {
	content, err := n.Content(nil)
	if err != nil {
		return ""
	}
	return string(content)
}

func (n *Attribute) Prefix() string {
	if n.ns == nil {
		return ""
	}
	return n.ns.Prefix()
}

func (n *Attribute) URI() string {
	if n.ns == nil {
		return ""
	}
	return n.ns.URI()
}
