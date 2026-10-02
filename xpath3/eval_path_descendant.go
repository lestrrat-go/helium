package xpath3

import (
	"context"
	"slices"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// This file evaluates the abbreviation `//` followed by a child or attribute
// step (`//x`, `//x[p]`, `//@a`, `a//x`) as one walk of each context node's
// subtree.
//
// `//` expands to a descendant-or-self::node() step, so `//x` is two steps:
// the whole subtree, then the x children of each of its nodes. The second
// step runs from context nodes at many depths, so its result has to be sorted
// into document order, which builds the whole-document order index. The walk
// in this file visits each subtree in pre-order instead and emits each result
// node when it is reached, so the result comes out in document order:
//
//   - a child of a subtree node is emitted when the walk visits it;
//   - the attributes of a subtree node are emitted when the walk visits their
//     element, before its children, which is where the index numbers them.
//
// The predicates of the second step still see one parent's candidate list
// each, so position() and last() keep their meaning. They are evaluated in the
// order the two-step evaluation evaluates them: parent by parent, in document
// order.
//
// The fused walk charges what the two steps charge and fails where they fail:
//
//   - the descendant-or-self step walks each context node's subtree in turn,
//     failing with ErrNodeSetLimit when one subtree holds more than maxNodes
//     nodes and charging one operation per subtree node after each subtree;
//     it then fails with ErrNodeSetLimit when the subtrees hold more than
//     maxNodes nodes together, and registers the document in the order cache
//     when they hold more than one;
//   - the second step charges, for every subtree node, the candidates it
//     enumerates (XDM children or attributes), then applies the predicates.
//
// When the second step has predicates, a first pass counts the subtrees, so
// the node-set limits, the first charges and the registration happen before
// any predicate runs, as in the two-step evaluation. Without predicates
// nothing can fail or observe the order of the second step's charges, so one
// pass does both.
//
// The fusion needs context nodes whose subtrees are disjoint and in document
// order, and whose pre-order walks match the order index
// (fusesDescendantContexts); any other context list evaluates the two steps
// one by one.

// fusesDescendantStep reports whether steps[i] is descendant-or-self::node()
// without predicates and steps[i+1] is a child or attribute step.
func fusesDescendantStep(steps []vmLocationStep, i int) bool {
	if i+1 >= len(steps) {
		return false
	}
	step := steps[i]
	if step.Axis != AxisDescendantOrSelf || len(step.Predicates) != 0 {
		return false
	}
	tt, ok := step.NodeTest.(TypeTest)
	if !ok || tt.Kind != NodeKindNode {
		return false
	}
	switch steps[i+1].Axis {
	case AxisChild, AxisAttribute:
		return true
	}
	return false
}

// fusesDescendantContexts reports whether the fused walk can run from the
// context list nodes, which is sorted in document order and duplicate-free.
// Every node must pass ixpath.OrderedFrom. With more than one node, none may
// be an attribute or namespace node (whose position sits between an element
// and its children) and none may lie in the subtree of another, so the
// subtrees are disjoint and follow each other in document order.
func fusesDescendantContexts(nodes []helium.Node) bool {
	if len(nodes) == 0 {
		return false
	}
	if len(nodes) == 1 {
		return ixpath.OrderedFrom(nodes[0])
	}
	// top is the last node that no earlier node contains. Sorted nodes put a
	// node inside an earlier subtree only inside the subtree of top: an
	// earlier subtree that held it would hold top as well.
	var top, prevParent helium.Node
	for i, n := range nodes {
		switch n.Type() {
		case helium.AttributeNode, helium.NamespaceNode:
			return false
		}
		if !ixpath.OrderedFrom(n) {
			return false
		}
		parent := n.Parent()
		// A sibling of the previous node is not inside top: top would be an
		// ancestor of their parent, and so of the previous node too.
		if i > 0 && (parent == nil || parent != prevParent) && hasAncestor(n, top) {
			return false
		}
		top = n
		prevParent = parent
	}
	return true
}

// hasAncestor reports whether a is an ancestor of n.
func hasAncestor(n, a helium.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p == a {
			return true
		}
	}
	return false
}

// evalVMDescendantStep evaluates descendant-or-self::node()/step from the
// context list contexts, where step is a child or attribute step. contexts
// must pass fusesDescendantContexts.
func evalVMDescendantStep(evalFn exprEvaluator, ctx context.Context, ec *evalContext, contexts []helium.Node, step vmLocationStep) ([]helium.Node, error) {
	if len(step.Predicates) == 0 {
		return descendantStepNoPredicates(ctx, ec, contexts, step)
	}
	return descendantStepWithPredicates(evalFn, ctx, ec, contexts, step)
}

