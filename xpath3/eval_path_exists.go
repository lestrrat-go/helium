package xpath3

import (
	"context"
	"slices"
	"sync"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// This file evaluates a location path where only whether it selects a node
// matters: an operand whose effective boolean value is taken
// (vmInstruction.ebv), and the argument of fn:exists, fn:empty, fn:boolean
// and fn:not. The evaluation stops as soon as the path is known to select a
// node, instead of collecting every node first.
//
// The steps run depth first: the next step runs from each node a step
// selects as soon as it is selected, and the last step stops at the first
// node it selects. Depth first never repeats work the step-by-step
// evaluation does once, because only steps that cannot select one node twice
// from different context nodes run that way (pathExistsSplit): child,
// attribute and self steps, and one `//` walk (descendant-or-self::node()
// followed by a child or attribute step) from context nodes whose subtrees
// do not overlap. The steps before them run one at a time as in
// evalVMSteps, and depth first starts from their result.
//
// A child step whose predicates test each node on its own ([@a], [@a = 's'];
// nodeLocalPredicates) takes the children one at a time, so it stops at the
// first child it selects. Any other step builds the whole candidate list of a
// context node, charges it and filters it with the step's predicates, so
// position() and last() keep their meaning; only the last predicate of the
// last step stops at the first node it keeps. The evaluation charges one
// operation per node it enumerates and per node a predicate is evaluated
// for, as the step-by-step evaluation does, and applies the node-set limit to
// every step result it has seen so far and to every `//` walk. It therefore
// never does more work than the step-by-step evaluation and fails only where
// that evaluation fails too, though running steps depth first can make it
// meet another of that evaluation's errors first. When the path selects
// nothing, it does
// the same work and charges the same operations. When it stops early, the
// operations, node-set limits and predicate errors of the nodes it did not
// reach no longer apply.
//
// A step whose result holds more than one node registers its document in the
// order cache (ixpath.OrderStepResult), and the registration order orders
// nodes from different documents. The evaluation registers the document when
// a step it has seen holds more than one node, or a `//` walk covers more
// than one node, and only stops once that outcome is settled: the document
// is registered, or nothing the step-by-step evaluation registers is left. A
// later union across documents therefore orders them as before.

// pathExistsSplit returns the index of the step at which pathExists starts
// to run the steps of a location path depth first. The steps before it run
// one at a time. The steps are grouped the way evalVMSteps groups them: a
// descendant-or-self::node() step followed by a child or attribute step is
// one `//` walk, any other step is a step of its own.
//
// Depth first may run, before the last group, child, attribute and self
// steps, which select every node from at most one context node, and one
// `//` walk, provided its context nodes do not lie in each other's subtrees.
// That holds for the context list of the first group, which the walk handles
// as a whole, and, when the path starts depth first from its single start
// node, for every node that child, attribute and self steps select from it.
// Any other step before the last group, and every `//` walk but the last
// one, run one at a time.
func pathExistsSplit(steps []vmLocationStep) int {
	split := 0
	walks := 0
	firstWalk, lastWalk := 0, 0
	for i := 0; i < len(steps); {
		if fusesDescendantStep(steps, i) {
			if walks == 0 {
				firstWalk = i
			}
			lastWalk = i
			walks++
			i += 2
			continue
		}
		if i < len(steps)-1 && !selectsFromOneContext(steps[i].Axis) {
			split = i + 1
			walks = 0
		}
		i++
	}
	switch {
	case walks > 1:
		return lastWalk
	case walks == 1 && split > 0 && firstWalk != split:
		return firstWalk
	}
	return split
}

// selectsFromOneContext reports whether every node a step on axis selects
// from a list of context nodes is selected from only one of them.
func selectsFromOneContext(axis AxisType) bool {
	switch axis {
	case AxisChild, AxisAttribute, AxisSelf:
		return true
	}
	return false
}

// nodeLocalPredicates reports whether preds keep or drop each node on its
// own, whatever its position: each is an attribute test ([@a]) or an
// attribute comparison with a string ([@a = 's']) that never falls back to
// evaluating the comparison, which it does only for an attribute with a type
// annotation. Such predicates charge no operations and fail only when ctx is
// cancelled.
func nodeLocalPredicates(ec *evalContext, preds []Expr) bool {
	for _, pred := range preds {
		switch pred.(type) {
		case vmAttributeExistsPredicateExpr:
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

// keepsNode reports whether every predicate of preds, which must pass
// nodeLocalPredicates, keeps n.
func keepsNode(ctx context.Context, ec *evalContext, n helium.Node, preds []Expr) (bool, error) {
	for _, pred := range preds {
		var keep bool
		var err error
		switch p := pred.(type) {
		case vmAttributeExistsPredicateExpr:
			keep, err = nodeHasMatchingAttribute(ctx, ec, n, p.NodeTest)
		case vmAttributeEqualsStringPredicateExpr:
			// Without type annotations the comparison never falls back.
			keep, _, err = vmAttributeEqualsStringPredicateMatches(ctx, ec, n, p)
		}
		if err != nil || !keep {
			return false, err
		}
	}
	return true, nil
}

// pathExists reports whether the location path lp selects a node from the
// context node of ec, stopping as soon as the answer is known (see the top
// of this file). resultRegisters reports whether the caller registers the
// document of a result of more than one node in the order cache, as the
// merge of a path expression E1/path does; a location path on its own leaves
// that to its steps.
func pathExists(evalFn exprEvaluator, ctx context.Context, ec *evalContext, lp vmLocationPathExpr, resultRegisters bool) (bool, error) {
	start, err := locationPathStart(ec, lp)
	if err != nil {
		return false, err
	}
	if len(lp.Steps) == 0 {
		return true, nil
	}
	split := pathExistsSplit(lp.Steps)
	p := pathProbePool.Get().(*pathProbe) //nolint:forcetypeassert // the pool only holds *pathProbe
	defer p.release()
	p.start = start
	contexts := append(p.contexts[:0], start)
	p.contexts = contexts
	if split > 0 {
		contexts, err = evalVMSteps(evalFn, ctx, ec, contexts, lp.Steps[:split])
		if err != nil {
			return false, err
		}
		if len(contexts) == 0 {
			return false, nil
		}
	}
	p.ec = ec
	p.steps = lp.Steps
	p.resultRegisters = resultRegisters
	// A step result of more than one node has registered its document.
	p.reserved = len(contexts) > 1
	if len(lp.Steps) <= len(p.countsBuf) {
		p.counts = p.countsBuf[:len(lp.Steps)]
		clear(p.counts)
	} else {
		p.counts = make([]int, len(lp.Steps))
	}
	if _, err := p.run(evalFn, ctx, split, contexts); err != nil {
		return false, err
	}
	return p.found, nil
}

// pathProbePool keeps the buffers of finished probes, so that a path
// evaluated once per node, such as a predicate, does not allocate them anew.
var pathProbePool = sync.Pool{New: newPathProbe}

func newPathProbe() any {
	return &pathProbe{}
}

// maxPooledProbeBuffer is the largest buffer capacity a pooled probe keeps.
const maxPooledProbeBuffer = 1024

// release returns p to pathProbePool, dropping its references to nodes and
// its buffers that grew past maxPooledProbeBuffer.
func (p *pathProbe) release() {
	p.ec = nil
	p.steps = nil
	p.start = nil
	p.counts = nil
	p.contexts = trimProbeBuffer(p.contexts)
	p.buf = trimProbeBuffer(p.buf)
	p.walk = trimProbeBuffer(p.walk)
	p.sizes.reset(nil)
	p.sizes.nested = nil
	p.pending = 0
	p.found = false
	p.reserved = false
	p.lastWalkSettled = false
	p.resultRegisters = false
	p.slack = 0
	p.registeredChecked = false
	pathProbePool.Put(p)
}

// trimProbeBuffer returns buf emptied of node references, or nil when its
// capacity passed maxPooledProbeBuffer.
func trimProbeBuffer(buf []helium.Node) []helium.Node {
	if cap(buf) > maxPooledProbeBuffer {
		return nil
	}
	clear(buf[:cap(buf)])
	return buf[:0]
}

// pathProbe holds the state of one pathExists evaluation.
//
// The evaluator is a parameter of the methods rather than a field: a field
// would let the method value the VM passes as evalFn escape to the heap on
// every evaluation.
type pathProbe struct {
	ec    *evalContext
	steps []vmLocationStep
	// start is the node the path starts from, in the tree of every node it
	// visits.
	start helium.Node
	// contexts holds the start node as a context list.
	contexts []helium.Node
	// counts[i] is the number of nodes step i has selected so far. A `//`
	// walk counts under the index of its child or attribute step.
	counts    []int
	countsBuf [8]int
	// buf holds the candidate lists of the steps in progress, one after the
	// other, deepest last.
	buf []helium.Node
	// walk is the pre-order stack of the `//` walk, of which a probe runs at
	// most one at a time.
	walk  []helium.Node
	sizes nestedSubtrees
	// pending counts the operations the `//` walk has done and not charged
	// yet.
	pending int
	// found records that the last step selected a node.
	found bool
	// reserved records that the document of the path is registered in the
	// order cache.
	reserved bool
	// lastWalkSettled records that the last step is a `//` walk that has
	// registered whatever the step-by-step evaluation registers for it.
	lastWalkSettled bool
	resultRegisters bool
	// slack counts the nodes visited since the last step found a node while
	// the order cache was not settled, and registeredChecked records that
	// stopFound asked the cache.
	slack             int
	registeredChecked bool
}

// settled reports whether stopping now leaves the order cache as the
// step-by-step evaluation would leave it: the path's document is registered,
// or the last step is a `//` walk that settled it.
func (p *pathProbe) settled() bool {
	return p.reserved || p.lastWalkSettled
}

// foundSlack is how many more nodes a probe that has found a node, but has
// not settled the order cache, visits before it asks the cache whether the
// path's document is registered already.
const foundSlack = 16

// stopFound reports whether the probe can stop before it visits the next
// node: the last step has selected a node and the order cache is settled.
// Until then a probe that has found a node keeps looking for a second one,
// whose step result would register the document. After foundSlack more
// nodes it asks the cache once whether the document is registered already,
// which settles it too; a short search ends before it needs to ask.
func (p *pathProbe) stopFound() bool {
	if !p.found {
		return false
	}
	if p.settled() {
		return true
	}
	if p.registeredChecked {
		return false
	}
	p.slack++
	if p.slack < foundSlack {
		return false
	}
	p.registeredChecked = true
	if ixpath.DocumentRegistered(p.ec.docOrder, p.start) {
		p.reserved = true
	}
	return p.reserved
}

// reserve registers the document of the path in the order cache.
func (p *pathProbe) reserve() {
	if p.reserved {
		return
	}
	p.ec.docOrder.ReserveDocument(p.start)
	p.reserved = true
}

// run evaluates the steps from index i on from the context list contexts,
// which is sorted in document order and duplicate-free. It returns true when
// the probe stops.
func (p *pathProbe) run(evalFn exprEvaluator, ctx context.Context, i int, contexts []helium.Node) (bool, error) {
	if fusesDescendantStep(p.steps, i) {
		if fusesDescendantContexts(contexts) {
			return p.runDescendant(evalFn, ctx, i, contexts)
		}
		// The descendant-or-self step on its own, as evalVMSteps evaluates
		// it from such a context list, and then its next step.
		nodes, err := evalVMStepNoPredicates(ctx, p.ec, contexts, p.steps[i])
		if err != nil {
			return false, err
		}
		if len(nodes) > 1 {
			p.reserved = true
		}
		i++
		contexts = nodes
	}
	step := p.steps[i]
	if step.Axis == AxisChild && nodeLocalPredicates(p.ec, step.Predicates) {
		for _, n := range contexts {
			if p.stopFound() {
				return true, nil
			}
			stop, err := p.runChildren(evalFn, ctx, i, n)
			if err != nil || stop {
				return stop, err
			}
		}
		return false, nil
	}
	for _, n := range contexts {
		if p.stopFound() {
			return true, nil
		}
		mark := len(p.buf)
		var traversed int
		var err error
		p.buf, traversed, err = appendAxisNodeMatches(ctx, p.buf, p.ec, n, step.Axis, step.NodeTest)
		if err != nil {
			return false, err
		}
		if err := p.ec.countOps(ctx, traversed); err != nil {
			return false, err
		}
		stop, err := p.selectFrom(evalFn, ctx, i, p.buf[mark:], false)
		p.buf = p.buf[:mark]
		if err != nil || stop {
			return stop, err
		}
	}
	return false, nil
}

// runChildren runs child step i, whose predicates pass nodeLocalPredicates,
// from n, taking the children of n one at a time. It charges the children it
// enumerates before the next step runs from a selected child, and when the
// probe stops.
func (p *pathProbe) runChildren(evalFn exprEvaluator, ctx context.Context, i int, n helium.Node) (bool, error) {
	if _, ok := n.(*helium.Attribute); ok {
		return false, nil
	}
	step := p.steps[i]
	traversed := 0
	for child := range helium.Children(n) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if !ixpath.IsXDMChild(child) {
			continue
		}
		if p.stopFound() {
			return true, p.ec.countOps(ctx, traversed)
		}
		traversed++
		if !matchNodeTest(step.NodeTest, child, AxisChild, p.ec) {
			continue
		}
		keep, err := keepsNode(ctx, p.ec, child, step.Predicates)
		if err != nil {
			return false, err
		}
		if !keep {
			continue
		}
		if err := p.ec.countOps(ctx, traversed); err != nil {
			return false, err
		}
		traversed = 0
		stop, err := p.selectNode(evalFn, ctx, i, child, false)
		if err != nil || stop {
			return stop, err
		}
	}
	return false, p.ec.countOps(ctx, traversed)
}

// selectFrom applies the predicates of step i to candidates, the nodes the
// step finds from one context node, or from one node of a `//` walk when
// walked is set, and goes on with the nodes they select (selectNodes).
func (p *pathProbe) selectFrom(evalFn exprEvaluator, ctx context.Context, i int, candidates []helium.Node, walked bool) (bool, error) {
	if len(candidates) == 0 {
		return false, nil
	}
	step := p.steps[i]
	last := i == len(p.steps)-1
	preds := step.Predicates
	if last && len(preds) > 0 {
		// The last predicate of the last step only needs to find nodes.
		preds = preds[:len(preds)-1]
	}
	selected := candidates
	for _, pred := range preds {
		var err error
		selected, err = applyVMPredicate(evalFn, ctx, p.ec, selected, pred)
		if err != nil {
			return false, err
		}
		if len(selected) == 0 {
			return false, nil
		}
	}
	n := len(selected)
	if last && len(step.Predicates) > 0 {
		var err error
		n, err = p.countLastPredicate(evalFn, ctx, i, selected, step.Predicates[len(step.Predicates)-1])
		if err != nil || n == 0 {
			return false, err
		}
	}
	return p.selectNodes(evalFn, ctx, i, selected, n, walked)
}

// selectNode goes on with the single node n that step i selects.
func (p *pathProbe) selectNode(evalFn exprEvaluator, ctx context.Context, i int, n helium.Node, walked bool) (bool, error) {
	mark := len(p.buf)
	p.buf = append(p.buf, n)
	stop, err := p.selectNodes(evalFn, ctx, i, p.buf[mark:], 1, walked)
	p.buf = p.buf[:mark]
	return stop, err
}

// selectNodes counts n nodes that step i selects and runs the next step from
// each node of selected, or, for the last step, records that the path
// selects a node. selected holds the n nodes, except that the last step may
// pass fewer.
func (p *pathProbe) selectNodes(evalFn exprEvaluator, ctx context.Context, i int, selected []helium.Node, n int, walked bool) (bool, error) {
	last := i == len(p.steps)-1
	p.counts[i] += n
	if p.counts[i] > p.ec.maxNodes {
		return false, ixpath.ErrNodeSetLimit
	}
	// A step result of more than one node registers its document. The
	// result of a `//` walk does not (clampDescendantResult): the walk
	// registers it for the nodes it covers.
	if p.counts[i] > 1 && (!walked || (last && p.resultRegisters)) {
		p.reserve()
	}
	if last {
		p.found = true
		return p.settled(), nil
	}
	for k := range selected {
		stop, err := p.run(evalFn, ctx, i+1, selected[k:k+1])
		if err != nil || stop {
			return stop, err
		}
	}
	return false, nil
}

// countLastPredicate applies pred, the last predicate of the last step, to
// nodes and returns how many nodes it keeps, stopping as soon as that is
// enough to stop the probe: one node when the order cache is settled,
// otherwise as many as make the step result hold two nodes.
func (p *pathProbe) countLastPredicate(evalFn exprEvaluator, ctx context.Context, i int, nodes []helium.Node, pred Expr) (int, error) {
	switch pred.(type) {
	case vmPositionPredicateExpr, vmAttributeExistsPredicateExpr, vmAttributeEqualsStringPredicateExpr:
		selected, err := applyVMPredicate(evalFn, ctx, p.ec, nodes, pred)
		return len(selected), err
	}
	need := 1
	if !p.settled() {
		need = max(1, 2-p.counts[i])
	}
	return countPredicateUpTo(evalFn, ctx, p.ec, nodes, pred, need)
}

// countPredicateUpTo evaluates pred for the nodes of nodes as applyPredicate
// does, and returns how many it keeps, stopping once it keeps need of them.
// applyPredicate charges one operation per node before it evaluates any;
// countPredicateUpTo charges each node before it evaluates it.
func countPredicateUpTo(evalFn exprEvaluator, ctx context.Context, ec *evalContext, nodes []helium.Node, pred Expr, need int) (int, error) {
	size := len(nodes)
	kept := 0
	for i, n := range nodes {
		if err := ec.countOps(ctx, 1); err != nil {
			return 0, err
		}
		frame := ec.pushNodeContext(n, i+1, size)
		r, err := evalFn(ctx, ec, pred)
		ec.restoreContext(frame)
		if err != nil {
			return 0, err
		}
		match, err := predicateTrue(r, i+1)
		if err != nil {
			return 0, err
		}
		if !match {
			continue
		}
		kept++
		if kept == need {
			break
		}
	}
	return kept, nil
}

// runDescendant runs the `//` walk made of steps i and i+1 from contexts,
// which must pass fusesDescendantContexts, as evalVMDescendantStep evaluates
// it: one pre-order walk of each outermost context node's subtree, applying
// step i+1 and its predicates to every subtree node.
func (p *pathProbe) runDescendant(evalFn exprEvaluator, ctx context.Context, i int, contexts []helium.Node) (bool, error) {
	ec := p.ec
	si := i + 1
	last := si == len(p.steps)-1
	// The descendant-or-self step registers its document when it covers more
	// than one node (finishDescendantOrSelfStep): with several context
	// nodes, or one with a child. Registering first keeps it ahead of any
	// document a predicate registers, as in the step-by-step evaluation.
	if len(contexts) > 1 || hasXDMChild(contexts[0]) {
		p.reserve()
	}
	if last && !p.resultRegisters {
		p.lastWalkSettled = true
	}
	total := 0
	for k := 0; k < len(contexts); {
		c := contexts[k]
		p.sizes.reset(contexts[k+1 : k+1+nestedContextRun(contexts, k)])
		// The checks startDescendantWalk makes before the walk.
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if ec.maxNodes < 1 {
			return false, ixpath.ErrNodeSetLimit
		}
		visited, stop, err := p.walkSubtree(evalFn, ctx, si, c)
		if err != nil || stop {
			return stop, err
		}
		total += visited
		if total > ec.maxNodes {
			return false, ixpath.ErrNodeSetLimit
		}
		// The descendant-or-self step charges the subtree of each context
		// node nested in c too (chargeDescendantOrSelf).
		for _, size := range p.sizes.sizes {
			p.pending += size
		}
		if err := p.charge(ctx); err != nil {
			return false, err
		}
		k += 1 + len(p.sizes.sizes)
	}
	return false, nil
}

// walkSubtree walks the subtree of c in pre-order, applying step si to every
// node and measuring the subtrees of the nested context nodes in p.sizes. It
// returns the number of subtree nodes, and true when the probe stops.
//
// A child step whose predicates pass nodeLocalPredicates tests each node
// when the walk pops it, so the walk stops at the first node it selects.
// Otherwise the walk applies the step to each node it pops: it builds the
// candidate list (its matching children, or its matching attributes) and
// filters it with the predicates.
//
// The walk charges one operation per subtree node and one per candidate, as
// the step-by-step evaluation does. It charges them before a predicate that
// can observe them runs, before the next step runs and when the probe stops;
// quiet predicates of the last step let them wait until the subtree ends.
func (p *pathProbe) walkSubtree(evalFn exprEvaluator, ctx context.Context, si int, c helium.Node) (int, bool, error) {
	ec := p.ec
	step := p.steps[si]
	last := si == len(p.steps)-1
	streaming := step.Axis == AxisChild && nodeLocalPredicates(ec, step.Predicates)
	chargeEach := !last || !quietPredicates(ec, step.Predicates)
	p.walk = append(p.walk[:0], c)
	visited := 0
	for len(p.walk) > 0 {
		if err := ctx.Err(); err != nil {
			return 0, false, err
		}
		if len(p.sizes.pending) > 0 {
			p.sizes.close(len(p.walk), visited)
		}
		top := len(p.walk) - 1
		cur := p.walk[top]
		p.walk = p.walk[:top]
		visited++
		if visited > ec.maxNodes {
			return 0, false, ixpath.ErrNodeSetLimit
		}
		if p.sizes.next < len(p.sizes.nested) {
			p.sizes.open(cur, len(p.walk), visited)
		}
		// One operation for the descendant-or-self node.
		p.pending++
		var stop bool
		var err error
		if streaming {
			stop, err = p.walkChild(evalFn, ctx, si, cur, visited > 1, !last)
		} else {
			stop, err = p.walkParent(evalFn, ctx, si, cur, chargeEach)
		}
		if err != nil {
			return 0, false, err
		}
		if stop {
			if err := p.charge(ctx); err != nil {
				return 0, false, err
			}
			return visited, true, nil
		}
	}
	if len(p.sizes.pending) > 0 {
		p.sizes.close(0, visited)
	}
	return visited, false, nil
}

// walkChild applies child step si, whose predicates pass
// nodeLocalPredicates, to cur, a node the walk pops, and pushes the children
// of cur. cur is a candidate unless it is the context node the walk started
// from (candidate is false). chargeFirst charges the operations so far
// before the next step runs.
func (p *pathProbe) walkChild(evalFn exprEvaluator, ctx context.Context, si int, cur helium.Node, candidate, chargeFirst bool) (bool, error) {
	step := p.steps[si]
	if candidate {
		p.pending++
		if matchNodeTest(step.NodeTest, cur, AxisChild, p.ec) {
			keep, err := keepsNode(ctx, p.ec, cur, step.Predicates)
			if err != nil {
				return false, err
			}
			if keep {
				if chargeFirst {
					if err := p.charge(ctx); err != nil {
						return false, err
					}
				}
				stop, err := p.selectNode(evalFn, ctx, si, cur, true)
				if err != nil || stop {
					return stop, err
				}
			}
		}
	}
	// In XPath, attributes have no children.
	if _, ok := cur.(*helium.Attribute); ok {
		return false, nil
	}
	var err error
	p.walk, err = ixpath.PushXDMChildren(ctx, p.walk, cur)
	return false, err
}

// walkParent applies step si to cur, a node the walk pops: it finds the
// candidates of cur (its matching children, or its matching attributes),
// pushes the children of cur and filters the candidates with the step's
// predicates. chargeEach charges the operations so far before the
// predicates and the next step run.
func (p *pathProbe) walkParent(evalFn exprEvaluator, ctx context.Context, si int, cur helium.Node, chargeEach bool) (bool, error) {
	mark := len(p.buf)
	var traversed int
	var err error
	p.walk, p.buf, traversed, err = pushWalkCandidates(ctx, p.ec, p.walk, p.buf, cur, p.steps[si])
	if err != nil {
		return false, err
	}
	p.pending += traversed
	if len(p.buf) == mark {
		return false, nil
	}
	if chargeEach {
		if err := p.charge(ctx); err != nil {
			return false, err
		}
	}
	// The next steps leave p.walk alone: a probe runs one `//` walk.
	stop, err := p.selectFrom(evalFn, ctx, si, p.buf[mark:], true)
	p.buf = p.buf[:mark]
	return stop, err
}

// charge charges the operations the `//` walk has not charged yet.
func (p *pathProbe) charge(ctx context.Context) error {
	n := p.pending
	p.pending = 0
	return p.ec.countOps(ctx, n)
}

// pushWalkCandidates pushes the XDM children of n onto the pre-order stack
// and appends the candidates of step from n to matched: the children that
// match its node test on the child axis, the matching attributes on the
// attribute axis. It returns the number of candidates enumerated, which the
// step charges.
func pushWalkCandidates(ctx context.Context, ec *evalContext, stack, matched []helium.Node, n helium.Node, step vmLocationStep) ([]helium.Node, []helium.Node, int, error) {
	traversed := 0
	if step.Axis == AxisAttribute {
		var err error
		matched, traversed, err = appendAxisNodeMatches(ctx, matched, ec, n, AxisAttribute, step.NodeTest)
		if err != nil {
			return nil, nil, 0, err
		}
	}
	// In XPath, attributes have no children.
	if _, ok := n.(*helium.Attribute); ok {
		return stack, matched, traversed, nil
	}
	start := len(stack)
	for child := range helium.Children(n) {
		if err := ctx.Err(); err != nil {
			return nil, nil, 0, err
		}
		if !ixpath.IsXDMChild(child) {
			continue
		}
		stack = append(stack, child)
		if step.Axis != AxisChild {
			continue
		}
		traversed++
		if matchNodeTest(step.NodeTest, child, AxisChild, ec) {
			matched = append(matched, child)
		}
	}
	slices.Reverse(stack[start:])
	return stack, matched, traversed, nil
}

// hasXDMChild reports whether n has a child in the XPath data model.
func hasXDMChild(n helium.Node) bool {
	if _, ok := n.(*helium.Attribute); ok {
		return false
	}
	for child := range helium.Children(n) {
		if ixpath.IsXDMChild(child) {
			return true
		}
	}
	return false
}
