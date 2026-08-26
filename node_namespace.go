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

// Prefix returns the namespace prefix, or an empty string for the default
// namespace.
func (n Namespace) Prefix() string {
	return n.prefix
}

// URI returns the namespace URI.
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
	// owner is the element the wrapped declaration is in scope on. It is held
	// here instead of in the docnode parent pointer because a wrapper is a
	// VIRTUAL node: the namespace axis mints one per in-scope prefix on every
	// step and drops it with the query, it is linked into no child list, and
	// nothing ever unlinks it. A parent pointer here would take a claim on the
	// owner (see setParent) that no release could ever match, and every step
	// over that element would add another, so the element would lose
	// wouldCreateCycle's fast exit for the rest of its life.
	//
	// The link is one-way and outside the tree: a wrapper is not a MutableNode,
	// so no insertion can name one as a parent, and no parent pointer names a
	// wrapper. It therefore never sits on an ancestor chain, and the insertion
	// cycle guard never walks this link.
	owner Node
}

// NewNamespaceNodeWrapper creates a Node that wraps a Namespace for XPath use.
func NewNamespaceNodeWrapper(ns *Namespace, owner Node) *NamespaceNodeWrapper {
	n := &NamespaceNodeWrapper{ns: ns, owner: owner}
	n.etype = NamespaceNode
	n.name = ns.Prefix()
	return n
}

// Parent returns the node the wrapper is linked under once it has been
// inserted into a tree, and the element it was minted for otherwise.
func (n *NamespaceNodeWrapper) Parent() Node {
	if n.parent != nil {
		return n.parent
	}
	return n.owner
}

// Content returns the wrapped namespace's URI as bytes.
func (n *NamespaceNodeWrapper) Content() []byte {
	return []byte(n.ns.URI())
}

// ClarkName returns the Clark notation "{uri}local" for a namespace URI and
// local name pair.
func ClarkName(uri, local string) string {
	return "{" + uri + "}" + local
}