// descendantStepNoPredicates walks the subtree of every context node once,
// collecting the step's matches in document order.
func descendantStepNoPredicates(ctx context.Context, ec *evalContext, contexts []helium.Node, step vmLocationStep) ([]helium.Node, error) {
	var out []helium.Node
	total := 0
	// candidates counts what the second step enumerates. On the child axis
	// it is every subtree node but the context nodes.
	candidates := 0
	for _, c := range contexts {
		var visited, traversed int
		var err error
		out, visited, traversed, err = appendDescendantMatches(ctx, ec, out, c, step)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(ctx, visited); err != nil {
			return nil, err
		}
		total += visited
		candidates += traversed
	}
	if err := finishDescendantOrSelfStep(ec, contexts[0], total); err != nil {
		return nil, err
	}
	if err := ec.countOps(ctx, candidates); err != nil {
		return nil, err
	}
	return clampDescendantResult(ec, out)
}

// appendDescendantMatches walks the subtree of c, appending the matches of
// step to out. It returns the number of subtree nodes and the number of
// candidates the step enumerates from them.
func appendDescendantMatches(ctx context.Context, ec *evalContext, out []helium.Node, c helium.Node, step vmLocationStep) ([]helium.Node, int, int, error) {
	stack, err := startDescendantWalk(ctx, ec, c)
	if err != nil {
		return nil, 0, 0, err
	}
	visited := 1
	candidates := 0
	if step.Axis == AxisAttribute {
		var traversed int
		out, traversed, err = appendAxisNodeMatches(ctx, out, ec, c, AxisAttribute, step.NodeTest)
		if err != nil {
			return nil, 0, 0, err
		}
		candidates += traversed
	}
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, 0, 0, err
		}
		last := len(stack) - 1
		cur := stack[last]
		stack = stack[:last]
		visited++
		if visited > ec.maxNodes {
			return nil, 0, 0, ixpath.ErrNodeSetLimit
		}
		if step.Axis == AxisChild {
			candidates++
			if matchNodeTest(step.NodeTest, cur, AxisChild, ec) {
				out = append(out, cur)
			}
		} else {
			var traversed int
			out, traversed, err = appendAxisNodeMatches(ctx, out, ec, cur, AxisAttribute, step.NodeTest)
			if err != nil {
				return nil, 0, 0, err
			}
			candidates += traversed
		}
		stack, err = ixpath.PushXDMChildren(ctx, stack, cur)
		if err != nil {
			return nil, 0, 0, err
		}
	}
	return out, visited, candidates, nil
}

// descendantEntry is a node waiting on the pre-order walk stack. selected
// records whether the child step of its parent kept it.
type descendantEntry struct {
	node     helium.Node
	selected bool
}

// descendantStepWithPredicates counts the subtree of every context node and
// charges the descendant-or-self step, then walks the subtrees again,
// applying the second step and its predicates to every subtree node in
// document order.
func descendantStepWithPredicates(evalFn exprEvaluator, ctx context.Context, ec *evalContext, contexts []helium.Node, step vmLocationStep) ([]helium.Node, error) {
	total := 0
	for _, c := range contexts {
		visited, err := countDescendantOrSelf(ctx, ec, c)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(ctx, visited); err != nil {
			return nil, err
		}
		total += visited
	}
	if err := finishDescendantOrSelfStep(ec, contexts[0], total); err != nil {
		return nil, err
	}

	var out, matched []helium.Node
	var stack []descendantEntry
	for _, c := range contexts {
		stack = append(stack, descendantEntry{node: c})
		var err error
		for len(stack) > 0 {
			last := len(stack) - 1
			cur := stack[last]
			stack = stack[:last]
			if cur.selected {
				out = append(out, cur.node)
			}

			start := len(stack)
			var traversed int
			if step.Axis == AxisChild {
				stack, matched, traversed, err = pushMatchingChildEntries(ctx, ec, stack, matched[:0], cur.node, step.NodeTest)
				if err != nil {
					return nil, err
				}
			} else {
				matched, traversed, err = appendAxisNodeMatches(ctx, matched[:0], ec, cur.node, AxisAttribute, step.NodeTest)
				if err != nil {
					return nil, err
				}
				stack, err = pushChildEntries(ctx, stack, cur.node)
				if err != nil {
					return nil, err
				}
			}
			if err := ec.countOps(ctx, traversed); err != nil {
				return nil, err
			}
			selected := matched
			for _, pred := range step.Predicates {
				selected, err = applyVMPredicate(evalFn, ctx, ec, selected, pred)
				if err != nil {
					return nil, err
				}
			}
			if step.Axis == AxisChild {
				markSelectedEntries(stack[start:], selected)
			} else {
				out = append(out, selected...)
			}
			slices.Reverse(stack[start:])
		}
	}
	return clampDescendantResult(ec, out)
}

