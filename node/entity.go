package node

// EntityRef represents an entity reference
type EntityRef struct {
	treeNode
}

// Entity represents an XML entity
type Entity struct {
	treeNode
	etype      EntityType
	orig       string
	content    string
	externalID string
	systemID   string
	URI        string
	//checked    int
	checked bool
}

// AttributeType represents the type of an attribute
type AttributeType int

const (
	AttrInvalid AttributeType = iota
	AttrCDATA
	AttrID
	AttrIDRef
	AttrIDRefs
	AttrEntity
	AttrEntities
	AttrNMToken
	AttrNMTokens
	AttrEnumeration
	AttrNotation
)

// AttributeDefault represents the default declaration of an attribute
type AttributeDefault int

const (
	AttrDefaultNone AttributeDefault = iota
	AttrDefaultRequired
	AttrDefaultImplied
	AttrDefaultFixed
)

// Enumeration represents a list of possible values
type Enumeration []string

// ElementDecl represents an element declaration
type ElementDecl struct {
	treeNode
}

// ElementTypeVal represents the type of element content
type ElementTypeVal int

const (
	UndefinedElementType ElementTypeVal = iota
	EmptyElementType
	AnyElementType
	MixedElementType
	ElementElementType
)

// ElementContentType represents the type of element content
type ElementContentType int

const (
	UndefinedElementContent ElementContentType = iota
	PCDATAElementContent
	ElementElementContent
	SeqElementContent
	OrElementContent
)

// ElementContentOccur represents content occurrence
type ElementContentOccur int

const (
	OnceElementContent ElementContentOccur = iota
	OptElementContent
	MultElementContent
	PlusElementContent
)

// ElementContent represents element content specification
type ElementContent struct {
	ctype  ElementContentType
	occur  ElementContentOccur
	name   string
	c1     *ElementContent
	c2     *ElementContent
	parent *ElementContent
	prefix string
}

// EntityType represents the type of entity
type EntityType int

const (
	InternalGeneralEntity EntityType = iota + 1
	ExternalGeneralParsedEntity
	ExternalGeneralUnparsedEntity
	InternalParameterEntity
	ExternalParameterEntity
	InternalPredefinedEntity
)

// NamespaceContainer interface for nodes that can contain namespaces
type NamespaceContainer interface {
	Namespaces() []*Namespace
}

// Namespacer interface for nodes that have namespace support
type Namespacer interface {
	Namespace() *Namespace
	Prefix() string
	LocalName() string
	SetNamespace(*Namespace)
}

func newEntity(name string, typ EntityType, publicID, systemID, content, orig string) *Entity {
	e := &Entity{
		etype:      typ,
		orig:       orig,
		content:    content,
		externalID: publicID,
		systemID:   systemID,
	}
	e.name = name
	return e
}

func (e *Entity) EntityType() EntityType {
	return e.etype
}

func (e *Entity) Checked() bool {
	return e.checked
}

func (e *Entity) MarkChecked() {
	e.checked = true
}

func (e *Entity) Type() NodeType {
	return EntityNodeType
}

func (e *Entity) LocalName() string {
	return e.name
}

func (e *Entity) AddChild(cur Node) error {
	return addChild(e, cur)
}

func (e *Entity) AddContent(b []byte) error {
	return addContent(e, b)
}

func (e *Entity) AddSibling(cur Node) error {
	return addSibling(e, cur)
}

func (e *Entity) Replace(cur Node) error {
	return replaceNode(e, cur)
}

func (e *Entity) SetOwnerDocument(doc *Document) error {
	e.doc = doc
	return nil
}

func (e *Entity) SetNextSibling(sibling Node) error {
	return setNextSibling(e, sibling)
}

func (e *Entity) SetPrevSibling(sibling Node) error {
	return setPrevSibling(e, sibling)
}

func (e *Entity) SetOrig(orig string) {
	e.orig = orig
}

func (e *Entity) ExternalID() string {
	return e.externalID
}

func (e *Entity) SystemID() string {
	return e.systemID
}
