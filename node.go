package helium

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"

	"github.com/lestrrat-go/helium/internal/nodelink"
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

// ElementType identifies the kind of a DOM node (element, text, comment, and so
// on). It mirrors libxml2's xmlElementType. Use it to distinguish node kinds
// returned by the Node interface's Type method; the enumerated constants below
// name each kind.
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

// SetOwnerDocument makes doc this node's owning document. An off-chain-claim
// record travels with the node (adoptOffChainClaims), so a document that adopts a
// subtree holding such a claim inherits the record instead of starting out
// trusting its own lastChild.
func (n *docnode) SetOwnerDocument(doc *Document) {
	adoptOffChainClaims(n.doc, doc)
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
// docnode implementation (a container), as opposed to a self-contained leaf
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
// from cur by following CHILD pointers.
//
// A cur with an EMPTY child list is settled from cur's own claimant slots
// instead, at a cost bounded by cur's own shape rather than by depth(parent):
// the parser hot path appends childless leaves, and this is what keeps building
// a deep chain linear. The shortcut rests on one invariant: the search that
// replaces the walk (claimantReaches) traverses exactly the edges X -> Y for
// which Y names X as its parent, which is the ancestor walk's own relation read
// forward, and it is declined outright whenever a node may name cur from
// outside those edges (holdsOffChainClaim).
func wouldCreateCycle(parent, cur Node) bool {
	cdn := cur.baseDocNode()
	// Self-insertion is the one loop the claimant search below cannot see. The
	// ancestor walk is inclusive of parent, so parent == cur closes a loop even
	// when nothing else names cur at all.
	if parent != nil && parent.baseDocNode() == cdn {
		return true
	}
	// A cur with no child list is settled from its own claimant slots instead
	// of parent's ancestor chain. Both questions the walk answers need some
	// node X with X.parent == cur — the ancestor walk finds cur only by
	// stepping from such an X, and the child descent needs cur.firstChild —
	// and with the child list empty every remaining X sits in a slot cur owns,
	// SO LONG AS no off-chain parent claim has been made. Searching those slots
	// directly costs cur's own attribute count rather than depth(parent), which
	// is what makes a deep AddChild chain linear.
	if cdn.firstChild == nil && !holdsOffChainClaim(cur) {
		return claimantReaches(cur, parent)
	}
	for anc := parent; anc != nil; anc = anc.Parent() {
		if anc.baseDocNode() == cdn {
			return true
		}
	}
	if parent == nil {
		return false
	}
	return childReaches(cur, parent.baseDocNode())
}

// holdsOffChainClaim reports whether a node may name cur as its parent from
// outside every slot cur owns, which is the one way the claimant search can
// miss a claimant and answer false where the ancestor walk answers true.
//
// The guarded paths create exactly such a claim themselves: an append onto a
// parent that holds a firstChild with no lastChild overwrites firstChild, and
// the child that was there goes on claiming the parent from no slot at all
// (noteOrphanedChildClaim). Unlinking the replacement then empties the child
// list again, which is precisely when the claimant search is consulted.
//
// The record lives on the owning document, and on unownedOffChainClaim for a
// claim made while no document owned the nodes. A node with no document must
// consult that package-level record: its own document field cannot carry what
// was never written to a document, and a node whose document is taken away
// keeps its claim there (adoptOffChainClaims).
//
// A DOCUMENT can also be claimed as a parent from outside its own child list by
// a copied external subset, which Document.offChainChildClaim records. That
// record is about the document node itself, so it decides this question only
// when the document IS the operand; a node the document merely owns is not
// claimed by it.
func holdsOffChainClaim(cur Node) bool {
	doc := owningDocument(cur)
	if doc == nil {
		return unownedOffChainClaim.Load()
	}
	if doc.offChainClaims {
		return true
	}
	return doc == cur && doc.offChainChildClaim
}

// claimantReaches reports whether parent lies anywhere under a cur that has no
// child list, following ONLY the edges X -> Y for which Y names X as its
// parent. That relation is the ancestor walk's own, read forward: the walk
// finds cur from parent only along a chain of parent pointers, so a descent
// restricted to the same edges answers the same question.
//
// Only a node naming cur as its parent can begin such a chain, and with
// cur.firstChild nil and no off-chain claim outstanding every one of them sits
// in a slot cur itself owns: an element's attributes, or a document's DTD
// subsets. A slot entry that does NOT name cur back is skipped, because no
// ancestor chain can pass through it: unlinkNode clears a subset's parent while
// leaving Document.intSubset pointing at it, and the parser installs an
// external subset without writing its parent at all. Each surviving subtree is
// bounded by the operand's own shape, so the search never grows with the depth
// of the tree cur is being inserted into.
//
// A namespace-node wrapper also names its owner without appearing in any of
// the owner's slots, and is deliberately not consulted here: a wrapper has no
// AddChild/AddSibling of its own and docnode supplies none, so it can never be
// an insertion point, and nothing can ever be linked beneath one. No ancestor
// path this guard walks can pass through a wrapper.
func claimantReaches(cur Node, parent Node) bool {
	if parent == nil {
		return false
	}
	cdn := cur.baseDocNode()
	pdn := parent.baseDocNode()
	switch n := cur.(type) {
	case *Element:
		for attr := n.properties; attr != nil; attr = attr.NextAttribute() {
			if !namesParent(attr.baseDocNode(), cdn) {
				continue
			}
			if parentVerifiedReaches(attr, pdn) {
				return true
			}
		}
	case *Document:
		for _, sub := range [2]*DTD{n.intSubset, n.extSubset} {
			if sub == nil || !namesParent(sub.baseDocNode(), cdn) {
				continue
			}
			if parentVerifiedReaches(sub, pdn) {
				return true
			}
		}
	}
	return false
}

// namesParent reports whether x names owner as its parent. It is the one edge
// relation the claimant search may traverse, in both directions of use: a slot
// entry is consulted only when it names the operand, and a child is descended
// into only when it names the node being expanded.
func namesParent(x, owner *docnode) bool {
	p := x.parent
	return p != nil && p.baseDocNode() == owner
}

// parentVerifiedReaches reports whether target is reachable from node along
// child links the child names back — it descends into a child Y of X exactly
// when Y.parent is X. node itself counts, because the caller has already
// verified node names the operand.
//
// That is what separates it from childReaches, which follows a child pointer
// whoever the child claims as its parent, and the difference is the point. A
// FOREIGN child link — an entity reference's child is the shared Entity, whose
// parent stays the DTD — is a real child pointer but never a step the ancestor
// walk could take in reverse, so following it here would answer true where the
// walk answers false. childReaches follows foreign links because it exists to
// catch a cycle formed THROUGH one; this descent must not, because it stands in
// for the walk.
//
// A foreign child ends no enumeration: its next pointer belongs to another
// list, so what follows it may name any parent, but every node is tested
// against the node being expanded before it is descended into, and one that
// names that node is a genuine claimant however it was reached. Termination
// comes from the same two bounds childReaches uses — siblingCycleGuard on each
// sibling enumeration, and the popped-node visited set on the search as a
// whole.
func parentVerifiedReaches(node Node, target *docnode) bool {
	var visited childReachesVisited
	stack := []Node{node}
	for len(stack) > 0 {
		dn := stack[len(stack)-1].baseDocNode()
		stack = stack[:len(stack)-1]
		if dn == target {
			return true
		}
		if visited.has(dn) {
			continue
		}
		visited.add(dn)
		var g siblingCycleGuard
		for child := dn.firstChild; child != nil; {
			cdn := child.baseDocNode()
			if g.step(cdn) {
				break
			}
			if namesParent(cdn, dn) {
				stack = append(stack, child)
			}
			child = cdn.next
		}
	}
	return false
}

// childReachesInlineCap is the number of popped nodes childReachesVisited
// tracks in its inline array before promoting to a map. It selects a DATA
// STRUCTURE only, never a search cutoff: above the cap the search continues
// exactly as before, backed by a map instead of a linear scan. Instrumentation
// across the xslt3 conformance suite recorded 2,119,653 childReaches calls
// popping 6,457,455 nodes total (mean 3.05, max 10,922), with 94.1% of calls
// popping 2 or fewer nodes, so 64 clears nearly every real call while still
// bounding the linear scan's cost on the rare deep one.
const childReachesInlineCap = 64

// childReachesVisited is the popped-node visited set for childReaches. It
// starts as a fixed-size array scanned linearly — no allocation — and
// promotes itself to a map once more than childReachesInlineCap distinct
// nodes have been recorded, from which point the map is authoritative. This
// mirrors the measured call distribution: almost every call stays entirely in
// the array, and only the rare call with a very large subtree pays for a map.
type childReachesVisited struct {
	inline [childReachesInlineCap]*docnode
	n      int
	m      map[*docnode]struct{}
}

// has reports whether dn was previously recorded by add.
func (v *childReachesVisited) has(dn *docnode) bool {
	if v.m != nil {
		_, ok := v.m[dn]
		return ok
	}
	for i := range v.n {
		if v.inline[i] == dn {
			return true
		}
	}
	return false
}

// add records dn as visited, promoting the inline array to a map the first
// time the array fills up.
func (v *childReachesVisited) add(dn *docnode) {
	if v.m != nil {
		v.m[dn] = struct{}{}
		return
	}
	if v.n < len(v.inline) {
		v.inline[v.n] = dn
		v.n++
		return
	}
	v.m = make(map[*docnode]struct{}, v.n+1)
	for i := range v.inline {
		v.m[v.inline[i]] = struct{}{}
	}
	v.m[dn] = struct{}{}
}

// childReaches reports whether target is reachable from node by following child
// pointers (node inclusive). It walks ITERATIVELY with an explicit stack and a
// visited set, so it terminates on any child graph — shared (DAG) or hand-built
// cyclic — visiting each node at most once and never overflowing the goroutine
// stack on a deep tree. It is SOUND: it never bails out early, so a cycle at ANY
// depth is detected (a depth cap here would fail OPEN and admit a deep cycle).
// It enumerates each node's OWN children via nextOwnedSibling so a foreign child
// link (an entity reference's Entity child, owned by the DTD) is not followed
// into another list's siblings.
//
// The inner sibling enumeration is bounded by [siblingCycleGuard] — the same
// allocation-free Brent's-algorithm guard [Children]/[ChildElements]/
// [Descendants] already use — rather than a per-call sibling-seen set. The
// trade is where the enumeration stops: a seen set stops at the exact first
// repeat, while siblingCycleGuard stops within a small multiple of the cycle
// length, so on a corrupt sibling list a few nodes may be pushed onto the stack
// more than once. That is harmless here — the popped-node visited set
// deduplicates those extra pushes on pop, and the overshoot is bounded — and
// termination is unconditional either way.
func childReaches(node Node, target *docnode) bool {
	var visited childReachesVisited
	stack := []Node{node}
	for len(stack) > 0 {
		dn := stack[len(stack)-1].baseDocNode()
		stack = stack[:len(stack)-1]
		if dn == target {
			return true
		}
		if visited.has(dn) {
			continue
		}
		visited.add(dn)
		var g siblingCycleGuard
		for child := dn.firstChild; child != nil; {
			cdn := child.baseDocNode()
			if g.step(cdn) {
				break
			}
			stack = append(stack, child)
			child = nextOwnedSibling(dn, cdn)
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

// nextOwnedSibling returns the next sibling of cdn within owner's child list, or
// nil when cdn is foreign-owned. It is nextOwnedChild for a caller that already
// holds the child's *docnode, and applies the identical owned-boundary rule: a
// foreign child's sibling pointers belong to another list — an entity
// reference's Entity child is owned by the DTD — so following them would walk
// out of owner's children.
//
// It reads cdn.parent and cdn.next as FIELDS where nextOwnedChild calls Parent()
// and NextSibling(). Those methods have VALUE receivers on docnode, which is 136
// bytes, so every interface call copies the struct, and the iterators in iter.go
// pay that on each child. The field read yields the same value without the copy.
//
// This is a SPEED choice and carries no semantics. It holds only while no node
// type overrides FirstChild/Parent/NextSibling — none does, and
// TestNodeLinkAccessorsMatchFields pins it. Should one ever need to, this helper
// and the owner.firstChild reads at the head of each iterator loop return to the
// method calls with no other change and no difference in behavior.
func nextOwnedSibling(owner, cdn *docnode) Node {
	p := cdn.parent
	if p == nil || p.baseDocNode() != owner {
		return nil
	}
	return cdn.next
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
	if curDoc == dest {
		return
	}
	// A node crossing into dest's tree brings its links with it, so an
	// off-chain-claim record on the document it came from has to travel too:
	// dest's own lastChild is what the O(1) append-point resolution trusts.
	adoptOffChainClaims(curDoc, dest)
	if curDoc == nil {
		return
	}
	curDoc.slabEscaped = true
}

// noteCrossDocumentNamespaceEscape is the Namespace counterpart of
// noteCrossDocumentEscape. A slab-backed Namespace is drawn from its owning
// document's namespace slab (its owner is Namespace.context, set by
// Document.CreateNamespace; a heap-allocated Namespace has a nil context and no
// slab). When AddNamespaceDecl retains such a Namespace in another document's
// declarations, or SetNamespace installs it as another document's node's active
// namespace, the owning document must no longer recycle its namespace slab on
// Free, so mark it escaped exactly as a cross-document node move does.
func noteCrossDocumentNamespaceEscape(dest *Document, ns *Namespace) {
	if ns == nil {
		return
	}
	src := ns.context
	if src == nil || src == dest {
		return
	}
	src.slabEscaped = true
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

	// An Attribute is never an ordinary child node. On an *Element it belongs in
	// the properties list, mirroring libxml2's xmlAddChild, which routes an
	// attribute operand into the parent's properties (replacing a same-named one)
	// and never into the child list; on any other parent an attribute has no valid
	// placement and is rejected. Handle this BEFORE the generic child-splice (and
	// before any leaf content-merge fast path, which the leaf AddChild overrides
	// reach only for text-like operands) so an attribute can never land in a child
	// list and serialize as a spurious child element.
	if attr, ok := cur.(*Attribute); ok {
		elem, ok := n.(*Element)
		if !ok {
			return fmt.Errorf("%w: cannot add an attribute as a child of a %s node; attributes belong on an element", ErrInvalidOperation, n.Type())
		}
		// Preflight through the normal guards (cross-document escape marking,
		// auto-unlink from any previous parent/property chain). An attribute cannot
		// form a child-list cycle, but it must be detached from a prior location
		// before addProperty splices it in.
		if err := addChildPreflight(elem, attr); err != nil {
			return err
		}
		elem.addProperty(attr)
		return nil
	}

	pdn := n.baseDocNode()
	cdn := cur.baseDocNode()

	if err := addChildPreflight(n, cur); err != nil {
		return err
	}

	l := pdn.lastChild
	if l == nil {
		noteOrphanedChildClaim(n, pdn.firstChild)
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

// chainMember reports whether x is a member of the single child chain pdn owns:
// the chain that starts at pdn.firstChild and runs forward through next
// pointers. It answers ONE question, about the anchor of an append: may
// pdn.lastChild be taken as the tail of the chain that anchor sits on? It never
// decides whether pdn.lastChild may be WRITTEN — every write of that field is
// unconditional, exactly as it is in the sibling walk this shortcut replaces.
//
// One pointer comparison answers the anchor an append most often uses: an x that
// IS pdn.firstChild is the chain head by definition, so appending through a
// fixed early child stays O(1) per call.
//
// Every OTHER anchor costs a walk: step prev to the head of x's own chain, and
// x is a member exactly when that head is pdn.firstChild. The walk costs x's
// distance BEHIND it, never the length of the chain ahead of it, so it can
// never cost more than the NextSibling() walk it is protecting — but it does
// grow with the chain, so repeatedly appending through a fixed MIDDLE anchor
// stays quadratic and merely wins a constant factor. siblingCycleGuard bounds
// the walk, so a corrupt prev chain terminates instead of spinning.
//
// Every prev edge the proof crosses must be RECIPROCAL — the step from head to
// head.prev is taken only when that prev node points forward at head again.
// This is what a bare prev walk gets wrong. A one-way prev edge outlives any
// splice that cuts a node out of a chain from the FRONT: the node keeps pointing
// back at a neighbour that no longer points forward at it. Such a node can live
// on a chain of its OWN while still aiming at a genuine child of the parent it
// claims. Following the one-way edge would leave x's chain, arrive at
// pdn.firstChild, and "prove" a membership that does not exist — after which the
// caller would splice into the parent's real child list and abandon the rest of
// x's chain. Rejecting the non-reciprocal edge costs one pointer comparison per
// step, so the walk keeps the bound above.
func chainMember(pdn, x *docnode) bool {
	if pdn == nil || x == nil {
		return false
	}
	first := pdn.firstChild
	if first == nil {
		return false
	}
	if first.baseDocNode() == x {
		return true
	}

	var g siblingCycleGuard
	head := x
	for {
		if g.step(head) {
			return false
		}
		prev := reciprocalPrev(head)
		if prev == nil {
			// Either head starts its chain, or its prev edge is one-way. A
			// one-way edge proves nothing, so only a genuine head may match.
			return head.prev == nil && first.baseDocNode() == head
		}
		head = prev
	}
}

// reciprocalPrev returns x's previous sibling when that edge is reciprocal —
// when the prev node points forward at x again — and nil otherwise. A nil
// return therefore means "x has no usable prev edge", covering both a genuine
// chain head and a forged one-way link.
func reciprocalPrev(x *docnode) *docnode {
	prev := x.prev
	if prev == nil {
		return nil
	}
	pdn := prev.baseDocNode()
	if pdn.next == nil || pdn.next.baseDocNode() != x {
		return nil
	}
	return pdn
}

// tailJumpTarget returns the node an append through anchor ndn may be spliced
// after, resolved from parent.lastChild without walking the chain ahead of the
// anchor, or nil when the append must walk instead. A nil return is never an
// error: the walk is the behavior addSibling guarantees, and this is only a
// shortcut to the node that walk would reach.
//
// The shortcut needs two facts, and only the first can be established by reading
// the neighborhood:
//
//  1. The anchor is a member of the chain parent owns. chainMember proves this,
//     in a pointer comparison for an anchor that is parent.firstChild, and
//     otherwise by walking reciprocal prev edges back to the head. That proof is
//     sound on any tree.
//
//  2. parent.lastChild is the final node of that same chain. NO amount of local
//     reading can establish this: two trees can have pointer-identical
//     neighborhoods around parent and lastChild and differ only in a next
//     pointer an unbounded distance forward from firstChild. It holds as an
//     invariant of the guarded paths — every one of them moves lastChild only to
//     a node it has just linked onto the chain — so what is checked instead is
//     that no node in this document claims a parent it is not a child of.
//     Document.offChainClaims records exactly that, so the shortcut is declined
//     for a document that holds such a claim.
//
// The owning document is read through owningDocument, so a *Document parent
// names ITSELF: a document node holds the trees it owns and its own doc pointer
// stays nil, which is a fact about how a document is initialized and not a
// statement about whether its child list can be trusted. A document is claimed
// off-chain by CopyExtSubset, which gives the copied external subset the
// destination document as its parent and then leaves it reachable only through
// ExtSubset, never from the child list, so an append through that subset records
// its own result as the document's tail and moves the record off the child list.
// That is a condition, not a type, and Document.offChainChildClaim records it:
// the shortcut is declined for a document that has actually been handed such a
// claimant, and taken for every other one. (CreateInternalSubset also gives a
// DTD the document as its parent, but it splices that DTD into the child list,
// so it creates no claim.)
//
// The remaining reads are cheap confirmations that the record is a usable tail:
// it must not be the anchor itself, it must genuinely end its chain, and it must
// claim this very parent. On a document holding no off-chain claim they cannot
// fail, and they are kept anyway because the record is per-DOCUMENT while a tree
// is not: a cross-document node move (noteCrossDocumentEscape) leaves one
// document's chain holding nodes another document owns, and these three
// comparisons are what makes a claim that slipped past the record degrade to the
// walk instead of splicing onto the wrong node. They also keep the
// stale-lastChild repair path addChild and appendFastChild depend on: they call
// AddSibling on parent.lastChild precisely when that node's next is non-nil, so
// the shortcut declines and the walk finds and repairs the true tail.
func tailJumpTarget(parent Node, ndn *docnode) Node {
	if parent == nil {
		return nil
	}
	doc := owningDocument(parent)
	if doc == nil || doc.offChainClaims {
		return nil
	}
	if doc == parent && doc.offChainChildClaim {
		return nil
	}
	pdn := parent.baseDocNode()

	tail := pdn.lastChild
	if tail == nil {
		return nil
	}
	tdn := tail.baseDocNode()
	if tdn == ndn || tdn.next != nil || tdn.parent == nil || tdn.parent.baseDocNode() != pdn {
		return nil
	}
	if !chainMember(pdn, ndn) {
		return nil
	}
	return tail
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
	// element's properties chain: a generic Replace(attr) that swaps a child node
	// for an attribute can place it in the normal child list instead. Only use
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

	// Resolve the append point from parent.lastChild when that record can be
	// trusted, and otherwise walk the sibling chain. The walk is the behavior
	// this function guarantees: the O(1) resolution is taken only where it lands
	// on the very node the walk would have reached, so the tree an append leaves
	// behind never depends on which route was taken. Every parent.lastChild write
	// below is unconditional for the same reason — the walk writes it
	// unconditionally, so a guard on the write would be a behavior change, not a
	// safety check.
	parent := ndn.parent

	// n is already the tail: link directly, which is what the first iteration of
	// the walk below does. The lastChild write is unconditional there, so it is
	// unconditional here.
	if ndn.next == nil {
		// n ends its own chain, but that chain is not necessarily parent's child
		// list. A parent that lists NO children cannot be listing n, so cur is
		// about to name a parent it can be found from only through lastChild —
		// an off-chain claim, recorded here at the moment it is created so the
		// cycle guard's claimant search declines a parent whose claimants it can
		// no longer enumerate. On a parent that does list children this is the
		// ordinary tail append and records nothing.
		if parent != nil && parent.baseDocNode().firstChild == nil {
			noteOrphanedChildClaim(parent, cur)
		}
		ndn.next = cur
		cdn.prev = n
		cdn.parent = parent
		if parent != nil {
			parent.baseDocNode().lastChild = cur
		}
		return nil
	}

	// n is not the tail. Rather than walk NextSibling() from n — O(S) per call
	// over the whole chain AHEAD of n — jump straight to the tail parent.lastChild
	// already records, whenever tailJumpTarget can prove that record is the node
	// the walk would have found. Any tree it cannot vouch for falls through to the
	// walk unchanged.
	//
	// How much this buys depends on the anchor. An anchor that IS parent.firstChild
	// is proven by pointer comparison, so appending through it is O(1) per call and
	// O(S) over S appends, where the walk was O(S^2). An anchor in the MIDDLE of
	// the chain is proven by walking prev back to the head,
	// so it stays O(S^2) over S appends; what it gains is a large constant factor,
	// since the prev walk covers only the chain behind the anchor while the
	// NextSibling() walk covered the whole chain ahead of it. BenchmarkAddSibling
	// measures all three shapes.
	if tail := tailJumpTarget(parent, ndn); tail != nil {
		tdn := tail.baseDocNode()
		tdn.next = cur
		cdn.prev = tail
		cdn.parent = tdn.parent
		tdn.parent.baseDocNode().lastChild = cur
		return nil
	}

	iter := Node(n)
	for iter != nil {
		if iter.NextSibling() == nil {
			idn := iter.baseDocNode()
			idn.next = cur
			cdn.prev = iter
			endParent := iter.Parent()
			cdn.parent = endParent
			if endParent != nil {
				endParent.baseDocNode().lastChild = cur
			}
			return nil
		}
		iter = iter.NextSibling()
	}

	return errors.New("cannot add sibling to nil node")
}

// unsafeSetParent sets ONLY n's parent pointer. It performs none of the cycle
// detection, auto-unlinking, or reciprocal back-pointer maintenance that
// AddChild/AddSibling/Replace/UnlinkNode provide, so a misuse leaves the tree
// inconsistent or cyclic. It exists for low-level tree construction and for
// tests that must build a deliberately corrupt tree to exercise the traversal
// cycle guards. Ordinary code MUST use AddChild/AddSibling/UnlinkNode instead.
func unsafeSetParent(n Node, parent Node) {
	n.baseDocNode().parent = parent
}

// unsafeSetNextSibling sets ONLY n's next-sibling pointer. It performs none of
// the cycle detection, auto-unlinking, or reciprocal back-pointer maintenance
// that AddChild/AddSibling/Replace/UnlinkNode provide, so a misuse leaves the
// tree inconsistent or cyclic. It exists for low-level tree construction and
// for tests that must build a deliberately corrupt tree to exercise the
// traversal cycle guards. Ordinary code MUST use
// AddChild/AddSibling/UnlinkNode instead.
func unsafeSetNextSibling(n Node, next Node) {
	n.baseDocNode().next = next
}

// noteOrphanedChildClaim records an off-chain parent claim the GUARDED paths
// themselves create, which is how an ordinary caller reaches one. Two shapes
// produce one. A parent holding a firstChild with NO lastChild is a shape
// Document.stringToNodeList leaves behind on an entity referenced from an
// attribute value: an append onto such a parent takes the empty-parent branch,
// which overwrites firstChild, so the child that was there is detached from the
// chain while it goes on claiming this parent, and a later append THROUGH that
// detached child records its own result as the parent's lastChild, moving that
// record off the child list. A parent whose child list is EMPTY cannot be
// listing the anchor an append or a splice runs through, so the node that
// operation links claims that parent from lastChild alone (addSibling's tail
// arm, replaceNode's splice).
//
// Record it at the moment the claim is created, on the documents that own the
// parent whose chain is at stake and the node making the claim. orphan is that
// node — the parent's firstChild before the overwrite, or the node being linked
// — and a nil one means there is no claim to record.
func noteOrphanedChildClaim(parent Node, orphan Node) {
	if isNilNode(orphan) {
		return
	}
	attributed := markOffChainClaim(owningDocument(parent))
	attributed = markOffChainClaim(owningDocument(orphan)) && attributed
	if !attributed {
		unownedOffChainClaim.Store(true)
	}
}

// markOffChainClaim records an off-chain parent claim on doc and reports whether
// there was a document to record it on.
func markOffChainClaim(doc *Document) bool {
	if doc == nil {
		return false
	}
	doc.offChainClaims = true
	return true
}

// unownedOffChainClaim records an off-chain parent claim that no document owned
// at the time it was made. It is package-level because there is nowhere else to
// put it: a detached node carries no document, and the claim is still there when
// a document later adopts the subtree. It is atomic because the documents that
// read it are otherwise independent and may live in different goroutines.
var unownedOffChainClaim atomic.Bool

// adoptOffChainClaims carries an off-chain-claim record forward when a node
// changes owning document. The record lives on the DOCUMENT, so a subtree that
// moves between documents would otherwise leave it behind: the destination would
// trust a lastChild that is not the end of its own child chain. A node arriving
// from NO document is one case unownedOffChainClaim exists for — the claim had
// no document to be recorded on when it was made — and a node LEAVING every
// document is the other, since the record would otherwise be dropped.
//
// A node leaving every document is the mirror case: the record has nowhere
// document-shaped to live, so it lands on unownedOffChainClaim, which is what a
// document-less node reads.
//
// It errs toward marking. Marking a document that holds no claim only costs it
// the O(1) append-point resolution and the cycle guard's claimant search, both
// of which fall back to a walk every other tree operation performs
// unconditionally; failing to mark one that does hold a claim is a wrong tree
// or an accepted cycle.
func adoptOffChainClaims(from, to *Document) {
	if to == nil {
		// The node is leaving every document, so there is no document left to
		// carry the record. Mirror it onto the package-level one, which is what
		// a document-less node reads (holdsOffChainClaim); dropping it here
		// would leave the claimed node looking unclaimed.
		if from != nil && from.offChainClaims {
			unownedOffChainClaim.Store(true)
		}
		return
	}
	if to.offChainClaims || from == to {
		return
	}
	if from == nil {
		if unownedOffChainClaim.Load() {
			to.offChainClaims = true
		}
		return
	}
	if from.offChainClaims {
		to.offChainClaims = true
	}
}

// owningDocument returns the document whose trees n belongs to. A *Document is
// its OWN owner — a document node holds the trees it owns and its embedded
// docnode's doc pointer stays nil — so it must be resolved by type rather than
// by reading that pointer.
func owningDocument(n Node) *Document {
	if doc, ok := n.(*Document); ok {
		return doc
	}
	return n.baseDocNode().doc
}

func init() {
	nodelink.CorruptSelfNextSibling = nodelinkCorruptSelfNextSibling
	nodelink.CorruptTypedNilNextSibling = nodelinkCorruptTypedNilNextSibling
}

// nodelinkCorruptSelfNextSibling adapts unsafeSetNextSibling to the untyped
// internal/nodelink hook, writing n.next = n. It exists only for this module's
// corrupt-tree test fixtures; see the hook's own documentation.
func nodelinkCorruptSelfNextSibling(n any) {
	node, ok := n.(Node)
	if !ok {
		return
	}
	unsafeSetNextSibling(node, node)
}

// nodelinkCorruptTypedNilNextSibling adapts unsafeSetNextSibling to the untyped
// internal/nodelink hook, writing n.next = a typed-nil *Element. It exists only
// for this module's corrupt-tree test fixtures; see the hook's own
// documentation.
func nodelinkCorruptTypedNilNextSibling(n any) {
	node, ok := n.(Node)
	if !ok {
		return
	}
	var nilElem *Element
	unsafeSetNextSibling(node, nilElem)
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
	// attribute with an *Element parent is not guaranteed to be a property: a
	// generic Replace(attr) that swaps a child node for an attribute can place it
	// in the normal child list instead. Confirm it is actually reachable from
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
	// element's properties chain: a generic Replace(attr) that swaps a child node
	// for an attribute can place it in the normal child list instead. Only use
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
			// A parent that lists NO children cannot be listing n, so every
			// replacement names a parent reachable only through lastChild. Record
			// that off-chain claim where it is created, for the same reason
			// addSibling does: the cycle guard's claimant search must decline a
			// parent whose claimants it can no longer enumerate.
			if pdn.firstChild == nil {
				noteOrphanedChildClaim(parent, cur)
			}
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
	// directly, bypassing the MutableNode setters, so a non-MutableNode
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

// RemoveNamespaceByPrefix removes a namespace declaration (nsDefs entry) with the
// given prefix. Returns true if a declaration was removed.
//
// This drops ONLY the nsDefs entry. It does not clear the prefix's use by this
// element's active namespace (n.ns) or a prefixed attribute, so it is not on its
// own enough to rebind an in-use prefix: a subsequent DeclareNamespace/
// AddNamespaceDecl at a different URI still (correctly) rejects while that use
// remains. A caller that rebinds an in-use prefix must also reassign the active
// namespace (SetActiveNamespace/SetNamespace) and any prefixed attribute.
func (n *node) RemoveNamespaceByPrefix(prefix string) bool {
	for i, ns := range n.nsDefs {
		if ns.Prefix() == prefix {
			n.nsDefs = append(n.nsDefs[:i], n.nsDefs[i+1:]...)
			return true
		}
	}
	return false
}

// prefixConflictsInUse reports whether prefix currently qualifies this element's
// own name (its active namespace n.ns) or any of its non-empty-prefix attributes
// at a URI DIFFERENT from uri. Such a use means declaring prefix→uri in nsDefs
// would make the serializer emit two xmlns:prefix — one from the nsDefs entry
// (dumpNsList) and one reconciled for the active/attr namespace (reconcileOne) —
// so the result is a genuine conflict. A use at the SAME uri is not a conflict:
// the reconciler finds the prefix already bound to that URI and synthesizes
// nothing.
//
// An attribute with an EMPTY prefix never uses the default namespace (per XML,
// an unprefixed attribute name is in no namespace), and the serializer skips its
// namespace entirely (reconcileOne returns early for prefix=="" && !isElement),
// so it can never produce a second xmlns declaration and is not counted as a
// conflict for the empty/default prefix. The element's own name via n.ns with an
// empty prefix (a real default-namespace name) still counts.
//
// The attribute chain is walked with a per-list seen guard, mirroring the
// serializer, so a corrupt chain cannot loop.
//
// The conflict check is element-scoped: only an element node serializes n.ns as
// its own name and carries prefixed attributes, so on any non-element node
// (Text, Comment, CDATASection, …) n.ns is never emitted and there is nothing that could
// produce a second xmlns:prefix — no conflict is possible.
func (n *node) prefixConflictsInUse(prefix, uri string) bool {
	if n.etype != ElementNode {
		return false
	}
	if n.ns != nil && n.ns.Prefix() == prefix && n.ns.URI() != uri {
		return true
	}
	seen := make(map[*docnode]struct{})
	for attr := n.properties; attr != nil; attr = attr.NextAttribute() {
		key := attr.baseDocNode()
		if _, dup := seen[key]; dup {
			break
		}
		seen[key] = struct{}{}
		ans := attr.ns
		if ans == nil || ans.Prefix() == "" {
			continue
		}
		if ans.Prefix() == prefix && ans.URI() != uri {
			return true
		}
	}
	return false
}

// DeclareNamespace declares a namespace on this node without making it the
// node's active namespace (libxml2: xmlNewNs). It never changes the node's
// active namespace (n.ns) or its expanded name, and does not itself add a second
// declaration for a prefix in nsDefs:
//
//   - prefix is in use by this ELEMENT's own name or a non-empty-prefix
//     attribute at a URI DIFFERENT from uri: a genuine conflict; rejected with a
//     %w-wrapped ErrInvalidOperation and the tree left unchanged. This holds
//     whether or not an nsDefs entry already exists — declaring prefix→uri while
//     the active ns or such an attribute uses prefix→otherURI would make the
//     serializer emit two xmlns:prefix. The conflict check is element-scoped:
//     on a non-element node (Text, Comment, CDATASection, …) n.ns is never serialized, so
//     there is no conflict and the declaration proceeds to the dedup step below.
//     A caller that genuinely rebinds an in-use prefix must first clear the use
//     itself — reassign the element's active namespace (SetActiveNamespace/SetNamespace)
//     and any prefixed attribute; RemoveNamespaceByPrefix alone drops only the
//     nsDefs entry, not the n.ns/attribute use, so a rebind still (correctly)
//     rejects while the use remains.
//   - Otherwise: no existing declaration for prefix → appended; an existing
//     declaration with the same URI → idempotent no-op; an existing declaration
//     with a different URI → the single nsDefs slot is replaced in place
//     (collapse) with a fresh Namespace (the old object is left unmutated, so any
//     n.ns/attr.ns that aliases it is unaffected).
//
// This method does NOT reconcile a conflict introduced AFTER declaration by
// SetActiveNamespace/SetNamespace, which set the node's active namespace independently
// and may bind prefix to a different URI than an nsDefs entry. Guaranteeing at
// most one xmlns:prefix per element across all mutators is a serializer-level
// concern, outside this method's scope.
func (n *node) DeclareNamespace(prefix, uri string) error {
	if n.prefixConflictsInUse(prefix, uri) {
		return fmt.Errorf("cannot rebind namespace prefix %q while it is in use on this element: %w", prefix, ErrInvalidOperation)
	}
	for i, ns := range n.nsDefs {
		if ns == nil {
			continue
		}
		if ns.Prefix() != prefix {
			continue
		}
		if ns.URI() == uri {
			return nil
		}
		fresh, err := n.doc.CreateNamespace(prefix, uri)
		if err != nil {
			return err
		}
		n.nsDefs[i] = fresh
		return nil
	}
	ns, err := n.doc.CreateNamespace(prefix, uri)
	if err != nil {
		return err
	}
	n.nsDefs = append(n.nsDefs, ns)
	return nil
}

// AddNamespaceDecl attaches an existing Namespace to this node's declarations
// (nsDefs) without allocating a new one, so a caller building a tree can reuse
// one Namespace object as both a declaration and an element's active namespace.
// Like DeclareNamespace, it does not itself add a second declaration for a
// prefix, using ns's prefix and URI, and reports the same outcomes:
//
//   - ns is nil: ErrNilNode is returned and nsDefs is left unchanged.
//   - ns's prefix is in use by the ELEMENT's name or a non-empty-prefix
//     attribute at a URI DIFFERENT from ns's URI: a genuine conflict; ns is not
//     installed, the tree is left unchanged, and the same %w-wrapped
//     ErrInvalidOperation DeclareNamespace produces is returned (matchable with
//     errors.Is). This holds whether or not an nsDefs entry already exists. The
//     conflict check is element-scoped: on a non-element node n.ns is never
//     serialized, so there is no conflict and the call proceeds to the dedup step
//     below (a same-URI redeclare is still a no-op there).
//   - Otherwise nil is returned: no existing declaration for ns's prefix → ns is
//     appended; an existing declaration with the same URI → no-op (the existing
//     slot is kept; ns is not installed); an existing declaration with a
//     different URI → the slot is replaced in place (collapse) with ns (the
//     previously-installed object is dropped but never mutated, so a caller still
//     holding it is unaffected).
//
// Like DeclareNamespace, this does NOT reconcile a conflict introduced AFTER
// declaration by SetActiveNamespace/SetNamespace; keeping at most one xmlns:prefix per
// element across all mutators is a serializer-level concern outside this method's
// scope.
//
// The caller owns ns; it must not be shared as a declaration across nodes that
// could be mutated independently.
//
// When ns is RETAINED (case 1 append and case 3 collapse) and is slab-backed by
// a DIFFERENT document than this node's owner, that source document is marked so
// its Free will not recycle the namespace slab out from under the retained
// declaration — the same guard a cross-document node move applies (see
// noteCrossDocumentEscape). A same-document or heap-allocated ns, and the cases
// that do not retain ns (a same-URI no-op, a declined conflict), mark nothing.
func (n *node) AddNamespaceDecl(ns *Namespace) error {
	if ns == nil {
		return ErrNilNode
	}
	prefix := ns.Prefix()
	if n.prefixConflictsInUse(prefix, ns.URI()) {
		return fmt.Errorf("cannot rebind namespace prefix %q while it is in use on this element: %w", prefix, ErrInvalidOperation)
	}
	for i, existing := range n.nsDefs {
		if existing == nil {
			continue
		}
		if existing.Prefix() != prefix {
			continue
		}
		if existing.URI() == ns.URI() {
			return nil
		}
		// Case 3 (collapse): the caller's ns replaces the slot and is retained. If
		// ns is slab-backed by a DIFFERENT document, guard that document's Free
		// against recycling the chunk out from under this retained declaration,
		// mirroring a cross-document node move.
		noteCrossDocumentNamespaceEscape(n.doc, ns)
		n.nsDefs[i] = ns
		return nil
	}
	// Case 1 (append): the caller's ns is retained in nsDefs; guard a foreign
	// source document's slab the same way.
	noteCrossDocumentNamespaceEscape(n.doc, ns)
	n.nsDefs = append(n.nsDefs, ns)
	return nil
}

// SetActiveNamespace declares a namespace and sets it as this node's active
// namespace.
func (n *node) SetActiveNamespace(prefix, uri string) error {
	ns, err := n.doc.CreateNamespace(prefix, uri)
	if err != nil {
		return err
	}
	n.ns = ns
	n.invalidateQName()
	return nil
}

// SetNamespace sets the node's active namespace to an existing Namespace object
// without creating a new declaration.
//
// When ns is slab-backed by a DIFFERENT document than this node's owner, that
// source document is marked so its Free will not recycle the namespace slab out
// from under the retained active namespace — the same guard AddNamespaceDecl and
// a cross-document node move apply (see noteCrossDocumentNamespaceEscape). A nil,
// heap-allocated, or same-document ns marks nothing.
func (n *node) SetNamespace(ns *Namespace) {
	noteCrossDocumentNamespaceEscape(n.doc, ns)
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
		adoptOffChainClaims(cdn.doc, doc)
		cdn.doc = doc
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
			adoptOffChainClaims(prop.doc, doc)
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
