package helium

type AttributeType int

const (
	AttrInvalid AttributeType = iota
	AttrCDATA
	AttrID
	AttrIDRef
	AttrIDRefs
	AttrEntity
	AttrEntities
	AttrNmtoken
	AttrNmtokens
	AttrEnumeration
	AttrNotation
)

type AttributeDefault int

const (
	AttrDefaultInvalid AttributeDefault = iota
	AttrDefaultNone
	AttrDefaultRequired
	AttrDefaultImplied
	AttrDefaultFixed
)

type Enumeration []string

type Attribute struct {
	docnode
	// atype       AttributeType
	defaultAttr bool
	ns          *Namespace
}

func newAttributeDecl() *AttributeDecl {
	attr := &AttributeDecl{}
	attr.etype = AttributeDeclNode
	return attr
}

func (n *AttributeDecl) AddChild(cur Node) error {
	return addChild(n, cur)
}

func (n *AttributeDecl) AddContent(b []byte) error {
	return addContent(n, b)
}

func (n *AttributeDecl) AddSibling(cur Node) error {
	return addSibling(n, cur)
}

func (n *AttributeDecl) Replace(cur Node) {
	replaceNode(n, cur)
}

func (n *AttributeDecl) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

// AType returns the attribute type (e.g. AttrID, AttrCDATA).
func (n *AttributeDecl) AType() AttributeType {
	return n.atype
}

// Elem returns the element name this attribute declaration belongs to.
func (n *AttributeDecl) Elem() string {
	return n.elem
}

func newAttribute(name string, ns *Namespace) *Attribute {
	attr := &Attribute{}
	attr.name = name
	attr.ns = ns
	return attr
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

func (n *Attribute) Replace(cur Node) {
	replaceNode(n, cur)
}

func (n *Attribute) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}

func (n *Attribute) SetDefault(b bool) {
	n.defaultAttr = b
}

func (n *Attribute) IsDefault() bool {
	return n.defaultAttr
}

func (n Attribute) Value() string {
	return string(n.Content())
}

func (n Attribute) Name() string {
	if n.ns != nil {
		if p := n.ns.Prefix(); p != "" {
			return p + ":" + n.docnode.Name()
		}
	}
	return n.docnode.Name()
}

func (n Attribute) Prefix() string {
	if n.ns == nil {
		return ""
	}
	return n.ns.Prefix()
}

func (n Attribute) URI() string {
	if n.ns == nil {
		return ""
	}
	return n.ns.URI()
}
