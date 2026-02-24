package helium

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

func (n Namespace) Prefix() string {
	return n.prefix
}

func (n Namespace) URI() string {
	return n.href
}
