package helium

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
)

// AsNode performs a safe type assertion on a [Node], returning the
// concrete type T and true if the assertion succeeds, or the zero value
// of T and false otherwise.
//
//	if elem, ok := helium.AsNode[*helium.Element](node); ok {
//	    // use elem
//	}
//
// A typed-nil pointer stored in a non-nil Node interface (Go's interface nil
// trap — e.g. the *Element returned by [Document.DocumentElement] for a
// document with no root) reports (zero, false), never (nil, true): a caller
// that gets ok == true can always safely dereference the result.
func AsNode[T Node](n Node) (T, bool) {
	var zero T
	if n == nil {
		return zero, false
	}
	v, ok := n.(T)
	if !ok {
		return zero, false
	}
	// The assertion matched, so v is about to be returned as ok. Reject a
	// typed-nil pointer here (isNilNode only when the assertion already
	// succeeded, so ordinary calls skip the reflect check) so callers never
	// receive a non-nil (T, true) wrapping a nil pointer.
	if isNilNode(v) {
		return zero, false
	}
	return v, true
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
	SetOwnerDocument(doc *Document)
	SetTreeDoc(doc *Document)
}

// Raw single-pointer linkage (parent/prev/next) is deliberately NOT part of
// MutableNode. Those pointers are maintained by the guarded AddChild /
// AddSibling / Replace / UnlinkNode operations, which keep the reciprocal
// back-pointers consistent and reject cycles. The unchecked primitives that set
// exactly one pointer live behind the explicitly-unsafe UnsafeSet* package
// functions below, so ordinary tree mutation cannot reach them by accident.

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
	content []byte
	// properties is the head of the element's attribute chain, linked through
	// each Attribute's next pointer. It is built exclusively through the guarded
	// property-splice (Element.addProperty) and Attribute.AddSibling paths, which
	// reject self/cycle insertion and never install a foreign link, so a
	// well-formed chain is a short, self-owned, acyclic list. The hot
	// attribute-lookup walks (Element.addProperty / HasAttribute / Attributes /
	// ForEachAttribute) therefore traverse it with a plain NextAttribute loop and
	// no per-list cycle guard. Whole-tree walks that may be handed an
	// externally-corrupted chain (setTreeDoc, the serializer) do carry a cheap
	// per-list seen guard.
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
//   cur.baseDocNode().parent = n
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

// Content aggregates the content of this node's own children. It advances
// between children with the owned-boundary rule (nextOwnedChild): a foreign
// child — an entity reference's shared Entity child, owned by the DTD, whose
// sibling pointers belong to the DTD declaration list — ends the aggregation
// instead of spilling into another list's siblings, and a per-list seen set
// stops a cyclic sibling pointer from looping forever. The receiver is a pointer
// so it is the real owning node against which child ownership is checked. The
// recursion into a container child's own subtree carries an ACTIVE-PATH set, so
// a pure child-pointer cycle (element -> element -> ... -> element, not routed
// through an Entity's terminating stored-text Content) terminates on the
// back-edge instead of recursing forever.
func (n *docnode) Content() []byte {
	b := bytes.Buffer{}
	aggregateOwnedContent(n, &b, map[*docnode]struct{}{n: {}})
	return b.Bytes()
}

// aggregateOwnedContent appends the concatenated content of n's own children to
// b. onPath is the set of container docnodes currently being aggregated (n
// inclusive): a child already on that path is a back-edge (a child-pointer
// cycle) and is skipped so the recursion terminates. onPath is an ACTIVE-PATH
// set, not a global visited set, so a shared DAG node reached on a different
// path is still re-aggregated per occurrence. A per-list seen set independently
// bounds a cyclic sibling pointer within one child list.
func aggregateOwnedContent(n *docnode, b *bytes.Buffer, onPath map[*docnode]struct{}) {
	seen := make(map[*docnode]struct{})
	for child := n.firstChild; child != nil; child = nextOwnedChild(n, child) {
		cdn := child.baseDocNode()
		if _, dup := seen[cdn]; dup {
			break
		}
		seen[cdn] = struct{}{}
		if _, active := onPath[cdn]; active {
			continue
		}
		// A leaf child (Text/Comment/CDATA/PI/Entity/NS wrapper) overrides
		// Content() with self-contained text that cannot loop, so call it
		// directly. Any other node aggregates its own children through this same
		// docnode path, so recurse under the active-path guard.
		if aggregatesOwnContent(child) {
			onPath[cdn] = struct{}{}
			aggregateOwnedContent(cdn, b, onPath)
			delete(onPath, cdn)
			continue
		}
		_, _ = b.Write(child.Content())
	}
}