// startDescendantWalk begins the descendant-or-self walk from c the way
// ixpath.TraverseAxis does: it checks ctx, counts c against the node-set
// limit, and returns the stack of c's XDM children. An attribute has none.
func startDescendantWalk(ctx context.Context, ec *evalContext, c helium.Node) ([]helium.Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if ec.maxNodes < 1 {
		return nil, ixpath.ErrNodeSetLimit
	}
	if _, ok := c.(*helium.Attribute); ok {
		return nil, nil
	}
	return ixpath.PushXDMChildren(ctx, nil, c)
}

// countDescendantOrSelf returns the size of the descendant-or-self axis of c,
// failing where ixpath.TraverseAxis fails on that axis.
func countDescendantOrSelf(ctx context.Context, ec *evalContext, c helium.Node) (int, error) {
	stack, err := startDescendantWalk(ctx, ec, c)
	if err != nil {
		return 0, err
	}
	visited := 1
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		last := len(stack) - 1
		cur := stack[last]
		stack = stack[:last]
		visited++
		if visited > ec.maxNodes {
			return 0, ixpath.ErrNodeSetLimit
		}
		stack, err = ixpath.PushXDMChildren(ctx, stack, cur)
		if err != nil {
			return 0, err
		}
	}
	return visited, nil
}

// finishDescendantOrSelfStep ends the descendant-or-self::node() step whose
// result has total nodes the way ixpath.OrderStepResult ends it: a result of
// more than maxNodes nodes fails, and a result of more than one node
// registers the document of c in the order cache.
func finishDescendantOrSelfStep(ec *evalContext, c helium.Node, total int) error {
	if total > ec.maxNodes {
		return ixpath.ErrNodeSetLimit
	}
	if total > 1 {
		ec.docOrder.ReserveDocument(c)
	}
	return nil
}

// clampDescendantResult ends the second step the way ixpath.OrderStepResult
// ends a step whose result is already in document order: a result of more
// than maxNodes nodes fails, and the capacity of a longer-than-one result is
// clamped.
func clampDescendantResult(ec *evalContext, out []helium.Node) ([]helium.Node, error) {
	if len(out) <= 1 {
		return out, nil
	}
	if len(out) > ec.maxNodes {
		return nil, ixpath.ErrNodeSetLimit
	}
	return out[:len(out):len(out)], nil
}

// pushMatchingChildEntries pushes the XDM children of n onto stack in
// document order and appends the ones that match test to matched,
// enumerating them as the child step does (appendAxisNodeMatches). It returns
// the grown stack and matched slices and the number of children enumerated.
func pushMatchingChildEntries(ctx context.Context, ec *evalContext, stack []descendantEntry, matched []helium.Node, n helium.Node, test NodeTest) ([]descendantEntry, []helium.Node, int, error) {
	if _, ok := n.(*helium.Attribute); ok {
		return stack, matched, 0, nil
	}
	traversed := 0
	for child := range helium.Children(n) {
		if err := ctx.Err(); err != nil {
			return nil, nil, 0, err
		}
		if !ixpath.IsXDMChild(child) {
			continue
		}
		traversed++
		stack = append(stack, descendantEntry{node: child})
		if matchNodeTest(test, child, AxisChild, ec) {
			matched = append(matched, child)
		}
	}
	return stack, matched, traversed, nil
}

// pushChildEntries pushes the XDM children of n onto stack in document
// order.
func pushChildEntries(ctx context.Context, stack []descendantEntry, n helium.Node) ([]descendantEntry, error) {
	if _, ok := n.(*helium.Attribute); ok {
		return stack, nil
	}
	for child := range helium.Children(n) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ixpath.IsXDMChild(child) {
			stack = append(stack, descendantEntry{node: child})
		}
	}
	return stack, nil
}

// markSelectedEntries flags the entries whose node is in selected. Both lists
// are in document order and selected is a subsequence of the entries.
func markSelectedEntries(entries []descendantEntry, selected []helium.Node) {
	k := 0
	for i := range entries {
		if k == len(selected) {
			return
		}
		if entries[i].node == selected[k] {
			entries[i].selected = true
			k++
		}
	}
}
