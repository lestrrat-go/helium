package helium

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/lestrrat-go/pdebug"
)

type Node interface {
	AddChild(Node) error
	AddContent([]byte) error
	AddSibling(Node) error
	Content() []byte
	FirstChild() Node
	LastChild() Node
	Line() int
	Name() string
	NextSibling() Node
	OwnerDocument() *Document
	Parent() Node
	PrevSibling() Node
	Replace(Node)
	SetLine(int)
	SetNextSibling(Node)
	SetOwnerDocument(doc *Document)
	SetParent(Node)
	SetPrevSibling(Node)
	SetTreeDoc(doc *Document)
	Type() ElementType
}

// docnode is responsible for handling the basic tree-ish operations
type docnode struct {
	name       string
	etype      ElementType
	firstChild Node
	lastChild  Node
	parent     Node
	next       Node
	prev       Node
	doc        *Document
	line       int
}

// node represents a node in a XML tree.
type node struct {
	docnode
	// private    interface{}
	content    []byte
	properties *Attribute
	ns         *Namespace
	nsDefs     []*Namespace
}

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

const _ElementType_name = "ElementNodeAttributeNodeTextNodeCDATASectionNodeEntityRefNodeEntityNodeProcessingInstructionNodeCommentNodeDocumentNodeDocumentTypeNodeDocumentFragNodeNotationNodeHTMLDocumentNodeDTDNodeElementDeclNodeAttributeDeclNodeEntityDeclNodeNamespaceDeclNodeXIncludeStartNodeXIncludeEndNodeNamespaceNode"

var _ElementType_index = [...]uint16{0, 11, 24, 32, 48, 61, 71, 96, 107, 119, 135, 151, 163, 179, 186, 201, 218, 232, 249, 266, 281, 294}

func (i ElementType) String() string {
	i -= 1
	if i < 0 || i >= ElementType(len(_ElementType_index)-1) {
		return fmt.Sprintf("ElementType(%d)", i+1)
	}
	return _ElementType_name[_ElementType_index[i]:_ElementType_index[i+1]]
}

type NamespaceContainer interface {
	Namespaces() []*Namespace
}

// Nemaspacer is an interface for things that has a namespace
// prefix and uri
type Namespacer interface {
	Namespace() *Namespace
	Namespaces() []*Namespace
	Prefix() string
	URI() string
	LocalName() string
}

// because docnode contains links to other nodes, one tends to want to make
// methods for docnodes that cover the rest of the Node types. However,
// this cannot be done because the way Go does method reuse -- by delegation.
// For example, a method that changes the parent's point to the current node would
// be bad:
//
// func (n *docnode) MakeMeYourParent(cur Node) {
//   cur.SetParent(n)
// }
//
// Wait, you just passed a pointer to the docnode, not the container node
// such as Element, Text, Comment, etc.
//
// So basically the deal is: if you need methods that may mutate the current
// node AND the operand node, DO NOT implement it for docnode. That includes
// things like AddSibling, or AddChild.

func (n *docnode) baseDocNode() *docnode {
	return n
}

func setFirstChild(n Node, cur Node) {
	n.(interface{ baseDocNode() *docnode }).baseDocNode().firstChild = cur
}

func setLastChild(n Node, cur Node) {
	n.(interface{ baseDocNode() *docnode }).baseDocNode().lastChild = cur
}

func (n *docnode) SetOwnerDocument(doc *Document) {
	n.doc = doc
}

func (n docnode) OwnerDocument() *Document {
	return n.doc
}

func (n docnode) Parent() Node {
	return n.parent
}

func (n docnode) Content() []byte {
	b := bytes.Buffer{}
	for e := n.firstChild; e != nil; e = e.NextSibling() {
		_, _ = b.Write(e.Content())
	}
	return b.Bytes()
}

func addContent(n Node, b []byte) error {
	t := newText(b)
	return n.AddChild(t)
}

type WalkFunc func(Node) error

func Walk(n Node, f WalkFunc) error {
	if n == nil {
		return errors.New("nil node")
	}

	if err := f(n); err != nil {
		return err
	}
	for chld := n.FirstChild(); chld != nil; chld = chld.NextSibling() {
		if err := Walk(chld, f); err != nil {
			return err
		}
	}
	return nil
}

func (n docnode) LocalName() string {
	return n.name
}

func (n docnode) Name() string {
	return n.name
}

func (n docnode) Type() ElementType {
	return n.etype
}

func (n docnode) Line() int {
	return n.line
}

func (n *docnode) SetLine(line int) {
	n.line = line
}

func (n docnode) FirstChild() Node {
	return n.firstChild
}

func (n docnode) LastChild() Node {
	return n.lastChild
}

