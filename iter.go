package helium

import "iter"

// Children returns an iterator over the direct children of n.
// If n is nil or has no children, the iterator yields nothing.
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
// instructions. If n is nil or has no element children, the iterator
// yields nothing.
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
