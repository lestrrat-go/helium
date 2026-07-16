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
// raw SetNextSibling/SetPrevSibling link setters — terminates the iteration
// instead of looping forever, yielding the partial set gathered up to that
// point. A range-over-func iterator has no error channel, so this truncation is
// silent; to DETECT a cycle rather than silently stop at it, traverse with
// [Walk], which returns [ErrWalkCycle].
func Children(n Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		if n == nil {
			return
		}
		owner := n.baseDocNode()
		seen := make(map[*docnode]struct{})
		for child := n.FirstChild(); child != nil; child = nextOwnedChild(owner, child) {
			cdn := child.baseDocNode()
			if _, dup := seen[cdn]; dup {
				return
			}
			seen[cdn] = struct{}{}
			if !yield(child) {
				return
			}
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
// point. A range-over-func iterator has no error channel, so this truncation is
// silent; to DETECT a cycle rather than silently stop at it, traverse with
// [Walk], which returns [ErrWalkCycle]. A shared DAG node reached on a different
// path is not on the descent path and is still visited on each occurrence, so
// DAG traversal is unchanged.
func Descendants(n Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		if n == nil {
			return
		}
		onPath := make(map[*docnode]struct{})
		var walk func(Node) bool
		walk = func(parent Node) bool {
			pdn := parent.baseDocNode()
			onPath[pdn] = struct{}{}
			defer delete(onPath, pdn)
			seen := make(map[*docnode]struct{})
			for child := parent.FirstChild(); child != nil; child = nextOwnedChild(pdn, child) {
				cdn := child.baseDocNode()
				if _, dup := seen[cdn]; dup {
					return true
				}
				seen[cdn] = struct{}{}
				if !yield(child) {
					return false
				}
				if _, cyclic := onPath[cdn]; cyclic {
					continue
				}
				if !walk(child) {
					return false
				}
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
// partial set gathered up to that point. A range-over-func iterator has no error
// channel, so this truncation is silent; to DETECT a cycle rather than silently
// stop at it, traverse with [Walk], which returns [ErrWalkCycle].
func ChildElements(n Node) iter.Seq[*Element] {
	return func(yield func(*Element) bool) {
		if n == nil {
			return
		}
		owner := n.baseDocNode()
		seen := make(map[*docnode]struct{})
		for child := n.FirstChild(); child != nil; child = nextOwnedChild(owner, child) {
			cdn := child.baseDocNode()
			if _, dup := seen[cdn]; dup {
				return
			}
			seen[cdn] = struct{}{}
			if elem, ok := AsNode[*Element](child); ok {
				if !yield(elem) {
					return
				}
			}
		}
	}
}
