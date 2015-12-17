package helium

import "errors"

var (
	ErrNilNode          = errors.New("nil node")
	ErrInvalidOperation = errors.New("operation cannot be performed")
)

type nilNode struct {}

// common data
type node struct {
	typ     NodeType
	private interface{}
	//etype    ElementType
	name     string
	children []Node
	parent   Node
	next     Node
	prev     Node
	doc      *Document
}

type Document struct {
	node
	version    string
	encoding   string
	standalone bool

	intSubset *DTD
}

type ProcessingInstruction struct {
	node
	target string
	data string
}

type DTD struct {
	node
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

type Element struct {
	node
	content    string // XXX probably wrong
	attributes []Attribute
	prefix     string
}
