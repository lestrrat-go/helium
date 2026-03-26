package helium

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/lestrrat-go/pdebug"
)

// Node is a read-only view of an XML document tree node (libxml2: xmlNode).
type Node interface {
	baseDocNode() *docnode // prevents external implementation

	Content() []byte
	FirstChild() Node
	LastChild() Node
	Line() int
	Name() string
	NextSibling() Node
	OwnerDocument() *Document
	Parent() Node
	PrevSibling() Node
	Type() ElementType
}

// MutableNode extends Node with tree-mutation operations.
type MutableNode interface {
	Node

	AddChild(Node) error
	AddSibling(Node) error
	// AppendText appends text content to this node (libxml2: xmlNodeAddContent).
	AppendText([]byte) error
	Replace(...Node) error
	SetLine(int)
	SetNextSibling(Node)
	SetOwnerDocument(doc *Document)
	SetParent(Node)
	SetPrevSibling(Node)
	SetTreeDoc(doc *Document)
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

// NamespaceContainer is an interface for nodes that carry namespace declarations.
type NamespaceContainer interface {
	Namespaces() []*Namespace
}

// Namespacer is an interface for things that have a namespace
// prefix and URI.
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

func setFirstChild(n MutableNode, cur Node) {
	n.baseDocNode().firstChild = cur
}

func setLastChild(n MutableNode, cur Node) {
	n.baseDocNode().lastChild = cur
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

func appendText(n MutableNode, b []byte) error {
	// Fast path: if last child is already a text node, append directly
	// without allocating a new Text node.
	if last := n.LastChild(); last != nil && last.Type() == TextNode {
		return last.(*Text).AppendText(b)
	}
	// Use slab allocator when the node belongs to a document.
	if doc := n.OwnerDocument(); doc != nil {
		t := doc.CreateText(b)
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

func addChild(n MutableNode, cur Node) error {
	pdn := n.baseDocNode()
	cdn := cur.baseDocNode()

	l := pdn.lastChild
	if l == nil {
		if pdebug.Enabled {
			pdebug.Printf("LastChild is nil, setting firstChild and lastChild")
		}
		pdn.firstChild = cur
		pdn.lastChild = cur
		cdn.parent = n
		return nil
	}

	ldn := l.baseDocNode()
	curType := cdn.etype
	// Fast path: when lastChild has no next sibling (the normal case),
	// link directly without virtual dispatch through AddSibling.
	if ldn.next == nil && (curType != TextNode || ldn.etype != TextNode) {
		ldn.next = cur
		cdn.prev = l
		cdn.parent = n
		pdn.lastChild = cur
		return nil
	}

	// AddSibling handles setting the parent, and the
	// lastChild pointer (also merges adjacent text nodes)
	if err := l.(MutableNode).AddSibling(cur); err != nil {
		return err
	}

	// If the last child was a text node, keep the old LastChild
	if curType == TextNode && ldn.etype == TextNode {
		pdn.lastChild = l
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

func addSibling(n MutableNode, cur Node) error {
	cdn := cur.baseDocNode()
	iter := Node(n)
	for iter != nil {
		if iter.NextSibling() == nil {
			idn := iter.baseDocNode()
			idn.next = cur
			cdn.prev = iter
			parent := iter.Parent()
			cdn.parent = parent
			if parent != nil {
				parent.baseDocNode().lastChild = cur
			}
			return nil
		}
		iter = iter.NextSibling()
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
func UnlinkNode(n MutableNode) {
	if n == nil {
		return
	}

	ndn := n.baseDocNode()

	if parent := ndn.parent; parent != nil {
		pdn := parent.baseDocNode()
		if pdn.firstChild == n {
			pdn.firstChild = ndn.next
		}
		if pdn.lastChild == n {
			pdn.lastChild = ndn.prev
		}
	}

	if prev := ndn.prev; prev != nil {
		prev.baseDocNode().next = ndn.next
	}
	if next := ndn.next; next != nil {
		next.baseDocNode().prev = ndn.prev
	}

	ndn.parent = nil
	ndn.prev = nil
	ndn.next = nil
}

func replaceNode(n MutableNode, nodes ...Node) error {
	if len(nodes) == 0 {
		return nil
	}
	cur := nodes[0]
	cdn := cur.baseDocNode()
	ndn := n.baseDocNode()
	afterN := ndn.next

	// Patch first replacement into n's position
	if ndn.prev != nil {
		cdn.prev = ndn.prev
		ndn.prev.baseDocNode().next = cur
	}
	if parent := ndn.parent; parent != nil {
		pdn := parent.baseDocNode()
		if pdn.firstChild == n {
			pdn.firstChild = cur
		}
		if pdn.lastChild == n {
			pdn.lastChild = cur
		}
		cdn.parent = parent
	}

	// Determine the true last replacement node
	last := cur.(MutableNode)
	for i := 1; i < len(nodes); i++ {
		c := nodes[i].(MutableNode)
		c.SetParent(last.Parent())
		c.SetPrevSibling(last)
		last.SetNextSibling(c)
		last = c
	}

	// Link last replacement to whatever followed n
	last.SetNextSibling(afterN)
	if afterN != nil {
		afterN.(MutableNode).SetPrevSibling(last)
	}

	// Update parent's lastChild if n was the last child and we added more nodes
	if afterN == nil && len(nodes) > 1 {
		if parent := cdn.parent; parent != nil {
			setLastChild(parent.(MutableNode), last)
		}
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

func setListDoc(n MutableNode, doc *Document) {
	if n == nil || n.Type() == NamespaceDeclNode {
		return
	}

	for cur := Node(n); cur != nil; cur = cur.NextSibling() {
		if cur.OwnerDocument() != doc {
			cur.(MutableNode).SetTreeDoc(doc)
		}
	}
}

func setTreeDoc(n MutableNode, doc *Document) {
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
				setListDoc(child.(MutableNode), doc)
			}
		}
	}
	if child := n.FirstChild(); child != nil {
		setListDoc(child.(MutableNode), doc)
	}
	n.SetOwnerDocument(doc)
}
