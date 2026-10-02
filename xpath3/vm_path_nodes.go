package xpath3

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// This file lets VM instructions pass node lists to each other. A location
// path, a union, an intersect/except, a path expression ending in an axis
// step, and a filter expression over one of these all compute a
// []helium.Node and return it as a Sequence that holds nodeItemFor(n) for
// every node n. Building that sequence costs one allocation per node.
//
// An instruction whose operand is one of these producers reads the node list
// instead when it only needs the nodes: fn:count and its relatives, the
// operands of union and intersect/except, the base of a filter, of a path
// step and of a simple map, a predicate or condition that only takes the
// effective boolean value, and the result of the whole expression
// (Evaluator.Evaluate keeps the node list in the Result). The producer is
// evaluated exactly as v.evalExpr evaluates it, with the same recursion
// accounting, so results, errors and limits do not change.

// nodeListInstruction returns the index of the instruction expr refers to
// when that instruction is a node-list producer: its result is always the
// nodeItemFor wrapping of a node list, or an error. The producers are a
// location path, a union, an intersect/except, a path expression with an
// axis step after its first operand, and a filter expression over one of
// these.
func nodeListInstruction(instructions []vmInstruction, expr Expr) (int, bool) {
	ref, ok := expr.(compiledExprRef)
	if !ok || ref.index < 0 || ref.index >= len(instructions) {
		return 0, false
	}
	inst := instructions[ref.index]
	switch inst.op {
	case vmOpLocationPath:
		_, ok = AsExpr[vmLocationPathExpr](inst.payload)
	case vmOpUnion:
		_, ok = AsExpr[UnionExpr](inst.payload)
	case vmOpIntersectExcept:
		_, ok = AsExpr[IntersectExceptExpr](inst.payload)
	case vmOpPath:
		var e vmPathExpr
		e, ok = AsExpr[vmPathExpr](inst.payload)
		ok = ok && e.Path != nil
	case vmOpFilter:
		var e FilterExpr
		e, ok = AsExpr[FilterExpr](inst.payload)
		if ok {
			_, ok = nodeListInstruction(instructions, e.Expr)
		}
	default:
		ok = false
	}
	return ref.index, ok
}

// markEBV records that the parent of expr only takes the effective boolean
// value of its result, when expr refers to a node-list producer. The
// instruction then evaluates to xs:boolean, true when the node list is not
// empty: the effective boolean value of a node sequence is true exactly when
// it is not empty, and a single xs:boolean is never taken as a position by a
// predicate.
func (b *vmBuilder) markEBV(expr Expr) {
	if index, ok := nodeListInstruction(b.instructions, expr); ok {
		b.instructions[index].ebv = true
	}
}

// evalNodeListRef evaluates expr the way v.evalExpr does when expr refers to
// a node-list producer (nodeListInstruction), and returns the node list. ok
// reports whether expr is such a reference; when it is false nothing was
// evaluated and the caller evaluates expr itself.
func (v *vm) evalNodeListRef(ctx context.Context, ec *evalContext, expr Expr) ([]helium.Node, bool, error) {
	index, ok := nodeListInstruction(v.program.instructions, expr)
	if !ok {
		return nil, false, nil
	}
	inst := v.program.instructions[index]
	// The recursion accounting of evalWith: no evaluation panics past a
	// recover, so decrementing after the call matches its deferred decrement.
	ec.depth++
	if ec.maxRecursionDepth > 0 && ec.depth > ec.maxRecursionDepth {
		return nil, true, ErrRecursionLimit
	}
	nodes, err := v.instructionNodes(ctx, ec, inst)
	ec.depth--
	return nodes, true, err
}

