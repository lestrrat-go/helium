package helium

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/lestrrat-go/pdebug"
)

// Node represents a node in an XML document tree (libxml2: xmlNode).
type Node interface {
	baseDocNode() *docnode // prevents external implementation

	AddChild(Node) error
	// AppendText appends text content to this node (libxml2: xmlNodeAddContent).
	AppendText([]byte) error
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
	Replace(Node) error
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
	name          string
	etype         ElementType
	firstChild    Node
	lastChild     Node
	parent        Node
	next          Node
	prev          Node
	doc           *Document
	line          int
	entityBaseURI string // non-empty when this node originates from an external parsed entity
}

// node represents a node in a XML tree.
type node struct {
	docnode
	// private    interface{}
	content    []byte
	properties *Attribute
	ns         *Namespace
	nsDefs     []*Namespace
	qname      string // cached qualified name (prefix:local or just local)
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
	n.baseDocNode().firstChild = cur
}

func setLastChild(n Node, cur Node) {
	n.baseDocNode().lastChild = cur
}

// SetLastChild updates the last-child pointer of a parent node.
// This is needed when manually splicing nodes into a sibling list
// without going through AddChild/AddSibling.
func SetLastChild(n Node, cur Node) {
	setLastChild(n, cur)
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

func appendText(n Node, b []byte) error {
	// Fast path: if last child is already a text node, append directly
	// without allocating a new Text node.
	if last := n.LastChild(); last != nil && last.Type() == TextNode {
		return last.(*Text).AppendText(b)
	}
	// Use slab allocator when the node belongs to a document.
	if doc := n.OwnerDocument(); doc != nil {
		t, _ := doc.CreateText(b)
		return n.AddChild(t)
	}
	t := newText(b)
	return n.AddChild(t)
}

// NodeWalker visits nodes during tree traversal.
type NodeWalker interface {
	Visit(Node) error
}

// NodeWalkerFunc is an adapter to allow use of ordinary functions as NodeWalker.
// Similar to http.HandlerFunc.
type NodeWalkerFunc func(Node) error

func (f NodeWalkerFunc) Visit(n Node) error {
	return f(n)
}

// Walk performs a depth-first traversal of the node tree rooted at n,
// calling w.Visit for each node. There is no direct libxml2 equivalent; callers
// typically write manual tree traversal loops in C.
func Walk(n Node, w NodeWalker) error {
	if n == nil {
		return errors.New("nil node")
	}

	type walkFrame struct {
		node        Node
		entered     bool
		activeChild Node
	}

	stack := []walkFrame{{node: n}}
	for len(stack) > 0 {
		top := &stack[len(stack)-1]
		if !top.entered {
			if err := w.Visit(top.node); err != nil {
				return err
			}
			top.entered = true
			top.activeChild = top.node.FirstChild()
			continue
		}

		if top.activeChild == nil {
			stack = stack[:len(stack)-1]
			if len(stack) > 0 {
				parent := &stack[len(stack)-1]
				parent.activeChild = parent.activeChild.NextSibling()
			}
			continue
		}

		stack = append(stack, walkFrame{node: top.activeChild})
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

	// Fast path: when lastChild has no next sibling (the normal case),
	// link directly without virtual dispatch through AddSibling.
	if l.NextSibling() == nil && (cur.Type() != TextNode || l.Type() != TextNode) {
		l.SetNextSibling(cur)
		cur.SetPrevSibling(l)
		cur.SetParent(n)
		setLastChild(n, cur)
		return nil
	}

	// AddSibling handles setting the parent, and the
	// lastChild pointer (also merges adjacent text nodes)
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

func replaceNode(n Node, cur Node) error {
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
	return nil
}

func (n node) Namespace() *Namespace {
	return n.ns
}

func (n node) Namespaces() []*Namespace {
	return n.nsDefs
}

// RemoveNamespaceByPrefix removes a namespace declaration with the given prefix.
// Returns true if a declaration was removed.
func (n *node) RemoveNamespaceByPrefix(prefix string) bool {
	for i, ns := range n.nsDefs {
		if ns.Prefix() == prefix {
			n.nsDefs = append(n.nsDefs[:i], n.nsDefs[i+1:]...)
			return true
		}
	}
	return false
}

// DeclareNamespace declares a namespace on this node without making it the
// node's active namespace (libxml2: xmlNewNs).
func (n *node) DeclareNamespace(prefix, uri string) error {
	ns, err := n.doc.CreateNamespace(prefix, uri)
	if err != nil {
		return err
	}
	n.nsDefs = append(n.nsDefs, ns)
	return nil
}

// SetActiveNamespace declares a namespace and sets it as this node's active
// namespace (libxml2: xmlSetNs).
func (n *node) SetActiveNamespace(prefix, uri string) error {
	ns, err := n.doc.CreateNamespace(prefix, uri)
	if err != nil {
		return err
	}
	n.ns = ns
	n.invalidateQName()
	return nil
}

// SetNs sets the node's active namespace to an existing Namespace object
// without creating a new declaration.
func (n *node) SetNs(ns *Namespace) {
	n.ns = ns
	n.invalidateQName()
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

func (n *node) Name() string {
	if n.qname != "" {
		return n.qname
	}
	if ns := n.ns; ns != nil && ns.Prefix() != "" {
		n.qname = ns.Prefix() + ":" + n.name
		return n.qname
	}
	return n.name
}

func (n *node) invalidateQName() {
	n.qname = ""
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