// aggregatesOwnContent reports whether n's Content() is the child-aggregating
// docnode implementation (a container) rather than a self-contained leaf
// override. The leaf types enumerated here store their text directly and their
// Content() cannot recurse; every other node type — including any future
// container — aggregates its children and must be recursed under the
// active-path cycle guard.
func aggregatesOwnContent(n Node) bool {
	switch n.(type) {
	case *Text, *Comment, *CDATASection, *ProcessingInstruction, *Entity, *NamespaceNodeWrapper:
		return false
	default:
		return true
	}
}

// rawContentNode is implemented by leaf nodes (Text, Comment, CDATASection)
// that store their textual content in an internal mutable byte slice. It
// exposes that slice directly (without the defensive copy the exported
// Content() makes) for internal read-only hot paths such as serialization.
type rawContentNode interface {
	rawContent() []byte
}

// rawContent returns the internal content byte slice of n WITHOUT copying when
// n is a leaf node that aliases its content (Text, Comment, CDATASection).
// Callers MUST treat the result as read-only; mutating it corrupts the DOM.
// For any other node it falls back to the (already copy-safe) Content().
func rawContent(n Node) []byte {
	if rc, ok := n.(rawContentNode); ok {
		return rc.rawContent()
	}
	return n.Content()
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
//
// Walk is safe on hand-built or foreign-linked graphs that a plain
// child-pointer descent would loop on. It advances between siblings using the
// OWNED-BOUNDARY rule — a child whose Parent() is not the frame's node (an
// entity reference's shared Entity child, owned by the DTD, whose sibling
// pointers belong to another list) ends that child list — so the traversal
// never wanders out of a node's own children. It also carries the set of nodes
// currently on the DFS stack (the active path): descending into a node already
// on that path is a back-edge (a cycle), and Walk returns ErrWalkCycle instead
// of looping. Memory is O(active-path depth). A shared DAG node reached on a
// different path (not currently on the stack) is not a cycle and is still
// visited on each occurrence — Walk does not maintain a global visited set, so
// DAG traversal is unchanged. On an acyclic, parent-consistent tree behavior is
// identical to a naive recursive descent.
func Walk(n Node, w NodeWalker) error {
	// Reject both a literal nil interface and a typed-nil pointer (e.g. the
	// *Element that Document.DocumentElement returns for a rootless document)
	// with the matchable ErrNilNode, before any baseDocNode() dereference that
	// would panic on a typed nil.
	if isNilNode(n) {
		return ErrNilNode
	}

	type walkFrame struct {
		node        Node
		entered     bool
		activeChild Node
		// seenChildren records every child of node this frame has already
		// enumerated, so a child that repeats within the SAME sibling list —
		// a sibling cycle longer than one node (a -> b -> a, all siblings of
		// node) — is detected. The active-path guard alone misses it: each
		// child is popped and removed from onPath before its next sibling is
		// examined, so the enumeration would otherwise spin forever.
		seenChildren map[*docnode]struct{}
	}

	onPath := make(map[*docnode]struct{})
	stack := []walkFrame{{node: n}}
	for len(stack) > 0 {
		top := &stack[len(stack)-1]
		if !top.entered {
			if err := w.Visit(top.node); err != nil {
				return err
			}
			top.entered = true
			onPath[top.node.baseDocNode()] = struct{}{}
			top.activeChild = top.node.FirstChild()
			continue
		}

		if top.activeChild == nil {
			delete(onPath, top.node.baseDocNode())
			stack = stack[:len(stack)-1]
			if len(stack) > 0 {
				parent := &stack[len(stack)-1]
				parent.activeChild = nextWalkSibling(parent.node, parent.activeChild)
			}
			continue
		}

		childKey := top.activeChild.baseDocNode()
		if _, cyclic := onPath[childKey]; cyclic {
			return ErrWalkCycle
		}
		if _, dup := top.seenChildren[childKey]; dup {
			return ErrWalkCycle
		}
		if top.seenChildren == nil {
			top.seenChildren = make(map[*docnode]struct{})
		}
		top.seenChildren[childKey] = struct{}{}
		// top may dangle after the append reallocates stack; mark before it.
		stack = append(stack, walkFrame{node: top.activeChild})
	}
	return nil
}

// nextWalkSibling advances child to the next sibling within owner's own child
// list, applying the owned-boundary rule. It does NOT special-case a
// self-referential sibling pointer (child.next == child): the duplicate flows
// back to the caller so the per-frame seenChildren set detects it and Walk
// returns ErrWalkCycle, exactly as it does for a longer sibling cycle
// (a -> b -> a). Silently terminating the self-loop here would instead let Walk
// report SUCCESS on a corrupt one-node sibling cycle.
func nextWalkSibling(owner Node, child Node) Node {
	return nextOwnedChild(owner.baseDocNode(), child)
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
// cycle. That happens when parent is cur itself or is already reachable from
// cur: closing the link parent->cur then forms the loop parent -> cur -> ... ->
// parent.
//
// Walking parent's ANCESTOR chain (inclusive of parent) and looking for cur
// covers every such case — including the self-insertion cur == parent — at
// O(depth(parent)) WHEN parent/child links are consistent. But a child link may
// point at a node whose own parent pointer points elsewhere: an entity
// reference's child is the shared Entity node, whose parent stays the DTD
// (mirroring libxml2). A cycle formed through such a foreign link (e.g.
// ent.AddChild(ref) where ref's child is ent) is invisible to the ancestor
// walk, so when cur has children we additionally verify parent is not reachable
// from cur by following CHILD pointers. The parser hot path appends childless
// leaves and skips that second descent entirely.
func wouldCreateCycle(parent, cur Node) bool {
	cdn := cur.baseDocNode()
	for anc := parent; anc != nil; anc = anc.Parent() {
		if anc.baseDocNode() == cdn {
			return true
		}
	}
	if parent == nil || cdn.firstChild == nil {
		return false
	}
	return childReaches(cur, parent.baseDocNode())
}

// childReaches reports whether target is reachable from node by following child
// pointers (node inclusive). It walks ITERATIVELY with an explicit stack and a
// visited set, so it terminates on any child graph — shared (DAG) or hand-built
// cyclic — visiting each node at most once and never overflowing the goroutine
// stack on a deep tree. It is SOUND: it never bails out early, so a cycle at ANY
// depth is detected (a depth cap here would fail OPEN and admit a deep cycle).
// It enumerates each node's OWN children via nextOwnedChild so a foreign child
// link (an entity reference's Entity child, owned by the DTD) is not followed
// into another list's siblings, and it stops enumerating a sibling list as soon
// as it revisits a node — a cyclic sibling pointer (child.next == child, or a
// longer sibling loop) would otherwise spin here forever, since the popped-node
// visited set alone does not bound the inner enumeration.
func childReaches(node Node, target *docnode) bool {
	visited := make(map[*docnode]struct{})
	stack := []Node{node}
	for len(stack) > 0 {
		dn := stack[len(stack)-1].baseDocNode()
		stack = stack[:len(stack)-1]
		if dn == target {
			return true
		}
		if _, seen := visited[dn]; seen {
			continue
		}
		visited[dn] = struct{}{}
		siblingSeen := make(map[*docnode]struct{})
		for child := dn.firstChild; child != nil; child = nextOwnedChild(dn, child) {
			cdn := child.baseDocNode()
			if _, dup := siblingSeen[cdn]; dup {
				break
			}
			siblingSeen[cdn] = struct{}{}
			stack = append(stack, child)
		}
	}
	return false
}

// nextOwnedChild returns the next sibling of child within owner's child list, or
// nil when child is foreign-owned (its parent is not owner). A foreign child's
// sibling pointers belong to another list — an entity reference's Entity child
// is owned by the DTD — so following them would walk out of owner's children.
func nextOwnedChild(owner *docnode, child Node) Node {
	cp := child.Parent()
	if cp == nil || cp.baseDocNode() != owner {
		return nil
	}
	return child.NextSibling()
}

// destinationDocument returns the document a node inserted under n would belong
// to. For a Document receiver that is the document itself; for any other node it
// is the node's owning document.
func destinationDocument(n MutableNode) *Document {
	if d, ok := n.(*Document); ok {
		return d
	}
	return n.OwnerDocument()
}

// noteCrossDocumentEscape records that cur is being linked into a different
// document than the one that owns it. A node's backing storage (its struct and
// any text-content bytes) is drawn from its owning document's slab allocator, so
// once the node is referenced from another document that owning document must no
// longer recycle its slab chunks on Free — a later parse could otherwise reuse a
// chunk still holding the moved node and overwrite it. Marking the SOURCE
// document turns its Free into a no-op (GC reclaims the still-referenced chunks
// instead). A nil owner (a heap-allocated standalone node) has no slab to guard.
func noteCrossDocumentEscape(dest *Document, cur Node) {
	curDoc := cur.OwnerDocument()
	if curDoc == nil || curDoc == dest {
		return
	}
	curDoc.slabEscaped = true
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

	// A node linked into a different document keeps its slab-backed storage in its
	// original document, so guard that document's Free against recycling it. Mark
	// BEFORE any unlink, while cur still reports its original owner.
	noteCrossDocumentEscape(destinationDocument(n), cur)

	// Cycle guard: a node may not be inserted into itself, nor into one of
	// its own descendants (which would make an ancestor a descendant of
	// itself). This also catches the self-insertion case when n == cur.
	if wouldCreateCycle(n, cur) {
		return fmt.Errorf("%w: cannot add a node as a child of itself or one of its descendants", ErrCyclicNode)
	}

	// Detach cur from its current parent/sibling chain before relinking, so a
	// node that already lives elsewhere in a tree cannot remain in two places.
	// unlinkNode works for every sealed node type, including non-MutableNode
	// nodes such as NamespaceNodeWrapper, so the detach can never be silently
	// skipped and leave stale old-parent links behind.
	if cdn.parent != nil || cdn.prev != nil || cdn.next != nil {
		unlinkNode(cur)
	}

	return nil
}

func addChild(n MutableNode, cur Node) error {
	// Reject a nil or typed-nil operand BEFORE any baseDocNode() dereference so
	// the call returns ErrNilNode instead of panicking and leaves the tree
	// untouched.
	if isNilNode(cur) {
		return ErrNilNode
	}

	pdn := n.baseDocNode()
	cdn := cur.baseDocNode()

	if err := addChildPreflight(n, cur); err != nil {
		return err
	}

	l := pdn.lastChild
	if l == nil {
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

	// A sibling of n shares n's document; if cur comes from elsewhere, guard its
	// original document's Free against recycling its slab storage. Mark BEFORE any
	// unlink, while cur still reports its original owner. See noteCrossDocumentEscape.
	noteCrossDocumentEscape(n.OwnerDocument(), cur)

	// Cycle guard: a sibling of n is installed under n's parent, so the same
	// self/ancestor rule that protects addChild applies here against the
	// effective insertion parent. This also rejects cur == n (a node cannot be
	// its own sibling) since n is its parent's child.
	if cur.baseDocNode() == n.baseDocNode() || wouldCreateCycle(n.Parent(), cur) {
		return fmt.Errorf("%w: cannot add a node as a sibling of itself or one of its descendants", ErrCyclicNode)
	}

	// Detach cur from its current parent/sibling chain before relinking, so a
	// node that already lives elsewhere in a tree cannot remain in two places.
	// unlinkNode works for every sealed node type, including non-MutableNode
	// nodes such as NamespaceNodeWrapper, so the detach can never be silently
	// skipped and leave stale old-parent links behind.
	if cdn.parent != nil || cdn.prev != nil || cdn.next != nil {
		unlinkNode(cur)
	}

	return nil
}

func addSibling(n MutableNode, cur Node) error {
	// Reject a nil or typed-nil operand BEFORE any baseDocNode() dereference so
	// the call returns ErrNilNode instead of panicking and leaves the tree
	// untouched.
	if isNilNode(cur) {
		return ErrNilNode
	}

	cdn := cur.baseDocNode()
	ndn := n.baseDocNode()

	// Attribute-list semantics: attributes USUALLY live in the owning Element's
	// properties linked list, NOT in the parent's child list. When n is such a
	// property attribute, a new sibling must itself be an attribute and the splice
	// must stay within the attribute chain, never touching firstChild/lastChild.
	//
	// But an *Attribute with an *Element parent is not guaranteed to live in that
	// element's properties chain: public paths (elem.AddChild(attr), a generic
	// Replace(attr)) can place it in the normal child list instead. Only use
	// property-list logic when the anchor is genuinely reachable from
	// ownerElem.properties; otherwise fall through to the generic child-list path.
	if nAttr, ok := n.(*Attribute); ok {
		if ownerElem, ok := ndn.parent.(*Element); ok && ownerElem.hasAttributeInProperties(nAttr) {
			// Reject a non-attribute operand BEFORE the preflight unlink so a
			// rejected call leaves cur's old tree position untouched.
			if _, ok := cur.(*Attribute); !ok {
				return fmt.Errorf("%w: cannot add a non-attribute node as a sibling of an attribute", ErrInvalidOperation)
			}

			if err := addSiblingPreflight(n, cur); err != nil {
				return err
			}

			// Splice cur in only within the attribute sibling chain. Walk to the
			// tail attribute and append. Never touch parent.firstChild/lastChild:
			// attributes are not in the owner element's child list.
			iter := Node(n)
			for iter.NextSibling() != nil {
				iter = iter.NextSibling()
			}
			idn := iter.baseDocNode()
			idn.next = cur
			cdn.prev = iter
			cdn.parent = ownerElem
			return nil
		}
	}

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

// UnsafeSetParent sets ONLY n's parent pointer. It performs none of the cycle
// detection, auto-unlinking, or reciprocal back-pointer maintenance that
// AddChild/AddSibling/Replace/UnlinkNode provide, so a misuse leaves the tree
// inconsistent or cyclic. It exists for low-level tree construction and for
// tests that must build a deliberately corrupt tree to exercise the traversal
// cycle guards. Ordinary code MUST use AddChild/AddSibling/UnlinkNode instead.
func UnsafeSetParent(n Node, parent Node) {
	n.baseDocNode().parent = parent
}

// UnsafeSetPrevSibling sets ONLY n's previous-sibling pointer, with the same
// no-safeguards contract as UnsafeSetParent.
func UnsafeSetPrevSibling(n Node, prev Node) {
	n.baseDocNode().prev = prev
}

// UnsafeSetNextSibling sets ONLY n's next-sibling pointer, with the same
// no-safeguards contract as UnsafeSetParent.
func UnsafeSetNextSibling(n Node, next Node) {
	n.baseDocNode().next = next
}

// UnlinkNode detaches a node from its parent and sibling chain.
// After unlinking, the node has no parent, prev, or next pointers.
//
// A nil or typed-nil node (e.g. the *Element that Document.DocumentElement
// returns for a rootless document) is a no-op — there is nothing to detach.
func UnlinkNode(n MutableNode) {
	if isNilNode(n) {
		return
	}
	unlinkNode(n)
}

// unlinkNode detaches any [Node] from its parent and sibling chain, operating
// purely through baseDocNode() pointers. It works for every sealed node type,
// including those that are NOT MutableNode (e.g. NamespaceNodeWrapper), so any
// already-linked incoming node can be safely detached before relinking without
// a MutableNode type assertion that would silently skip or panic.
func unlinkNode(n Node) {
	if n == nil {
		return
	}

	ndn := n.baseDocNode()

	// Attributes are USUALLY stored in the owning Element's properties linked
	// list, NOT in the parent's child list. Detach via spliceOutAttribute so the
	// Element.properties head is repaired and the attribute sibling chain is
	// patched, without ever touching the parent's firstChild/lastChild. But an
	// attribute with an *Element parent is not guaranteed to be a property:
	// public paths (elem.AddChild(attr), a generic Replace(attr)) can place it in
	// the normal child list instead. Confirm it is actually reachable from
	// elem.properties before using property-list logic; otherwise fall through to
	// the generic child-list unlink below.
	if attr, ok := n.(*Attribute); ok {
		if elem, ok := ndn.parent.(*Element); ok && elem.hasAttributeInProperties(attr) {
			elem.spliceOutAttribute(attr)
			return
		}
	}

	if parent := ndn.parent; parent != nil {
		pdn := parent.baseDocNode()
		if pdn.firstChild != nil && pdn.firstChild.baseDocNode() == ndn {
			pdn.firstChild = ndn.next
		}
		if pdn.lastChild != nil && pdn.lastChild.baseDocNode() == ndn {
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
	// An empty replacement set is rejected with ErrInvalidOperation, matching
	// Document.Replace: "replace this node with nothing" is not a supported way
	// to delete a node (use UnlinkNode for that). Reporting success here would
	// silently do nothing while every other Replace contract mutates the tree.
	if len(nodes) == 0 {
		return ErrInvalidOperation
	}

	// Reject a nil or typed-nil replacement operand BEFORE any baseDocNode()
	// dereference so the call returns ErrNilNode instead of panicking and
	// leaves the tree untouched. Validate every operand, not just the first.
	if slices.ContainsFunc(nodes, isNilNode) {
		return ErrNilNode
	}

	cur := nodes[0]
	cdn := cur.baseDocNode()
	ndn := n.baseDocNode()

	// Attribute-list semantics: attributes USUALLY live in the owning Element's
	// properties linked list, NOT in the parent's child list. When n is such a
	// property attribute, every replacement must itself be an attribute, and the
	// Element.properties head must be repaired instead of firstChild/lastChild.
	// Reject a mixed/non-attribute replacement before any unlink/splice so a
	// rejected call leaves the tree untouched.
	//
	// But an *Attribute with an *Element parent is not guaranteed to live in that
	// element's properties chain: public paths (elem.AddChild(attr), a generic
	// Replace(attr)) can place it in the normal child list instead. Only use
	// property-list logic when the attribute is genuinely reachable from
	// ownerElem.properties; otherwise fall back to the generic child-list splice
	// so firstChild/lastChild are repaired.
	nAttr, nIsAttr := n.(*Attribute)
	ownerElem, _ := ndn.parent.(*Element)
	attrList := nIsAttr && ownerElem != nil && ownerElem.hasAttributeInProperties(nAttr)
	if attrList {
		for _, nn := range nodes {
			if nn.baseDocNode() == ndn {
				continue
			}
			if _, ok := nn.(*Attribute); !ok {
				return fmt.Errorf("%w: cannot replace an attribute with a non-attribute node", ErrInvalidOperation)
			}
		}
	}

	// Duplicate-operand guard: the same node cannot appear twice among the
	// replacements. Splicing it into two positions of the new sibling chain
	// would corrupt its prev/next links (e.g. b.prev == b). Reject before any
	// unlink/splice so a rejected call leaves the tree untouched.
	seen := make(map[*docnode]struct{}, len(nodes))
	for _, nn := range nodes {
		dn := nn.baseDocNode()
		if _, dup := seen[dn]; dup {
			return fmt.Errorf("%w: cannot replace a node with duplicate replacement operands", ErrInvalidOperation)
		}
		seen[dn] = struct{}{}
	}

	// Cycle guard: each replacement node takes n's place under n's parent, so
	// installing the parent (or any ancestor of it) below itself would create a
	// cycle. Reject before any unlink/splice so a rejected call leaves the tree
	// untouched. n itself is exempt: when n is among the replacements it stays
	// live in place (handled below as replacedIsInserted).
	parent := ndn.parent
	replDoc := n.OwnerDocument()
	for _, nn := range nodes {
		if nn.baseDocNode() == ndn {
			continue
		}
		// Each replacement takes n's place. If it comes from a different document,
		// guard that document's Free against recycling its slab storage. Mark BEFORE
		// any unlink, while nn still reports its original owner. See
		// noteCrossDocumentEscape.
		noteCrossDocumentEscape(replDoc, nn)
		if wouldCreateCycle(parent, nn) {
			return fmt.Errorf("%w: cannot replace a node with one of its own ancestors", ErrCyclicNode)
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
		// unlinkNode handles every sealed node type, so a non-MutableNode
		// replacement (e.g. NamespaceNodeWrapper) is detached safely instead of
		// panicking on a MutableNode force-cast.
		unlinkNode(nn)
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
		if attrList {
			// n is the owner Element's first attribute when properties points at
			// it; move the head to the first replacement attribute. Never touch
			// firstChild/lastChild: attributes are not in the child list. cur is
			// guaranteed an *Attribute here: the attribute-only check above rejected
			// any non-attribute replacement when n is an attribute.
			if curAttr, ok := cur.(*Attribute); ok && ownerElem.properties == n {
				ownerElem.properties = curAttr
			}
		}
		if !attrList {
			pdn := parent.baseDocNode()
			if pdn.firstChild == n {
				pdn.firstChild = cur
			}
			if pdn.lastChild == n {
				pdn.lastChild = cur
			}
		}
		cdn.parent = parent
	}

	// Determine the true last replacement node. Operate on baseDocNode() links
	// directly rather than through MutableNode setters so a non-MutableNode
	// replacement (e.g. NamespaceNodeWrapper) is spliced safely instead of
	// panicking on a force-cast.
	last := cur
	ldn := cdn
	for i := 1; i < len(nodes); i++ {
		c := nodes[i]
		cn := c.baseDocNode()
		cn.parent = ldn.parent
		cn.prev = last
		ldn.next = c
		last = c
		ldn = cn
	}

	// Link last replacement to whatever followed n
	ldn.next = afterN
	if afterN != nil {
		afterN.baseDocNode().prev = last
	}

	// Update parent's lastChild if n was the last child and we added more nodes.
	// Skip for the attribute-list case: attributes are not in the child list, so
	// the parent's lastChild must never be retargeted at an attribute.
	if !attrList && afterN == nil && len(nodes) > 1 {
		if parent := cdn.parent; parent != nil {
			parent.baseDocNode().lastChild = last
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
	return slices.Clone(n.nsDefs)
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
//
// The declaration is idempotent per prefix and leaves the node holding at most
// one declaration for the prefix: if the prefix is already declared with the
// same URI the call is a no-op; if it is declared with a different URI (or is
// declared more than once) the first declaration is rebound to uri and every
// other declaration of the same prefix is dropped. The first declaration is
// replaced with a fresh Namespace rather than rewriting the existing object, so
// a Namespace a caller supplied via AddNamespaceDecl is never mutated. When the
// node's active namespace uses the same prefix it is repointed to the single
// surviving declaration. Serialization therefore never emits a duplicate or
// conflicting xmlns for the prefix.
//
// The call is rejected with an error wrapping ErrInvalidOperation, leaving the
// node unchanged, when an attribute on the node binds the same prefix to a
// different URI: rebinding the prefix would change that attribute's namespace,
// and keeping the attribute's binding would force a second, conflicting xmlns
// for the prefix, so the at-most-one-declaration guarantee cannot be met.
func (n *node) DeclareNamespace(prefix, uri string) error {
	// An attribute genuinely bound to this prefix at a different URI cannot be
	// reconciled to a single declaration without changing its meaning. Checked
	// before any mutation so the node is unchanged on rejection.
	for attr := n.properties; attr != nil; attr = attr.NextAttribute() {
		if ans := attr.ns; ans != nil && ans.prefix == prefix && ans.href != uri {
			return fmt.Errorf("%w: attribute binds prefix %q to a conflicting namespace %q", ErrInvalidOperation, prefix, ans.href)
		}
	}

	activeSamePrefix := n.ns != nil && n.ns.prefix == prefix

	// Fast path: the prefix is already declared exactly once with this URI and the
	// active namespace (if it uses the prefix) already resolves to this URI, so
	// there is nothing to reconcile and no allocation is needed.
	matches := 0
	var firstMatch *Namespace
	for _, ns := range n.nsDefs {
		if ns.Prefix() == prefix {
			matches++
			if firstMatch == nil {
				firstMatch = ns
			}
		}
	}
	if matches == 1 && firstMatch.href == uri && (!activeSamePrefix || n.ns.href == uri) {
		return nil
	}

	decl, err := n.doc.CreateNamespace(prefix, uri)
	if err != nil {
		return err
	}

	// Rebind the first declaration of the prefix to uri and drop every later one,
	// so the node holds at most one declaration per prefix. The first slot is
	// replaced with the fresh Namespace rather than rewriting a possibly
	// caller-owned object.
	replaced := false
	kept := n.nsDefs[:0]
	for _, ns := range n.nsDefs {
		if ns.Prefix() != prefix {
			kept = append(kept, ns)
			continue
		}
		if !replaced {
			kept = append(kept, decl)
			replaced = true
		}
	}
	if !replaced {
		kept = append(kept, decl)
	}
	n.nsDefs = kept

	// When the active namespace uses this prefix it must resolve to the single
	// surviving declaration, not a stale second object serialized separately.
	if activeSamePrefix {
		n.ns = decl
	}
	return nil
}

// AddNamespaceDecl appends an existing Namespace to this node's declarations
// (nsDefs) without allocating a new one. Unlike DeclareNamespace it does not
// create a fresh Namespace, so a caller building a tree can reuse one Namespace
// object as both the declaration and an element's active namespace. The caller
// owns ns; it must not be shared as a declaration across nodes that could be
// mutated independently.
func (n *node) AddNamespaceDecl(ns *Namespace) {
	n.nsDefs = append(n.nsDefs, ns)
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

func setListDoc(n Node, doc *Document) {
	if isNilNode(n) || n.Type() == NamespaceDeclNode {
		return
	}

	// A per-list seen guard bounds a cyclic sibling pointer: the OwnerDocument
	// early-continue below does NOT terminate a 2-cycle (a -> b -> a) once both
	// nodes already carry doc, so without this the walk would spin. Child-pointer
	// cycles are already broken by setTreeDoc's "already owns doc" early return.
	seen := make(map[*docnode]struct{})
	for cur := n; cur != nil; cur = cur.NextSibling() {
		cdn := cur.baseDocNode()
		if _, dup := seen[cdn]; dup {
			break
		}
		seen[cdn] = struct{}{}
		if cur.OwnerDocument() == doc {
			continue
		}
		// A non-MutableNode node (e.g. NamespaceNodeWrapper) cannot recurse
		// through SetTreeDoc; set its document directly via baseDocNode(),
		// mirroring unlinkNode's force-cast-free approach. MutableNode nodes
		// still go through SetTreeDoc so their children are walked too.
		if mn, ok := cur.(MutableNode); ok {
			mn.SetTreeDoc(doc)
			continue
		}
		cur.baseDocNode().doc = doc
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
		// A per-list seen guard bounds a cyclic attribute chain (a low-level
		// SetNextSibling misuse); a normal properties list is short and acyclic.
		seenAttrs := make(map[*docnode]struct{})
		for prop := e.properties; prop != nil; prop = prop.NextAttribute() {
			pdn := prop.baseDocNode()
			if _, dup := seenAttrs[pdn]; dup {
				break
			}
			seenAttrs[pdn] = struct{}{}
			// if prop.atype == XML_ATTRIBUTE_ID; xmlRemoveID(tree->doc, prop)
			prop.doc = doc
			if child := prop.firstChild; child != nil {
				setListDoc(child, doc)
			}
		}
	}
	if child := n.FirstChild(); child != nil {
		setListDoc(child, doc)
	}
	n.SetOwnerDocument(doc)
}