// instructionNodes evaluates the node-list producer inst to its node list.
func (v *vm) instructionNodes(ctx context.Context, ec *evalContext, inst vmInstruction) ([]helium.Node, error) {
	switch inst.op {
	case vmOpLocationPath:
		if lp, ok := AsExpr[vmLocationPathExpr](inst.payload); ok {
			return evalVMLocationPathNodes(v.evalExpr, ctx, ec, lp)
		}
	case vmOpUnion:
		if e, ok := AsExpr[UnionExpr](inst.payload); ok {
			return v.unionNodes(ctx, ec, e)
		}
	case vmOpIntersectExcept:
		if e, ok := AsExpr[IntersectExceptExpr](inst.payload); ok {
			return v.intersectExceptNodes(ctx, ec, e)
		}
	case vmOpPath:
		if e, ok := AsExpr[vmPathExpr](inst.payload); ok && e.Path != nil {
			return v.pathNodes(ctx, ec, e)
		}
	case vmOpFilter:
		if e, ok := AsExpr[FilterExpr](inst.payload); ok {
			return v.filterNodes(ctx, ec, e)
		}
	}
	return nil, fmt.Errorf("%w: bad payload for %s", ErrUnsupportedExpr, inst.op)
}

// evalEBV evaluates the node-list producer inst, whose parent only takes its
// effective boolean value (vmInstruction.ebv), to xs:boolean.
func (v *vm) evalEBV(ctx context.Context, ec *evalContext, inst vmInstruction) (Sequence, error) {
	nodes, err := v.instructionNodes(ctx, ec, inst)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(len(nodes) > 0), nil
}

// vmOperand is an evaluated operand: the node list of a node-list producer,
// or the sequence of any other expression.
type vmOperand struct {
	nodes   []helium.Node
	seq     Sequence
	isNodes bool
}

// evalOperand evaluates expr as v.evalExpr does, keeping the node list when
// expr is a node-list producer.
func (v *vm) evalOperand(ctx context.Context, ec *evalContext, expr Expr) (vmOperand, error) {
	nodes, ok, err := v.evalNodeListRef(ctx, ec, expr)
	if ok {
		return vmOperand{nodes: nodes, isNodes: true}, err
	}
	seq, err := v.evalExpr(ctx, ec, expr)
	return vmOperand{seq: seq}, err
}

// length returns the number of items of o.
func (o vmOperand) length() int {
	if o.isNodes {
		return len(o.nodes)
	}
	return seqLen(o.seq)
}

// appendNodes appends the nodes of o to dst, as appendSequenceNodes does.
func (o vmOperand) appendNodes(dst []helium.Node) ([]helium.Node, bool) {
	if o.isNodes {
		return append(dst, o.nodes...), true
	}
	return appendSequenceNodes(dst, o.seq)
}

// nodeList returns the nodes of o, as NodesFrom does.
func (o vmOperand) nodeList() ([]helium.Node, bool) {
	if o.isNodes {
		return o.nodes, true
	}
	return NodesFrom(o.seq)
}

// evalFunctionCall evaluates a static function call. When the call has one
// argument and it is a node-list producer, the argument is evaluated to its
// node list first, and the built-in fn:count, fn:exists, fn:empty,
// fn:boolean, fn:not and fn:head compute their result from that list. Every
// other function gets the node items evalFunctionCall would pass it.
func (v *vm) evalFunctionCall(ctx context.Context, ec *evalContext, e FunctionCall) (Sequence, error) {
	if len(e.Args) != 1 {
		return evalFunctionCall(v.evalExpr, ctx, ec, e)
	}
	nodes, ok, err := v.evalNodeListRef(ctx, ec, e.Args[0])
	if !ok {
		return evalFunctionCall(v.evalExpr, ctx, ec, e)
	}
	if err != nil {
		return nil, err
	}
	r, err := resolveFunctionInfo(ctx, ec, e.Prefix, e.Name, 1)
	if err != nil {
		return nil, err
	}
	if seq, ok := callNodeListFunction(ctx, ec, r, nodes); ok {
		return seq, nil
	}
	args := []Sequence{enrichNodeItems(ctx, ec, nodeItemsFor(ctx, ec, nodes))}
	return callResolvedFunction(ctx, ec, r, args)
}

// Local names of the fn: functions callNodeListFunction computes.
const (
	fnNameCount   = "count"
	fnNameExists  = "exists"
	fnNameBoolean = "boolean"
	fnNameEmpty   = "empty"
	fnNameNot     = "not"
	fnNameHead    = "head"
)

