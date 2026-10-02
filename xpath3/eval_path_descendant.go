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
// A context node that lies inside the subtree of an earlier one (nested
// contexts, as in `//b//c` when b elements nest) is walked only as part of
// the outer subtree: the descendant-or-self step deduplicates its result, so
// the second step runs from every subtree node once. The walk still measures
// the nested node's subtree, and the step charges its size after the outer
// node's, as the step-by-step evaluation does when it walks the nested
// subtree again. A nested subtree is smaller than the outer one, so it cannot
// exceed maxNodes when the outer one does not.
//
// When the second step has predicates that can fail or charge operations, a
// first pass counts the subtrees, so the node-set limits, the first charges
// and the registration happen before any predicate runs, as in the two-step
// evaluation. Without predicates, or with predicates that only select
// (quietPredicates), nothing can fail or observe the order of the second
// step's charges, so one pass does both.
//
// The fusion needs context nodes in document order whose pre-order walks
// match the order index (fusesDescendantContexts); any other context list
// evaluates the two steps one by one.

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
// and its children), and a node inside the subtree of an earlier one must be
// a kind the walk of that subtree visits (ixpath.IsXDMChild).
func fusesDescendantContexts(nodes []helium.Node) bool {
	if len(nodes) == 0 {
		return false
	}
	if len(nodes) == 1 {
		return ixpath.OrderedFrom(nodes[0])
	}
	for _, n := range nodes {
		switch n.Type() {
		case helium.AttributeNode, helium.NamespaceNode:
			return false
		}
		if !ixpath.OrderedFrom(n) {
			return false
		}
	}
	for i := 0; i < len(nodes); {
		nested := nodes[i+1 : i+1+nestedContextRun(nodes, i)]
		for _, n := range nested {
			if !ixpath.IsXDMChild(n) {
				return false
			}
		}
		i += 1 + len(nested)
	}
	return true
}

// nestedContextRun returns how many of the nodes that follow contexts[i] lie
// inside its subtree. contexts is sorted in document order, so they are the
// ones right after it.
func nestedContextRun(contexts []helium.Node, i int) int {
	top := contexts[i]
	prevParent := top.Parent()
	prevNested := false
	run := 0
	for _, n := range contexts[i+1:] {
		parent := n.Parent()
		nested := prevNested
		// A sibling of the previous node lies inside top exactly when the
		// previous node does: top is not one of them, so it is an ancestor
		// of one exactly when it is an ancestor of their parent.
		if parent == nil || parent != prevParent {
			nested = hasAncestor(n, top)
		}
		if !nested {
			break
		}
		run++
		prevParent = parent
		prevNested = true
	}
	return run
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
	switch {
	case len(step.Predicates) == 0:
		return descendantStepNoPredicates(ctx, ec, contexts, step)
	case quietPredicates(ec, step.Predicates):
		return descendantStepQuietPredicates(evalFn, ctx, ec, contexts, step)
	default:
		return descendantStepWithPredicates(evalFn, ctx, ec, contexts, step)
	}
}

