package helium

// NamespaceNodeWrapper wraps a Namespace to implement the Node interface
// for XPath namespace axis traversal. In XPath, namespace nodes appear as
// nodes with a name (the prefix), content (the URI), and a parent
// (the owning element).
type NamespaceNodeWrapper struct {
	docnode
	ns *Namespace
}

// NewNamespaceNodeWrapper creates a Node that wraps a Namespace for XPath use.
func NewNamespaceNodeWrapper(ns *Namespace, owner Node) *NamespaceNodeWrapper {
	n := &NamespaceNodeWrapper{ns: ns}
	n.etype = NamespaceNode
	n.name = ns.Prefix()
	n.parent = owner
	return n
}

func (n *NamespaceNodeWrapper) Content() []byte {
	return []byte(n.ns.URI())
}

func (n *NamespaceNodeWrapper) AddChild(Node) error   { return nil }
func (n *NamespaceNodeWrapper) AppendText([]byte) error { return nil }
func (n *NamespaceNodeWrapper) AddSibling(Node) error  { return nil }
func (n *NamespaceNodeWrapper) Replace(Node) error     { return nil }
func (n *NamespaceNodeWrapper) SetTreeDoc(*Document)   {}