// callNodeListFunction returns the result of the built-in function r applied
// to the sequence of nodes, and true, when r is one whose result depends only
// on the length of its argument or its first item. Each of them takes item()*,
// which every node sequence matches without coercion, in XPath 1.0
// compatibility mode too.
func callNodeListFunction(ctx context.Context, ec *evalContext, r resolvedFunction, nodes []helium.Node) (Sequence, bool) {
	if !r.isBuiltin || r.uri != NSFn {
		return nil, false
	}
	switch r.name {
	case fnNameCount:
		return SingleInteger(int64(len(nodes))), true
	case fnNameExists, fnNameBoolean:
		// A sequence that starts with a node has the effective boolean value
		// true; the empty sequence has false.
		return SingleBoolean(len(nodes) > 0), true
	case fnNameEmpty, fnNameNot:
		return SingleBoolean(len(nodes) == 0), true
	case fnNameHead:
		if len(nodes) == 0 {
			return validNilSequence, true
		}
		return ItemSlice{nodeItemFor(ctx, ec, nodes[0])}, true
	}
	return nil, false
}

// unionNodes evaluates a union to its node list, as evalUnionExpr does.
func (v *vm) unionNodes(ctx context.Context, ec *evalContext, e UnionExpr) ([]helium.Node, error) {
	left, err := v.evalOperand(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := v.evalOperand(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	// Gather both operands into one buffer, left then right, so the union
	// can return it as is when it is already in document order.
	nodes, ok := left.appendNodes(make([]helium.Node, 0, left.length()+right.length()))
	if !ok {
		return nil, ErrUnionNotNodeSet
	}
	split := len(nodes)
	nodes, ok = right.appendNodes(nodes)
	if !ok {
		return nil, ErrUnionNotNodeSet
	}
	return ixpath.UnionNodeSets(nodes, split, ec.docOrder, ec.maxNodes)
}

// intersectExceptNodes evaluates intersect or except to its node list, as
// evalIntersectExceptExpr does.
func (v *vm) intersectExceptNodes(ctx context.Context, ec *evalContext, e IntersectExceptExpr) ([]helium.Node, error) {
	left, err := v.evalOperand(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := v.evalOperand(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	leftNodes, ok1 := left.nodeList()
	rightNodes, ok2 := right.nodeList()
	if !ok1 || !ok2 {
		return nil, ErrUnionNotNodeSet
	}
	return intersectExceptNodes(ec, e.Op, leftNodes, rightNodes)
}

// pathNodes evaluates a path expression E1/path to its node list, as
// evalVMPathExpr does. e.Path must not be nil.
func (v *vm) pathNodes(ctx context.Context, ec *evalContext, e vmPathExpr) ([]helium.Node, error) {
	base, err := v.evalOperand(ctx, ec, e.Filter)
	if err != nil {
		return nil, err
	}
	baseNodes, ok := base.nodeList()
	if !ok {
		return nil, ErrPathNotNodeSet
	}
	return vmPathStepNodes(v.evalExpr, ctx, ec, e, baseNodes)
}

// evalPathExpr evaluates a path expression, reading the first operand as a
// node list when it is a node-list producer.
func (v *vm) evalPathExpr(ctx context.Context, ec *evalContext, e vmPathExpr) (Sequence, error) {
	if e.Path == nil {
		return evalVMPathExpr(v.evalExpr, ctx, ec, e)
	}
	nodes, err := v.pathNodes(ctx, ec, e)
	if err != nil {
		return nil, err
	}
	return nodeItemsFor(ctx, ec, nodes), nil
}

// evalUnionExpr evaluates a union, reading node-list operands as node
// lists.
func (v *vm) evalUnionExpr(ctx context.Context, ec *evalContext, e UnionExpr) (Sequence, error) {
	nodes, err := v.unionNodes(ctx, ec, e)
	if err != nil {
		return nil, err
	}
	return nodeItemsFor(ctx, ec, nodes), nil
}

// evalIntersectExceptExpr evaluates intersect or except, reading node-list
// operands as node lists.
func (v *vm) evalIntersectExceptExpr(ctx context.Context, ec *evalContext, e IntersectExceptExpr) (Sequence, error) {
	nodes, err := v.intersectExceptNodes(ctx, ec, e)
	if err != nil {
		return nil, err
	}
	return nodeItemsFor(ctx, ec, nodes), nil
}

// evalFilterExpr evaluates a filter expression. When the filtered expression
// is a node-list producer, its node list goes straight to the predicates,
// which evalFilterExpr would reach through a sequence of node items.
func (v *vm) evalFilterExpr(ctx context.Context, ec *evalContext, e FilterExpr) (Sequence, error) {
	if _, ok := nodeListInstruction(v.program.instructions, e.Expr); !ok {
		return evalFilterExpr(v.evalExpr, ctx, ec, e)
	}
	nodes, err := v.filterNodes(ctx, ec, e)
	if err != nil {
		return nil, err
	}
	return nodeItemsFor(ctx, ec, nodes), nil
}

// filterNodes evaluates a filter expression over a node-list producer to its
// node list.
func (v *vm) filterNodes(ctx context.Context, ec *evalContext, e FilterExpr) ([]helium.Node, error) {
	nodes, _, err := v.evalNodeListRef(ctx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	for _, pred := range e.Predicates {
		nodes, err = v.applyFilterPredicate(ctx, ec, nodes, pred)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// applyFilterPredicate applies one predicate of a filter expression to nodes,
// as applyPredicate does. A numeric literal predicate selects by position
// without evaluating the literal once per node: applyPredicate charges one
// operation per node, then evaluates the literal for each node through
// evalWith, which can only fail on the recursion limit, the same way for
// every node.
func (v *vm) applyFilterPredicate(ctx context.Context, ec *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
	lit, ok := pred.(LiteralExpr)
	if !ok {
		return applyPredicate(v.evalExpr, ctx, ec, nodes, pred)
	}
	seq, err := evalLiteral(lit)
	if err != nil {
		return applyPredicate(v.evalExpr, ctx, ec, nodes, pred)
	}
	av, ok := seq.Get(0).(AtomicValue)
	if !ok || !av.IsNumeric() {
		return applyPredicate(v.evalExpr, ctx, ec, nodes, pred)
	}
	if err := ec.countOps(ctx, len(nodes)); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	// evalWith on the first node: on failure it leaves the depth raised.
	ec.depth++
	if ec.maxRecursionDepth > 0 && ec.depth > ec.maxRecursionDepth {
		return nil, ErrRecursionLimit
	}
	ec.depth--
	// predicateTrue keeps the node at position i+1 when the number equals it.
	f := av.ToFloat64()
	var result []helium.Node
	for i, n := range nodes {
		if f == float64(i+1) {
			result = append(result, n)
		}
	}
	return result, nil
}

// evalSimpleMapExpr evaluates E1 ! E2, reading E1 as a node list when it is
// a node-list producer. evalSimpleMapExpr gives each node item of E1 its node
// as the context node, so the node list sets the same focus.
func (v *vm) evalSimpleMapExpr(ctx context.Context, ec *evalContext, e SimpleMapExpr) (Sequence, error) {
	nodes, ok, err := v.evalNodeListRef(ctx, ec, e.Left)
	if !ok {
		return evalSimpleMapExpr(v.evalExpr, ctx, ec, e)
	}
	if err != nil {
		return nil, err
	}
	return simpleMapNodes(v.evalExpr, ctx, ec, e, nodes)
}

// evalPathStepExpr evaluates E1/E2 for a non-axis E2, reading E1 as a node
// list when it is a node-list producer.
func (v *vm) evalPathStepExpr(ctx context.Context, ec *evalContext, e PathStepExpr) (Sequence, error) {
	nodes, ok, err := v.evalNodeListRef(ctx, ec, e.Left)
	if !ok {
		return evalPathStepExpr(v.evalExpr, ctx, ec, e)
	}
	if err != nil {
		return nil, err
	}
	return pathStepFromNodes(v.evalExpr, ctx, ec, e, nodes)
}
