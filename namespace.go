package helium

func NewNamespace() *Namespace {
	n := Namespace{}
	n.typ = NamespaceNode
	return &n
}

func (n *Namespace) AddChild(cur Node) error {
	return n.node.AddChild(n)
}

func (n *Namespace) AddContent(_ []byte) error {
	return ErrInvalidOperation
}