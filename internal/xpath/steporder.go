package xpath

import (
	"slices"

	helium "github.com/lestrrat-go/helium"
)

// maxSameDepth bounds the parent-chain walk sameDepth performs on the first
// input node. Deeper inputs take the DeduplicateNodes path.
const maxSameDepth = 64

// OrderStepResult returns the result of one location step sorted in document
// order and free of duplicates, the same node-set and order DeduplicateNodes
// produces, without building the whole-document order index when the shape
// of the step already proves the order.
//
// out is the concatenation, in traversal order, of the axis result (after
// predicates) for each node of inputs. inputs MUST be sorted in document order
// and duplicate-free: the context node-set of a step is either the single
// start node of a location path or the output of the previous step, so the
// property holds by induction. The first rule that fits applies:
//
//  1. len(out) <= 1: out is returned as is, as DeduplicateNodes does.
//  2. len(inputs) <= 1: one axis traversal yields distinct nodes, in document
//     order for the forward axes and in reverse document order for ancestor,
//     ancestor-or-self, preceding and preceding-sibling (see axes.go). The
//     reverse axes are reversed in place. Predicates only remove nodes, so the
//     property survives them. The namespace nodes of one element share an
//     index position, and the stable sort keeps them in traversal order, so
//     returning them as is matches too.
//  3. axis is child, attribute, self, namespace or parent and every input
//     sits at the same depth: sorted distinct nodes of equal depth have
//     disjoint subtrees in document order (a subtree being the node, its
//     attribute and namespace nodes and its descendants), so the per-input
//     child, attribute, namespace and self results are disjoint and in order.
//     The parents of such inputs are non-decreasing, so duplicates on the
//     parent axis are adjacent and slices.Compact removes them.
//  4. Otherwise it returns DeduplicateNodes(out, cache, maxNodes).
//
// Rules 2 and 3 enforce maxNodes and clamp the capacity of the returned slice
// as DeduplicateNodes does. They do not index the document, but they reserve
// its registration order in cache (reserveDocument) at the point where
// DeduplicateNodes would have indexed it, so a later merge orders nodes from
// different documents exactly as before. Every node of out belongs to one
// tree: a step traverses from nodes of a single location path.
func OrderStepResult(out, inputs []helium.Node, axis AxisType, cache *DocOrderCache, maxNodes int) ([]helium.Node, error) {
	if len(out) <= 1 {
		return out, nil
	}
	if len(inputs) <= 1 {
		if isReverseAxis(axis) {
			slices.Reverse(out)
		}
		cache.reserveDocument(out[0])
		return clampStepResult(out, maxNodes)
	}
	switch axis {
	case AxisChild, AxisAttribute, AxisSelf, AxisNamespace, AxisParent:
		if !sameDepth(inputs) {
			break
		}
		if axis == AxisParent {
			out = slices.Compact(out)
		}
		cache.reserveDocument(out[0])
		return clampStepResult(out, maxNodes)
	}
	return DeduplicateNodes(out, cache, maxNodes)
}

// clampStepResult applies the node-set limit and returns out with its
// capacity clamped, so the result never shares spare capacity with the
// caller's buffer.
func clampStepResult(out []helium.Node, maxNodes int) ([]helium.Node, error) {
	if len(out) > maxNodes {
		return nil, ErrNodeSetLimit
	}
	return out[:len(out):len(out)], nil
}

// isReverseAxis reports whether axis traverses in reverse document order.
func isReverseAxis(axis AxisType) bool {
	switch axis {
	case AxisAncestor, AxisAncestorOrSelf, AxisPreceding, AxisPrecedingSibling:
		return true
	}
	return false
}

// sameDepth reports whether every node in nodes has the same number of
// ancestors. It gives up (false) when the first node is deeper than
// maxSameDepth, and stops at the first node whose depth differs.
func sameDepth(nodes []helium.Node) bool {
	depth := 0
	for p := nodes[0].Parent(); p != nil; p = p.Parent() {
		depth++
		if depth > maxSameDepth {
			return false
		}
	}
	for _, n := range nodes[1:] {
		p := n
		for range depth {
			p = p.Parent()
			if p == nil {
				return false
			}
		}
		if p.Parent() != nil {
			return false
		}
	}
	return true
}
