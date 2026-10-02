package xpath3

import (
	"context"
	"slices"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// This file evaluates the abbreviation `//` followed by a child or attribute
// step (`//x`, `//x[p]`, `//@a`) as one walk of the context node's subtree.
//
// `//` expands to a descendant-or-self::node() step, so `//x` is two steps:
// the whole subtree, then the x children of each of its nodes. The second
// step runs from context nodes at many depths, so its result has to be sorted
// into document order, which builds the whole-document order index. The walk
// in this file visits the subtree in pre-order instead and emits each result
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
//   - the descendant-or-self step fails with ErrNodeSetLimit when the subtree
//     holds more than maxNodes nodes, then charges one operation per subtree
//     node and registers the document in the order cache (when the subtree
//     has more than one node);
//   - the second step charges, for every subtree node, the candidates it
//     enumerates (XDM children or attributes), then applies the predicates.
//
// When the second step has predicates, a first pass counts the subtree, so the
// node-set limit, the first charge and the registration happen before any
// predicate runs, as in the two-step evaluation. Without predicates nothing
// can fail or observe the order of the charges in between, so one pass does
// both.
//
// The fusion applies only from a single context node that ixpath.OrderedFrom
// accepts; any other context list evaluates the two steps one by one.

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

// evalVMDescendantStep evaluates descendant-or-self::node()/step from the
// single context node c, where step is a child or attribute step. c must pass
// ixpath.OrderedFrom.
func evalVMDescendantStep(evalFn exprEvaluator, ctx context.Context, ec *evalContext, c helium.Node, step vmLocationStep) ([]helium.Node, error) {
	if len(step.Predicates) == 0 {
		return descendantStepNoPredicates(ctx, ec, c, step)
	}
	return descendantStepWithPredicates(evalFn, ctx, ec, c, step)
}

// descendantStepNoPredicates walks the subtree of c once, collecting the
// step's matches in document order.
func descendantStepNoPredicates(ctx context.Context, ec *evalContext, c helium.Node, step vmLocationStep) ([]helium.Node, error) {
	stack, err := startDescendantWalk(ctx, ec, c)
	if err != nil {
		return nil, err
	}
	visited := 1
	var out []helium.Node
	// candidates counts what the second step enumerates. On the child axis
	// it is every subtree node but c, which the walk counts in visited.
	candidates := 0
	if step.Axis == AxisAttribute {
		var traversed int
		out, traversed, err = appendAxisNodeMatches(ctx, out, ec, c, AxisAttribute, step.NodeTest)
		if err != nil {
			return nil, err
		}
		candidates += traversed
	}
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		last := len(stack) - 1
		cur := stack[last]
		stack = stack[:last]
		visited++
		if visited > ec.maxNodes {
			return nil, ixpath.ErrNodeSetLimit
		}
		if step.Axis == AxisChild {
			if matchNodeTest(step.NodeTest, cur, AxisChild, ec) {
				out = append(out, cur)
			}
		} else {
			var traversed int
			out, traversed, err = appendAxisNodeMatches(ctx, out, ec, cur, AxisAttribute, step.NodeTest)
			if err != nil {
				return nil, err
			}
			candidates += traversed
		}
		stack, err = ixpath.PushXDMChildren(ctx, stack, cur)
		if err != nil {
			return nil, err
		}
	}
	if step.Axis == AxisChild {
		candidates = visited - 1
	}
	if err := chargeDescendantOrSelfStep(ctx, ec, c, visited); err != nil {
		return nil, err
	}
	if err := ec.countOps(ctx, candidates); err != nil {
		return nil, err
	}
	return ixpath.OrderStepResult(out, []helium.Node{c}, AxisDescendant, ec.docOrder, ec.maxNodes)
}

// descendantEntry is a node waiting on the pre-order walk stack. selected
// records whether the child step of its parent kept it.
type descendantEntry struct {
	node     helium.Node
	selected bool
}

// descendantStepWithPredicates counts the subtree of c, charges the
// descendant-or-self step, then walks the subtree again, applying the second
// step and its predicates to every subtree node in document order.
func descendantStepWithPredicates(evalFn exprEvaluator, ctx context.Context, ec *evalContext, c helium.Node, step vmLocationStep) ([]helium.Node, error) {
	visited, err := countDescendantOrSelf(ctx, ec, c)
	if err != nil {
		return nil, err
	}
	if err := chargeDescendantOrSelfStep(ctx, ec, c, visited); err != nil {
		return nil, err
	}

	stack := []descendantEntry{{node: c}}
	var out, matched []helium.Node
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
	return ixpath.OrderStepResult(out, []helium.Node{c}, AxisDescendant, ec.docOrder, ec.maxNodes)
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

// chargeDescendantOrSelfStep charges the descendant-or-self::node() step from
// c, whose result has visited nodes, and registers the document of c in the
// order cache where ixpath.OrderStepResult would.
func chargeDescendantOrSelfStep(ctx context.Context, ec *evalContext, c helium.Node, visited int) error {
	if err := ec.countOps(ctx, visited); err != nil {
		return err
	}
	if visited > 1 {
		ec.docOrder.ReserveDocument(c)
	}
	return nil
}

// pushMatchingChildEntries pushes the XDM children of n onto stack in document order
// and appends the ones that match test to matched, enumerating them as the
// child step does (appendAxisNodeMatches). It returns the grown stack and
// matched slices and the number of children enumerated.
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
