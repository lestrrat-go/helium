package xpath3

import (
	"context"

	"github.com/lestrrat-go/helium"
)

// This file lets VM instructions that only need the nodes of a location path
// read them as a node list. A location path evaluates to a []helium.Node;
// returning it as a Sequence wraps every node in a NodeItem, which costs one
// allocation per node. fn:count and its relatives, and a filter expression
// over a location path, read the node list directly instead. They evaluate
// the path exactly as v.evalExpr does, with the same recursion accounting, so
// results, errors and limits do not change.

// evalLocationPathRef evaluates expr the way v.evalExpr does when expr refers
// to a compiled location path, and returns the path's nodes. ok reports
// whether expr is such a reference; when it is false nothing was evaluated
// and the caller evaluates expr itself.
func (v *vm) evalLocationPathRef(ctx context.Context, ec *evalContext, expr Expr) ([]helium.Node, bool, error) {
	ref, ok := expr.(compiledExprRef)
	if !ok || ref.index < 0 || ref.index >= len(v.program.instructions) {
		return nil, false, nil
	}
	inst := v.program.instructions[ref.index]
	if inst.op != vmOpLocationPath {
		return nil, false, nil
	}
	lp, ok := AsExpr[vmLocationPathExpr](inst.payload)
	if !ok {
		return nil, false, nil
	}
	// The recursion accounting of evalWith: no evaluation panics past a
	// recover, so decrementing after the call matches its deferred decrement.
	ec.depth++
	if ec.maxRecursionDepth > 0 && ec.depth > ec.maxRecursionDepth {
		return nil, true, ErrRecursionLimit
	}
	nodes, err := evalVMLocationPathNodes(v.evalExpr, ctx, ec, lp)
	ec.depth--
	return nodes, true, err
}

// evalFunctionCall evaluates a static function call. When the call has one
// argument and it is a location path, the path is evaluated to its node list
// first, and the built-in fn:count, fn:exists, fn:empty, fn:boolean, fn:not
// and fn:head compute their result from that list. Every other function gets
// the node items evalFunctionCall would pass it.
func (v *vm) evalFunctionCall(ctx context.Context, ec *evalContext, e FunctionCall) (Sequence, error) {
	if len(e.Args) != 1 {
		return evalFunctionCall(v.evalExpr, ctx, ec, e)
	}
	nodes, ok, err := v.evalLocationPathRef(ctx, ec, e.Args[0])
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

// evalFilterExpr evaluates a filter expression. When the filtered expression
// is a location path, its node list goes straight to the predicates, which
// evalFilterExpr would reach through a sequence of node items.
func (v *vm) evalFilterExpr(ctx context.Context, ec *evalContext, e FilterExpr) (Sequence, error) {
	nodes, ok, err := v.evalLocationPathRef(ctx, ec, e.Expr)
	if !ok {
		return evalFilterExpr(v.evalExpr, ctx, ec, e)
	}
	if err != nil {
		return nil, err
	}
	for _, pred := range e.Predicates {
		nodes, err = v.applyFilterPredicate(ctx, ec, nodes, pred)
		if err != nil {
			return nil, err
		}
	}
	return nodeItemsFor(ctx, ec, nodes), nil
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
