package helium

import "errors"

var (
	ErrNilNode          = errors.New("nil node")
	ErrInvalidOperation = errors.New("operation cannot be performed")
)

type ErrUnimplemented struct {
	target string
}

func (e ErrUnimplemented) Error() string {
	return "unimplemented method: '" + e.target + "'"
}

type nilNode struct{}

// common data
type node struct {
	typ     NodeType
	private interface{}
	etype    ElementType
	name     string
	children []Node
	parent   Node
	next     Node
	prev     Node
	doc      *Document
}

type DocumentStandaloneType int
const (
	StandaloneInvalidValue = -99
	StandaloneExplicitYes = 1
	StandaloneExplicitNo  = 0
	StandaloneNoXMLDecl   = -1
	StandaloneImplicitNo  = -2
)
type Document struct {
	node
	version    string
	encoding   string
	standalone DocumentStandaloneType

	intSubset *DTD
	extSubset *DTD
}

type ProcessingInstruction struct {
	node
	target string
	data   string
}

type DTD struct {
	node
	elements map[string]Element
	entities map[string]Entity
	pentities map[string]Entity
	externalID string
	systemID string
}

type Namespace struct {
	node
}

type Attribute struct {
	node
	elem *Node
}

type NodeType int

const (
	ElementNode NodeType = iota + 1
	ProcessingInstructionNode
	NamespaceNode
	TextNode
	CommentNode
)

// helium.Node
type Node interface {
	AddChild(Node) error
	AddContent([]byte) error
	AddSibling(Node) error
	Content() []byte
	FirstChild() Node
	LastChild() Node
	NextSibling() Node
	PrevSibling() Node
	SetNextSibling(Node)
	SetPrevSibling(Node)
	Type() NodeType
}

type Text struct {
	node
	content []byte
}

type Comment struct {
	node
	content []byte
}

type ElementType int

const (
	UndefinedElementType ElementType = iota
	EmptyElementType
	AnyElementType
	MixedElementType
	ElementElementType
)

type Element struct {
	node
	content    string // XXX probably wrong
	attributes []Attribute
	prefix     string
}

type ElementContentType int

const (
	ElementContentPCDATA ElementContentType = iota + 1
	ElementContentElement
	ElementContentSeq
	ElementContentOr
)

type ElementContentOccur int

const (
	ElementContentOnce ElementContentOccur = iota + 1
	ElementContentOpt
	ElementContentMult
	ElementContentPlus
)

type ElementContent struct {
	// XXX no doc?
	ctype  ElementContentType
	coccur ElementContentOccur
	name   string
	prefix string
	c1     *ElementContent
	c2     *ElementContent
	parent *ElementContent
}

type EntityType int

const (
	InternalGeneralEntity EntityType = iota + 1
	ExternalGeneralParsedEntity
	ExternalGeneralUnparsedEntity
	InternalParameterEntity
	ExternalParameterEntity
	InternalPredefinedEntity
)

type Entity struct {
	node
	orig       string     // content without substitution
	content    string     // content or ndata if unparsed
	entityType EntityType // the entity type
	externalID string     // external identifier for PUBLIC
	systemID   string     // URI for a SYSTEM or PUBLIC entity
	uri        string     // the full URI as computed
	owner      bool       // does the entity own children
}

var (
	EntityLT = Entity{
		node: node{
			name: "lt",
		},
		orig:"<",
		content: "<",
		entityType: InternalPredefinedEntity,
		owner: false,
	}
	EntityApostrophe = Entity{
		node: node{
			name: "apos",
		},
		orig:"'",
		content: "'",
		entityType: InternalPredefinedEntity,
		owner: false,
	}
)
