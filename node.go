package helium

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/lestrrat-go/pdebug"
)

// AsNode performs a safe type assertion on a [Node], returning the
// concrete type T and true if the assertion succeeds, or the zero value
// of T and false otherwise.
//
//	if elem, ok := helium.AsNode[*helium.Element](node); ok {
//	    // use elem
//	}
func AsNode[T Node](n Node) (T, bool) {
	if n == nil {
		var zero T
		return zero, false
	}
	if v, ok := n.(T); ok {
		return v, true
	}
	var zero T
	return zero, false
}

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

	// NamespaceNode represents a namespace declaration (does not exist in libxml2).
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
	if last := n.LastChild(); last != nil {
		if t, ok := AsNode[*Text](last); ok {
			return t.AppendText(b)
		}
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

// wouldCreateCycle reports whether installing cur under parent would create a
// cycle. That happens when cur is the parent itself or an ancestor of the
// parent: making it a descendant would put a node below itself. Walking
// parent's ancestor chain (inclusive of parent) and looking for cur covers
// both cases, including the self-insertion case when cur == parent.
func wouldCreateCycle(parent, cur Node) bool {
	cdn := cur.baseDocNode()
	for anc := parent; anc != nil; anc = anc.Parent() {
		if anc.baseDocNode() == cdn {
			return true
		}
	}
	return false
}

// addChildPreflight runs the shared self/cycle guard and auto-unlink that every
// AddChild path must perform before relinking. It returns a non-nil error when
// the operation must be rejected; on success cur is detached from any previous
// position and safe to splice in. Leaf AddChild overrides (Text, Comment, ...)
// reuse this so their content-merge fast paths cannot bypass the guard: a node
// must not be merged into itself, and an already-linked incoming node must be
// unlinked from its old parent first.
func addChildPreflight(n MutableNode, cur Node) error {
	cdn := cur.baseDocNode()

	// Cycle guard: a node may not be inserted into itself, nor into one of
	// its own descendants (which would make an ancestor a descendant of
	// itself). This also catches the self-insertion case when n == cur.
	if wouldCreateCycle(n, cur) {
		return errors.New("cannot add a node as a child of itself or one of its descendants")
	}

	// Detach cur from its current parent/sibling chain before relinking, so a
	// node that already lives elsewhere in a tree cannot remain in two places.
	if cdn.parent != nil || cdn.prev != nil || cdn.next != nil {
		if cmn, ok := cur.(MutableNode); ok {
			UnlinkNode(cmn)
		}
	}

	return nil
}

func addChild(n MutableNode, cur Node) error {
	pdn := n.baseDocNode()
	cdn := cur.baseDocNode()

	if err := addChildPreflight(n, cur); err != nil {
		return err
	}

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
	if err := l.(MutableNode).AddSibling(cur); err != nil { //nolint:forcetypeassert
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

// addSiblingPreflight runs the shared self/cycle guard and auto-unlink that
// every AddSibling path must perform before relinking. It returns a non-nil
// error when the operation must be rejected; on success cur is detached from
// any previous position and safe to splice in. Text.AddSibling reuses this so
// its text-merge fast path cannot bypass the guard.
func addSiblingPreflight(n MutableNode, cur Node) error {
	cdn := cur.baseDocNode()

	// Cycle guard: a sibling of n is installed under n's parent, so the same
	// self/ancestor rule that protects addChild applies here against the
	// effective insertion parent. This also rejects cur == n (a node cannot be
	// its own sibling) since n is its parent's child.
	if cur.baseDocNode() == n.baseDocNode() || wouldCreateCycle(n.Parent(), cur) {
		return errors.New("cannot add a node as a sibling of itself or one of its descendants")
	}

	// Detach cur from its current parent/sibling chain before relinking, so a
	// node that already lives elsewhere in a tree cannot remain in two places.
	if cdn.parent != nil || cdn.prev != nil || cdn.next != nil {
		if cmn, ok := cur.(MutableNode); ok {
			UnlinkNode(cmn)
		}
	}

	return nil
}

func addSibling(n MutableNode, cur Node) error {
	cdn := cur.baseDocNode()

	if err := addSiblingPreflight(n, cur); err != nil {
		return err
	}

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

	// Duplicate-operand guard: the same node cannot appear twice among the
	// replacements. Splicing it into two positions of the new sibling chain
	// would corrupt its prev/next links (e.g. b.prev == b). Reject before any
	// unlink/splice so a rejected call leaves the tree untouched.
	seen := make(map[*docnode]struct{}, len(nodes))
	for _, nn := range nodes {
		dn := nn.baseDocNode()
		if _, dup := seen[dn]; dup {
			return errors.New("cannot replace a node with duplicate replacement operands")
		}
		seen[dn] = struct{}{}
	}

	// Cycle guard: each replacement node takes n's place under n's parent, so
	// installing the parent (or any ancestor of it) below itself would create a
	// cycle. Reject before any unlink/splice so a rejected call leaves the tree
	// untouched. n itself is exempt: when n is among the replacements it stays
	// live in place (handled below as replacedIsInserted).
	parent := ndn.parent
	for _, nn := range nodes {
		if nn.baseDocNode() == ndn {
			continue
		}
		if wouldCreateCycle(parent, nn) {
			return errors.New("cannot replace a node with one of its own ancestors")
		}
	}

	// A replacement node may already be linked into the tree (e.g. replacing a
	// node with its own sibling). Detach every replacement node from its current
	// position before splicing so it cannot remain in n's neighbor chain and
	// create a self-loop. Skip n itself: when n is among the replacements it
	// stays live in place (handled below as replacedIsInserted).
	for _, nn := range nodes {
		if nn.baseDocNode() == ndn {
			continue
		}
		UnlinkNode(nn.(MutableNode)) //nolint:forcetypeassert
	}

	// Capture n's following sibling AFTER detaching replacement nodes so it
	// always points at a node that survives the splice.
	afterN := ndn.next

	// Patch first replacement into n's position
	if ndn.prev != nil {
		cdn.prev = ndn.prev
		ndn.prev.baseDocNode().next = cur
	}
	if parent != nil {
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
	last := cur.(MutableNode) //nolint:forcetypeassert
	for i := 1; i < len(nodes); i++ {
		c := nodes[i].(MutableNode) //nolint:forcetypeassert
		c.SetParent(last.Parent())
		c.SetPrevSibling(last)
		last.SetNextSibling(c)
		last = c
	}

	// Link last replacement to whatever followed n
	last.SetNextSibling(afterN)
	if afterN != nil {
		afterN.(MutableNode).SetPrevSibling(last) //nolint:forcetypeassert
	}

	// Update parent's lastChild if n was the last child and we added more nodes
	if afterN == nil && len(nodes) > 1 {
		if parent := cdn.parent; parent != nil {
			setLastChild(parent.(MutableNode), last) //nolint:forcetypeassert
		}
	}

	// The replaced node is logically removed from the tree. Clear its own
	// parent/sibling links so a stale handle cannot rewrite the spliced-in
	// replacement (e.g. via a later UnlinkNode or Replace). Skip this when the
	// replaced node is itself one of the inserted nodes (e.g. self-replacement),
	// since it remains live in the tree and clearing its links would corrupt it.
	replacedIsInserted := false
	for _, nn := range nodes {
		if nn.baseDocNode() == ndn {
			replacedIsInserted = true
			break
		}
	}
	if !replacedIsInserted {
		ndn.parent = nil
		ndn.prev = nil
		ndn.next = nil
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
			cur.(MutableNode).SetTreeDoc(doc) //nolint:forcetypeassert
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

	if e, ok := AsNode[*Element](n); ok {
		for prop := e.properties; prop != nil; prop = prop.NextAttribute() {
			// if prop.atype == XML_ATTRIBUTE_ID; xmlRemoveID(tree->doc, prop)
			prop.doc = doc
			if child := prop.firstChild; child != nil {
				setListDoc(child.(MutableNode), doc) //nolint:forcetypeassert
			}
		}
	}
	if child := n.FirstChild(); child != nil {
		setListDoc(child.(MutableNode), doc) //nolint:forcetypeassert
	}
	n.SetOwnerDocument(doc)
}
