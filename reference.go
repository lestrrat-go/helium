package helium

// EntityRef represents an entity reference node in the DOM tree
// (libxml2: xmlNode with type XML_ENTITY_REF_NODE).
type EntityRef struct {
	node
}

func newEntityRef() *EntityRef {
	n := &EntityRef{}
	n.etype = EntityRefNode
	return n
}

func (e *EntityRef) AddChild(cur Node) error {
	return addChild(e, cur)
}

func (e *EntityRef) AddContent(b []byte) error {
	return addContent(e, b)
}

func (e *EntityRef) AddSibling(cur Node) error {
	return addSibling(e, cur)
}

func (e *EntityRef) Replace(cur Node) error {
	return replaceNode(e, cur)
}

func (n *EntityRef) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

