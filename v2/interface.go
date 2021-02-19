package helium

import "github.com/lestrrat-go/helium/v2/sax"

type Parser struct {
	sax sax.Handler
}

type Node interface {
	setFirstChild(Node)
	setLastChild(Node)

	AddChild(Node) error
	AddContent([]byte) error
	AddSibling(Node) error

	Content() []byte

	FirstChild() Node
	LastChild() Node
	NextSibling() Node
	Parent() Node
	PrevSibling() Node
	SetNextSibling(Node)
	SetPrevSibling(Node)
	SetParent(Node)
	Type() ElementType
}

type DocumentStandaloneType int

const (
	StandaloneInvalidValue = -99
	StandaloneExplicitYes  = 1
	StandaloneExplicitNo   = 0
	StandaloneNoXMLDecl    = -1
	StandaloneImplicitNo   = -2
)

type ElementType int

const (
	ElementNode ElementType = iota + 1
	AttributeNode
	TextNode
	CDATASectionNode
	EntityRefNode
	EntityNode
	ProcessingInstructionNode
	CommentNode
	DocumentNode
	DocumentTypeNode
	DocumentFragNode
	NotationNode
	HTMLDocumentNode
	DTDNode
	ElementDeclNode
	AttributeDeclNode
	EntityDeclNode
	NamespaceDeclNode
	XIncludeStartNode
	XIncludeEndNode

	// This doesn't exist in libxml2. Do we need it?
	NamespaceNode
)

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

type EntityType int

const (
  InternalGeneralEntity EntityType = iota + 1
  ExternalGeneralParsedEntity
  ExternalGeneralUnparsedEntity
  InternalParameterEntity
  ExternalParameterEntity
  InternalPredefinedEntity
)

