package node

import (
	"errors"
)

type prefix string

func (p *prefix) SetPrefix(s string) {
	*p = prefix(s)
}

func (p *prefix) Prefix() string {
	if p == nil {
		return ""
	}
	return string(*p)
}

// NodeType represents the type of a node in the XML tree
type NodeType int

const (
	ElementNodeType NodeType = iota + 1
	AttributeNodeType
	TextNodeType
	CDATASectionNodeType
	EntityRefNodeType
	EntityNodeType
	ProcessingInstructionNodeType
	CommentNodeType
	DocumentNodeType
	DocumentTypeNodeType
	DocumentFragNodeType
	NotationNodeType
	HTMLDocumentNodeType
	DTDNodeType
	ElementDeclNodeType
	AttributeDeclNodeType
	EntityDeclNodeType
	NamespaceDeclNodeType
	XIncludeStartNodeType
	XIncludeEndNodeType

	// This doesn't exist in libxml2. Do we need it?
	NamespaceNodeType
)

var ErrInvalidOperation = errors.New("invalid operation")

// Node interface defines the common functionality for all node types
type Node interface {
	// returns the treeNode (the part of the Node that handles the tree structure)
	getTreeNode() *treeNode

	AddChild(Node) error
	AddContent([]byte) error
	AddSibling(Node) error

	Type() NodeType
	// Content appends the content of the node to the provided byte slice and returns the result.
	// If dst is nil, a new slice is allocated.
	Content(dst []byte) ([]byte, error)

	FirstChild() Node
	LastChild() Node

	// LocalName returns the local name of the node.
	LocalName() string

	NextSibling() Node
	OwnerDocument() *Document
	Parent() Node
	PrevSibling() Node

	Replace(Node) error

	SetNextSibling(Node) error
	SetOwnerDocument(doc *Document) error
	SetParent(Node) error
	SetPrevSibling(Node) error
	//SetTreeDoc(doc *Document) error
}

type DocumentStandaloneType int

const (
	StandaloneInvalidValue = -99
	StandaloneExplicitYes  = 1
	StandaloneExplicitNo   = 0
	StandaloneNoXMLDecl    = -1
	StandaloneImplicitNo   = -2
)

// DTD represents a document type definition
type DTD struct {
	treeNode
	attributes map[string]*AttributeDecl
	elements   map[string]*ElementDecl
	entities   map[string]*Entity
	pentities  map[string]*Entity
	externalID string
	systemID   string
}