// quietPredicates reports whether preds only select nodes: each is a
// position ([N]), an attribute test ([@a]) or an attribute comparison with a
// string ([@a = 's']) that never falls back to evaluating the comparison,
// which it does only for an attribute with a type annotation. Such predicates
// charge no operations and fail only when ctx is cancelled.
func quietPredicates(ec *evalContext, preds []Expr) bool {
	for _, pred := range preds {
		switch pred.(type) {
		case vmPositionPredicateExpr, vmAttributeExistsPredicateExpr:
		case vmAttributeEqualsStringPredicateExpr:
			if ec.typeAnnotations != nil {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// descendantStepNoPredicates walks the subtree of every outermost context
// node once, collecting the step's matches in document order.
func descendantStepNoPredicates(ctx context.Context, ec *evalContext, contexts []helium.Node, step vmLocationStep) ([]helium.Node, error) {
	var out []helium.Node
	total := 0
	// candidates counts what the second step enumerates. On the child axis
	// it is every subtree node but the context nodes.
	candidates := 0
	var sizes nestedSubtrees
	for i := 0; i < len(contexts); {
		c := contexts[i]
		sizes.reset(contexts[i+1 : i+1+nestedContextRun(contexts, i)])
		var visited, traversed int
		var err error
		out, visited, traversed, err = appendDescendantMatches(ctx, ec, out, c, step, &sizes)
		if err != nil {
			return nil, err
		}
		if err := chargeDescendantOrSelf(ctx, ec, visited, sizes.sizes); err != nil {
			return nil, err
		}
		total += visited
		candidates += traversed
		i += 1 + len(sizes.sizes)
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
// step to out and measuring the nested context subtrees in sizes. It returns
// the number of subtree nodes and the number of candidates the step
// enumerates from them.
func appendDescendantMatches(ctx context.Context, ec *evalContext, out []helium.Node, c helium.Node, step vmLocationStep, sizes *nestedSubtrees) ([]helium.Node, int, int, error) {
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
		if len(sizes.pending) > 0 {
			sizes.close(len(stack), visited)
		}
		last := len(stack) - 1
		cur := stack[last]
		stack = stack[:last]
		visited++
		if visited > ec.maxNodes {
			return nil, 0, 0, ixpath.ErrNodeSetLimit
		}
		if sizes.next < len(sizes.nested) {
			sizes.open(cur, len(stack), visited)
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
	if len(sizes.pending) > 0 {
		sizes.close(0, visited)
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
// charges the descendant-or-self step, then walks the outermost subtrees
// again, applying the second step and its predicates to every subtree node
// in document order.
func descendantStepWithPredicates(evalFn exprEvaluator, ctx context.Context, ec *evalContext, contexts []helium.Node, step vmLocationStep) ([]helium.Node, error) {
	total := 0
	var tops []helium.Node
	var sizes nestedSubtrees
	for i := 0; i < len(contexts); {
		c := contexts[i]
		sizes.reset(contexts[i+1 : i+1+nestedContextRun(contexts, i)])
		visited, err := countDescendantOrSelf(ctx, ec, c, &sizes)
		if err != nil {
			return nil, err
		}
		if err := chargeDescendantOrSelf(ctx, ec, visited, sizes.sizes); err != nil {
			return nil, err
		}
		total += visited
		tops = append(tops, c)
		i += 1 + len(sizes.sizes)
	}
	if err := finishDescendantOrSelfStep(ec, contexts[0], total); err != nil {
		return nil, err
	}

	w := descendantWalker{evalFn: evalFn, step: step}
	for _, c := range tops {
		if _, err := w.walk(ctx, ec, c); err != nil {
			return nil, err
		}
	}
	return clampDescendantResult(ec, w.out)
}

// descendantStepQuietPredicates walks the subtree of every outermost context
// node once, counting it as countDescendantOrSelf does and applying the
// second step and its predicates, which must pass quietPredicates, to every
// subtree node in document order. It charges what
// descendantStepWithPredicates charges, in the same order: the predicates
// charge nothing, so the second step's charges can wait until the walk ends.
func descendantStepQuietPredicates(evalFn exprEvaluator, ctx context.Context, ec *evalContext, contexts []helium.Node, step vmLocationStep) ([]helium.Node, error) {
	total := 0
	w := descendantWalker{evalFn: evalFn, step: step, counting: true}
	for i := 0; i < len(contexts); {
		c := contexts[i]
		w.sizes.reset(contexts[i+1 : i+1+nestedContextRun(contexts, i)])
		// The checks startDescendantWalk makes before the walk.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ec.maxNodes < 1 {
			return nil, ixpath.ErrNodeSetLimit
		}
		visited, err := w.walk(ctx, ec, c)
		if err != nil {
			return nil, err
		}
		if err := chargeDescendantOrSelf(ctx, ec, visited, w.sizes.sizes); err != nil {
			return nil, err
		}
		total += visited
		i += 1 + len(w.sizes.sizes)
	}
	if err := finishDescendantOrSelfStep(ec, contexts[0], total); err != nil {
		return nil, err
	}
	if err := ec.countOps(ctx, w.candidates); err != nil {
		return nil, err
	}
	return clampDescendantResult(ec, w.out)
}

// descendantWalker applies the second step and its predicates to every node
// of a subtree, in pre-order. It emits each child the step selects when the
// walk pops it, and the attributes the step selects when the walk reaches
// their element.
//
// Without counting, it charges the candidates of each node before applying
// the predicates, as the step-by-step evaluation does. With counting (quiet
// predicates), it also counts the subtree, fails where countDescendantOrSelf
// fails, measures the nested context subtrees in sizes, and adds the
// candidates to candidates for the caller to charge.
type descendantWalker struct {
	evalFn     exprEvaluator
	step       vmLocationStep
	counting   bool
	out        []helium.Node
	matched    []helium.Node
	stack      []descendantEntry
	sizes      nestedSubtrees
	candidates int
}

// walk walks the subtree of c. When counting, it returns the number of
// subtree nodes.
func (w *descendantWalker) walk(ctx context.Context, ec *evalContext, c helium.Node) (int, error) {
	step := w.step
	w.stack = append(w.stack[:0], descendantEntry{node: c})
	visited := 0
	for len(w.stack) > 0 {
		if w.counting {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			if len(w.sizes.pending) > 0 {
				w.sizes.close(len(w.stack), visited)
			}
		}
		last := len(w.stack) - 1
		cur := w.stack[last]
		w.stack = w.stack[:last]
		if w.counting {
			visited++
			if visited > ec.maxNodes {
				return 0, ixpath.ErrNodeSetLimit
			}
			if w.sizes.next < len(w.sizes.nested) {
				w.sizes.open(cur.node, len(w.stack), visited)
			}
		}
		if cur.selected {
			w.out = append(w.out, cur.node)
		}

		start := len(w.stack)
		var traversed int
		var err error
		if step.Axis == AxisChild {
			w.stack, w.matched, traversed, err = pushMatchingChildEntries(ctx, ec, w.stack, w.matched[:0], cur.node, step.NodeTest)
			if err != nil {
				return 0, err
			}
		} else {
			w.matched, traversed, err = appendAxisNodeMatches(ctx, w.matched[:0], ec, cur.node, AxisAttribute, step.NodeTest)
			if err != nil {
				return 0, err
			}
			w.stack, err = pushChildEntries(ctx, w.stack, cur.node)
			if err != nil {
				return 0, err
			}
		}
		if w.counting {
			w.candidates += traversed
		} else if err := ec.countOps(ctx, traversed); err != nil {
			return 0, err
		}
		selected := w.matched
		for _, pred := range step.Predicates {
			selected, err = applyVMPredicate(w.evalFn, ctx, ec, selected, pred)
			if err != nil {
				return 0, err
			}
		}
		if step.Axis == AxisChild {
			markSelectedEntries(w.stack[start:], selected)
		} else {
			w.out = append(w.out, selected...)
		}
		slices.Reverse(w.stack[start:])
	}
	if len(w.sizes.pending) > 0 {
		w.sizes.close(0, visited)
	}
	return visited, nil
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
// failing where ixpath.TraverseAxis fails on that axis, and measures the
// nested context subtrees in sizes.
func countDescendantOrSelf(ctx context.Context, ec *evalContext, c helium.Node, sizes *nestedSubtrees) (int, error) {
	stack, err := startDescendantWalk(ctx, ec, c)
	if err != nil {
		return 0, err
	}
	visited := 1
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if len(sizes.pending) > 0 {
			sizes.close(len(stack), visited)
		}
		last := len(stack) - 1
		cur := stack[last]
		stack = stack[:last]
		visited++
		if visited > ec.maxNodes {
			return 0, ixpath.ErrNodeSetLimit
		}
		if sizes.next < len(sizes.nested) {
			sizes.open(cur, len(stack), visited)
		}
		stack, err = ixpath.PushXDMChildren(ctx, stack, cur)
		if err != nil {
			return 0, err
		}
	}
	if len(sizes.pending) > 0 {
		sizes.close(0, visited)
	}
	return visited, nil
}

// chargeDescendantOrSelf charges the descendant-or-self traversal of an
// outermost context node, whose subtree has visited nodes, and then of the
// context nodes nested in it, whose subtrees have nested nodes each, as the
// step-by-step evaluation charges them one after the other.
func chargeDescendantOrSelf(ctx context.Context, ec *evalContext, visited int, nested []int) error {
	if err := ec.countOps(ctx, visited); err != nil {
		return err
	}
	for _, size := range nested {
		if err := ec.countOps(ctx, size); err != nil {
			return err
		}
	}
	return nil
}

// nestedSubtrees measures the subtrees of the context nodes that lie inside
// the subtree a walk visits. The walks call open only while a nested node is
// still ahead (next < len(nested)) and close only while a subtree is pending,
// so a walk without nested context nodes pays two comparisons per node. The walk pops nodes in pre-order from a stack
// that holds the pending XDM children; when it pops a nested context node,
// the node's descendants are exactly the nodes it pops while the stack is
// longer than it was right after that pop.
type nestedSubtrees struct {
	nested []helium.Node
	// sizes[i] is the size of the subtree of nested[i].
	sizes []int
	// next is the index of the next nested context node the walk reaches.
	next int
	// pending holds the nested context nodes whose subtrees the walk is in,
	// innermost last.
	pending []pendingSubtree
}

// pendingSubtree is a nested context node whose subtree the walk is in.
type pendingSubtree struct {
	index    int // index in nestedSubtrees.nested
	stackLen int // stack length right after the node was popped
	start    int // visit count including the node
}

// reset starts measuring the nested context nodes nested, which are sorted
// in document order, reusing the buffers of the previous measure.
func (s *nestedSubtrees) reset(nested []helium.Node) {
	s.nested = nested
	s.next = 0
	s.pending = s.pending[:0]
	if len(nested) == 0 {
		s.sizes = s.sizes[:0]
		return
	}
	s.sizes = slices.Grow(s.sizes[:0], len(nested))[:len(nested)]
	clear(s.sizes)
}

// open records that the walk popped n, leaving stackLen entries on the
// stack, as its visited-th node.
func (s *nestedSubtrees) open(n helium.Node, stackLen, visited int) {
	if s.next == len(s.nested) || n != s.nested[s.next] {
		return
	}
	s.pending = append(s.pending, pendingSubtree{index: s.next, stackLen: stackLen, start: visited})
	s.next++
}

// close ends the subtrees the walk has left: those whose node was popped
// with at least stackLen entries left on the stack. visited is the number
// of nodes the walk has popped so far.
func (s *nestedSubtrees) close(stackLen, visited int) {
	for len(s.pending) > 0 {
		last := len(s.pending) - 1
		p := s.pending[last]
		if stackLen > p.stackLen {
			return
		}
		s.sizes[p.index] = visited - p.start + 1
		s.pending = s.pending[:last]
	}
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
