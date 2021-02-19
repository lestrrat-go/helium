// This file is auto-generated by internal/cmd/gennodes/main.go. DO NOT EDIT

package helium

type Entity struct {
	checked    int
	content    []byte
	doc        *Document
	entityType EntityType
	externalID string
	firstChild Node
	lastChild  Node
	name       string
	next       Node
	ns         *Namespace
	nsDefs     []*Namespace
	orig       string
	owner      bool
	parent     Node
	prev       Node
	private    interface{}
	properties *Attribute
	systemID   string
	uri        string
}

func (*Entity) Type() ElementType {
	return EntityNode
}

func (n *Entity) Content() []byte {
	return n.content
}

func (n *Entity) OwnerDocument() *Document {
	return n.doc
}

func (n *Entity) SetOwnerDocument(v *Document) {
	n.doc = v
}

func (n *Entity) EntityType() EntityType {
	return n.entityType
}

func (n *Entity) FirstChild() Node {
	return n.firstChild
}

func (n *Entity) setFirstChild(v Node) {
	n.firstChild = v
}

func (n *Entity) LastChild() Node {
	return n.lastChild
}

func (n *Entity) setLastChild(v Node) {
	n.lastChild = v
}

func (n *Entity) LocalName() string {
	return n.name
}

func (n *Entity) NextSibling() Node {
	return n.next
}

func (n *Entity) SetNextSibling(v Node) {
	n.next = v
}

func (n *Entity) Parent() Node {
	return n.parent
}

func (n *Entity) SetParent(v Node) {
	n.parent = v
}

func (n *Entity) PrevSibling() Node {
	return n.prev
}

func (n *Entity) SetPrevSibling(v Node) {
	n.prev = v
}

func (n *Entity) AddChild(c Node) error {
	return addChild(n, c)
}

func (n *Entity) AddContent(b []byte) error {
	return addContent(n, b)
}

func (n *Entity) AddSibling(c Node) error {
	return addSibling(n, c)
}

func (n *Entity) Replace(v Node) {
	replaceNode(n, v)
}