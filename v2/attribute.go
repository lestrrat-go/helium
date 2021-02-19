package helium

func newAttribute(name string, ns *Namespace) *Attribute {
	var attr Attribute
	attr.name = name
	attr.ns = ns
	return &attr
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

func (n *Attribute) Prefix() string {
	return n.ns.Prefix()
}

func (n *Attribute) URI() string {
	return n.ns.URI()
}
