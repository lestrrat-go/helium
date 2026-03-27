package helium

// Namespace represents an XML namespace declaration (libxml2: xmlNs).
type Namespace struct {
	etype   ElementType
	href    string
	prefix  string
	context *Document
}

func newNamespace(prefix, uri string) *Namespace {
	n := Namespace{}
	n.prefix = prefix
	n.href = uri
	n.etype = NamespaceNode
	return &n
}

// NewNamespace creates a new Namespace with the given prefix and URI.
func NewNamespace(prefix, uri string) *Namespace {
	return newNamespace(prefix, uri)
}

func (n Namespace) Prefix() string {
	return n.prefix
}

func (n Namespace) URI() string {
	return n.href
}

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

// ClarkName returns the Clark notation "{uri}local" for a namespace URI and
// local name pair.
func ClarkName(uri, local string) string {
	return "{" + uri + "}" + local
}