func addChild(n Node, cur Node) error {
	l := n.LastChild()
	if l == nil { // No children, set firstChild to cur
		if pdebug.Enabled {
			pdebug.Printf("LastChild is nil, setting firstChild and lastChild")
		}
		setFirstChild(n, cur)
		setLastChild(n, cur)
		cur.SetParent(n)
		return nil
	}

	// AddSibling handles setting the parent, and the
	// lastChild pointer
	if err := l.AddSibling(cur); err != nil {
		return err
	}

	// If the last child was a text node, keep the old LastChild
	if cur.Type() == TextNode && l.Type() == TextNode {
		setLastChild(n, l)
	}
	return nil
}

func (n docnode) NextSibling() Node {
	if n.next == nil {
		return nil
	}
	return n.next
}

func (n docnode) PrevSibling() Node {
	return n.prev
}

func addSibling(n, cur Node) error {
	for n != nil {
		if n.NextSibling() == nil {
			n.SetNextSibling(cur)
			cur.SetPrevSibling(n)
			parent := n.Parent()
			cur.SetParent(parent)
			if parent != nil {
				setLastChild(parent, cur)
			}
			return nil
		}
		n = n.NextSibling()
	}

	return errors.New("cannot add sibling to nil node")
}

func (n *docnode) SetPrevSibling(cur Node) {
	n.prev = cur
}

func (n *docnode) SetNextSibling(cur Node) {
	n.next = cur
}

func (n *docnode) SetParent(cur Node) {
	n.parent = cur
}

// UnlinkNode detaches a node from its parent and sibling chain.
// After unlinking, the node has no parent, prev, or next pointers.
func UnlinkNode(n Node) {
	if n == nil {
		return
	}

	if parent := n.Parent(); parent != nil {
		if parent.FirstChild() == n {
			setFirstChild(parent, n.NextSibling())
		}
		if parent.LastChild() == n {
			setLastChild(parent, n.PrevSibling())
		}
	}

	if prev := n.PrevSibling(); prev != nil {
		prev.SetNextSibling(n.NextSibling())
	}
	if next := n.NextSibling(); next != nil {
		next.SetPrevSibling(n.PrevSibling())
	}

	n.SetParent(nil)
	n.SetPrevSibling(nil)
	n.SetNextSibling(nil)
}

func replaceNode(n Node, cur Node) {
	if next := n.NextSibling(); next != nil {
		cur.SetNextSibling(next) // cur.next = n.next
		next.SetPrevSibling(cur) // n.next.prev = cur
	}

	if prev := n.PrevSibling(); prev != nil {
		cur.SetPrevSibling(prev) // cur.prev = n.prev
		prev.SetNextSibling(cur) // n.prev.next = cur
	}

	if parent := n.Parent(); parent != nil {
		if parent.FirstChild() == n {
			setFirstChild(parent, cur)
		}
		if parent.LastChild() == n {
			setLastChild(parent, cur)
		}
		cur.SetParent(parent)
	}
}

func (n node) Namespace() *Namespace {
	return n.ns
}

func (n node) Namespaces() []*Namespace {
	return n.nsDefs
}

func (n *node) SetNamespace(prefix, uri string, activate ...bool) error {
	ns, err := n.doc.CreateNamespace(prefix, uri)
	if err != nil {
		return err
	}

	a := false
	if len(activate) > 0 {
		a = activate[0]
	}
	if a {
		n.ns = ns
	} else {
		n.nsDefs = append(n.nsDefs, ns)
	}

	return nil
}

func (n node) Prefix() string {
	if ns := n.ns; ns != nil {
		return ns.Prefix()
	}
	return ""
}

func (n node) URI() string {
	if ns := n.ns; ns != nil {
		return ns.URI()
	}
	return ""
}

func (n node) Name() string {
	if ns := n.ns; ns != nil && ns.Prefix() != "" {
		return ns.Prefix() + ":" + n.name
	}
	return n.name
}

func SetListDoc(n Node, doc *Document) {
	if n == nil || n.Type() == NamespaceDeclNode {
		return
	}

	for ; n != nil; n = n.NextSibling() {
		if n.OwnerDocument() != doc {
			n.SetTreeDoc(doc)
		}
	}
}

func setTreeDoc(n Node, doc *Document) {
	if n == nil || n.Type() == NamespaceDeclNode {
		return
	}

	if n.OwnerDocument() == doc {
		return
	}

	if n.Type() == ElementNode {
		e := n.(*Element)
		for prop := e.properties; prop != nil; prop = prop.NextAttribute() {
			// if prop.atype == XML_ATTRIBUTE_ID; xmlRemoveID(tree->doc, prop)
			prop.doc = doc
			if child := prop.firstChild; child != nil {
				SetListDoc(child, doc)
			}
		}
	}
	if child := n.FirstChild(); child != nil {
		SetListDoc(child, doc)
	}
	n.SetOwnerDocument(doc)
}
