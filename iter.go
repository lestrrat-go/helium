package helium

import "iter"

// Children returns an iterator over all direct child nodes of n,
// including elements, text, comments, processing instructions, and any
// other node types. Use Children when you need to inspect or process
// every node in the child list regardless of type.
//
// To iterate over child elements only, use [ChildElements] instead.
//
// If n is nil or has no children, the iterator yields nothing.
//
// The caller must not modify the tree structure (add, remove, or reorder
// nodes) during iteration. Doing so may cause nodes to be skipped or
// visited more than once.
//
// Children advances between siblings using the owned-boundary rule: a child
// whose Parent() is not n (an entity reference's shared Entity child, owned by
// the DTD, whose sibling pointers belong to another list) ends the iteration,
// so Children never spills into another list's siblings. A cyclic sibling
// pointer — reachable only on a corrupt or hand-built graph, e.g. through the
// raw [UnsafeSetNextSibling] link setter — terminates
// the iteration instead of looping forever, yielding the partial set gathered up
// to that point. The guard is [siblingCycleGuard], so a well-formed list pays no
// allocation and no per-child bookkeeping; a cyclic one stops within a bounded
// multiple of the cycle length and may therefore yield some of its nodes more
// than once first. A range-over-func iterator has no error channel, so this
// truncation is silent; to DETECT a cycle instead of silently stopping at it,
// traverse with [Walk], which returns [ErrWalkCycle].
func Children(n Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		if isNilNode(n) {
			return
		}
		owner := n.baseDocNode()
		var g siblingCycleGuard
		for child := owner.firstChild; child != nil; {
			cdn := child.baseDocNode()
			if g.step(cdn) {
				return
			}
			if !yield(child) {
				return
			}
			child = nextOwnedSibling(owner, cdn)
		}
	}
}

// Descendants returns an iterator that performs a depth-first pre-order
// traversal of all descendants of n (not including n itself).
// If n is nil or has no children, the iterator yields nothing.
//
// The caller must not modify the tree structure (add, remove, or reorder
// nodes) during iteration. Doing so may cause nodes to be skipped or
// visited more than once.
//
// Descendants is cycle-safe on hand-built or foreign-linked graphs. It advances
// between siblings using the owned-boundary rule (a child owned by another node,
// such as an entity reference's shared Entity child, ends its sibling list) and
// carries the set of nodes on the current descent path: it visits a back-edge
// node but does not descend through it, so a child-pointer cycle terminates
// cleanly instead of looping, yielding the partial set gathered up to that
// point. Each sibling list is bounded by [siblingCycleGuard], with the same
// allocation-free common path and the same bounded overshoot on a cyclic list
// that [Children] documents. A range-over-func iterator has no error channel, so
// this truncation is silent; to DETECT a cycle instead of silently stopping at
// it, traverse with [Walk], which returns [ErrWalkCycle]. A shared DAG node
// reached on a different path is not on the descent path and is still visited on
// each occurrence, so DAG traversal is unchanged.
func Descendants(n Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		if isNilNode(n) {
			return
		}
		onPath := make(map[*docnode]struct{})
		var walk func(Node) bool
		walk = func(parent Node) bool {
			pdn := parent.baseDocNode()
			onPath[pdn] = struct{}{}
			defer delete(onPath, pdn)
			var g siblingCycleGuard
			for child := pdn.firstChild; child != nil; {
				cdn := child.baseDocNode()
				if g.step(cdn) {
					return true
				}
				if !yield(child) {
					return false
				}
				if _, cyclic := onPath[cdn]; !cyclic {
					if !walk(child) {
						return false
					}
				}
				child = nextOwnedSibling(pdn, cdn)
			}
			return true
		}
		walk(n)
	}
}

// ChildElements returns an iterator over the direct child elements of n,
// skipping non-element children such as text, comments, and processing
// instructions. Use ChildElements when you only care about element
// children and want to avoid type-checking each node yourself.
//
// To iterate over all child nodes including non-elements, use [Children]
// instead.
//
// If n is nil or has no element children, the iterator yields nothing.
//
// The caller must not modify the tree structure (add, remove, or reorder
// nodes) during iteration. Doing so may cause nodes to be skipped or
// visited more than once.
//
// Like [Children], ChildElements advances between siblings using the
// owned-boundary rule and terminates on a cyclic sibling pointer, yielding the
// partial set gathered up to that point, with the same bounded overshoot
// [Children] documents. A range-over-func iterator has no error channel, so this
// truncation is silent; to DETECT a cycle instead of silently stopping at it,
// traverse with [Walk], which returns [ErrWalkCycle].
func ChildElements(n Node) iter.Seq[*Element] {
	return func(yield func(*Element) bool) {
		if isNilNode(n) {
			return
		}
		owner := n.baseDocNode()
		var g siblingCycleGuard
		for child := owner.firstChild; child != nil; {
			cdn := child.baseDocNode()
			if g.step(cdn) {
				return
			}
			if elem, ok := AsNode[*Element](child); ok {
				if !yield(elem) {
					return
				}
			}
			child = nextOwnedSibling(owner, cdn)
		}
	}
}

// siblingCycleGuard bounds a sibling-list walk with Brent's cycle detection. A
// sibling list is a linked list a caller can corrupt into a loop through the
// Unsafe* link setters, so every enumeration of one needs a termination guard.
// Brent's algorithm needs three words and no allocation, where a seen-set needs
// a map and one insert per child on EVERY list, well-formed ones included, for a
// check that fires only on a corrupt graph.
//
// The trade is where the walk stops. A seen set stops at the exact first repeat;
// Brent stops once the chasing pointer catches the checkpoint, which is within a
// small multiple of the cycle length, so a corrupt list can yield some of its
// nodes more than once before terminating. Termination itself is unconditional,
// which is the property the traversal API actually promises.
type siblingCycleGuard struct {
	// tortoise is the checkpoint node the walk is compared against, nil until
	// the first step seeds it.
	tortoise *docnode
	// power is the current checkpoint interval and lam the number of steps
	// taken since the checkpoint moved.
	power int
	lam   int
}

// step records cur as the next node of the sibling list and reports whether the
// list has been proven cyclic, in which case the caller must stop.
func (g *siblingCycleGuard) step(cur *docnode) bool {
	if g.tortoise == nil {
		g.tortoise = cur
		g.power = 1
		return false
	}
	if g.tortoise == cur {
		return true
	}
	g.lam++
	if g.lam == g.power {
		g.power *= 2
		g.lam = 0
		g.tortoise = cur
	}
	return false
}
