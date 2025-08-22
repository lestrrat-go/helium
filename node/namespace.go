package node

// Namespace represents an XML namespace declaration
type Namespace struct {
	*prefix
	href string
}

func NewNamespace(prefixStr, uri string) *Namespace {
	var p prefix
	ns := &Namespace{
		prefix: &p,
		href:   uri,
	}
	ns.SetPrefix(prefixStr)
	return ns
}

func (n *Namespace) URI() string {
	return n.href
}
