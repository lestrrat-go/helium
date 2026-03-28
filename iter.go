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
func Children(n Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		if n == nil {
			return
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
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
func Descendants(n Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		if n == nil {
			return
		}
		var walk func(Node) bool
		walk = func(parent Node) bool {
			for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
				if !yield(child) {
					return false
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
func ChildElements(n Node) iter.Seq[*Element] {
	return func(yield func(*Element) bool) {
		if n == nil {
			return
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() == ElementNode {
				if !yield(child.(*Element)) {
					return
				}
			}
		}
	}
}
