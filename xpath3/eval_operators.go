package xpath3

import (
	"context"
	"fmt"
	"math/big"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func evalBinaryExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e BinaryExpr) (Sequence, error) {
	switch e.Op {
	case TokenOr:
		return evalLogicOr(evalFn, ctx, ec, e)
	case TokenAnd:
		return evalLogicAnd(evalFn, ctx, ec, e)
	case TokenEquals, TokenNotEquals, TokenLess, TokenLessEq, TokenGreater, TokenGreaterEq:
		return evalGeneralComparison(evalFn, ctx, ec, e)
	case TokenEq, TokenNe, TokenLt, TokenLe, TokenGt, TokenGe:
		return evalValueComparison(evalFn, ctx, ec, e)
	case TokenIs, TokenNodePre, TokenNodeFol:
		return evalNodeComparison(evalFn, ctx, ec, e)
	case TokenPlus, TokenMinus, TokenStar, TokenDiv, TokenIdiv, TokenMod:
		return evalArithmetic(evalFn, ctx, ec, e)
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedBinaryOp, e.Op)
}

func evalLogicOr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := evalFn(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	lb, err := EBV(left)
	if err != nil {
		return nil, err
	}
	if lb {
		return SingleBoolean(true), nil
	}
	right, err := evalFn(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	rb, err := EBV(right)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(rb), nil
}

func evalLogicAnd(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := evalFn(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	lb, err := EBV(left)
	if err != nil {
		return nil, err
	}
	if !lb {
		return SingleBoolean(false), nil
	}
	right, err := evalFn(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	rb, err := EBV(right)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(rb), nil
}

func evalConcatExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e ConcatExpr) (Sequence, error) {
	left, err := evalFn(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := evalFn(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	ls, err := concatToString(left)
	if err != nil {
		return nil, err
	}
	rs, err := concatToString(right)
	if err != nil {
		return nil, err
	}
	return SingleString(ls + rs), nil
}

// concatToString converts a sequence to string for the || operator.
// Raises FOTY0014 for function/map/array items that have no string value.
func concatToString(seq Sequence) (string, error) {
	if seqLen(seq) == 0 {
		return "", nil
	}
	if seq.Len() > 1 {
		return "", &XPathError{Code: errCodeXPTY0004, Message: "cannot get string value of sequence of length > 1"}
	}
	switch seq.Get(0).(type) {
	case FunctionItem:
		return "", &XPathError{Code: errCodeFOTY0014, Message: "cannot get string value of function item"}
	case MapItem:
		return "", &XPathError{Code: errCodeFOTY0014, Message: "cannot get string value of map item"}
	case ArrayItem:
		return "", &XPathError{Code: errCodeFOTY0014, Message: "cannot get string value of array item"}
	}
	return seqToStringErr(seq)
}

func evalSimpleMapExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e SimpleMapExpr) (Sequence, error) {
	left, err := evalFn(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	var result ItemSlice
	size := seqLen(left)
	i := 0
	for item := range seqItems(left) {
		var frame evalContextFrame
		if ni, ok := item.(NodeItem); ok {
			frame = ec.pushNodeContext(ni.Node, i+1, size)
		} else {
			frame = ec.pushContextItem(item, i+1, size)
		}
		r, err := evalFn(ctx, ec, e.Right)
		ec.restoreContext(frame)
		if err != nil {
			return nil, err
		}
		result = append(result, seqMaterialize(r)...)
		i++
	}
	return result, nil
}

func evalRangeExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e RangeExpr) (Sequence, error) {
	startSeq, err := evalFn(ctx, ec, e.Start)
	if err != nil {
		return nil, err
	}
	endSeq, err := evalFn(ctx, ec, e.End)
	if err != nil {
		return nil, err
	}
	if seqLen(startSeq) == 0 || seqLen(endSeq) == 0 {
		return validNilSequence, nil
	}
	sa, err := AtomizeItem(startSeq.Get(0))
	if err != nil {
		return nil, err
	}
	ea, err := AtomizeItem(endSeq.Get(0))
	if err != nil {
		return nil, err
	}
	// Per spec, operands are converted using function coercion rules for xs:integer?.
	// This allows untypedAtomic → integer, but rejects double/float/decimal → integer.
	saInt, err := coerceToInteger(sa)
	if err != nil {
		return nil, err
	}
	eaInt, err := coerceToInteger(ea)
	if err != nil {
		return nil, err
	}
	// Fast path: both bounds fit in int64
	sv, sok := saInt.Int64Val()
	ev, eok := eaInt.Int64Val()
	if sok && eok {
		if sv > ev {
			return validNilSequence, nil
		}
		count := ev - sv + 1
		if count > int64(ec.maxNodes) {
			return nil, ErrNodeSetLimit
		}
		return NewRangeSequence(sv, ev), nil
	}
	start := saInt.BigInt()
	end := eaInt.BigInt()
	if start.Cmp(end) > 0 {
		return validNilSequence, nil
	}
	count := new(big.Int).Sub(end, start)
	count.Add(count, big.NewInt(1))
	if !count.IsInt64() || count.Int64() > int64(ec.maxNodes) {
		return nil, ErrNodeSetLimit
	}
	// Fallback for big integer ranges (rare)
	n := count.Int64()
	result := make(ItemSlice, 0, n)
	cur := new(big.Int).Set(start)
	for cur.Cmp(end) <= 0 {
		result = append(result, AtomicValue{TypeName: TypeInteger, Value: new(big.Int).Set(cur)})
		cur.Add(cur, big.NewInt(1))
	}
	return result, nil
}

func evalUnionExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e UnionExpr) (Sequence, error) {
	left, err := evalFn(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := evalFn(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	leftNodes, ok1 := NodesFrom(left)
	rightNodes, ok2 := NodesFrom(right)
	if !ok1 || !ok2 {
		return nil, ErrUnionNotNodeSet
	}
	merged, err := ixpath.MergeNodeSets(leftNodes, rightNodes, ec.docOrder, ec.maxNodes)
	if err != nil {
		return nil, err
	}
	result := make(ItemSlice, len(merged))
	for i, n := range merged {
		result[i] = nodeItemFor(ec, n)
	}
	return result, nil
}

func evalIntersectExceptExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e IntersectExceptExpr) (Sequence, error) {
	left, err := evalFn(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := evalFn(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	leftNodes, ok1 := NodesFrom(left)
	rightNodes, ok2 := NodesFrom(right)
	if !ok1 || !ok2 {
		return nil, ErrUnionNotNodeSet
	}
	rightSet := make(map[nodeIdentityKey]struct{}, len(rightNodes))
	for _, n := range rightNodes {
		rightSet[makeNodeIdentityKey(n)] = struct{}{}
	}
	seen := make(map[nodeIdentityKey]struct{}, len(leftNodes))
	result := make([]helium.Node, 0, len(leftNodes))
	for _, n := range leftNodes {
		key := makeNodeIdentityKey(n)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		_, inRight := rightSet[key]
		if e.Op == TokenIntersect && inRight {
			result = append(result, n)
		} else if e.Op == TokenExcept && !inRight {
			result = append(result, n)
		}
	}
	// XPath requires intersect/except results in document order
	result, err = ixpath.DeduplicateNodes(result, ec.docOrder, ec.maxNodes)
	if err != nil {
		return nil, err
	}
	seq := make(ItemSlice, len(result))
	for i, n := range result {
		seq[i] = nodeItemFor(ec, n)
	}
	return seq, nil
}

func evalFilterExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e FilterExpr) (Sequence, error) {
	base, err := evalFn(ctx, ec, e.Expr)
	if err != nil {
		return nil, err
	}

	// Try node-set path (optimized for node predicates)
	if nodes, ok := NodesFrom(base); ok {
		for _, pred := range e.Predicates {
			nodes, err = applyPredicate(evalFn, ctx, ec, nodes, pred)
			if err != nil {
				return nil, err
			}
		}
		result := make(ItemSlice, len(nodes))
		for i, n := range nodes {
			result[i] = nodeItemFor(ec, n)
		}
		return result, nil
	}

	// General sequence filtering (XPath 3.1)
	seq := base
	for _, pred := range e.Predicates {
		seq, err = applySequencePredicate(evalFn, ctx, ec, seq, pred)
		if err != nil {
			return nil, err
		}
	}
	return seq, nil
}

// applySequencePredicate filters a sequence by a predicate expression.
// Each item becomes the context item; numeric predicates select by position.
func applySequencePredicate(evalFn exprEvaluator, ctx context.Context, ec *evalContext, seq Sequence, pred Expr) (Sequence, error) {
	size := seqLen(seq)
	result := make(ItemSlice, 0, size)
	i := 0
	for item := range seqItems(seq) {
		var frame evalContextFrame
		if ni, ok := item.(NodeItem); ok {
			frame = ec.pushNodeContext(ni.Node, i+1, size)
		} else {
			frame = ec.pushContextItem(item, i+1, size)
		}
		r, err := evalFn(ctx, ec, pred)
		ec.restoreContext(frame)
		if err != nil {
			return nil, err
		}
		// Numeric predicate: position match
		if seqLen(r) == 1 {
			if a, ok := r.Get(0).(AtomicValue); ok && a.IsNumeric() {
				f := a.ToFloat64()
				if f == float64(i+1) {
					result = append(result, item)
				}
				i++
				continue
			}
		}
		// Boolean predicate
		b, err := EBV(r)
		if err != nil {
			return nil, err
		}
		if b {
			result = append(result, item)
		}
		i++
	}
	return result, nil
}

func evalPathExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e PathExpr) (Sequence, error) {
	base, err := evalFn(ctx, ec, e.Filter)
	if err != nil {
		return nil, err
	}
	if e.Path == nil {
		return base, nil
	}
	baseNodes, ok := NodesFrom(base)
	if !ok {
		return nil, ErrPathNotNodeSet
	}
	result := make([]helium.Node, 0, len(baseNodes))
	for _, n := range baseNodes {
		frame := ec.pushNodeContext(n, 1, 1)
		subResult, err := evalLocationPath(evalFn, ctx, ec, e.Path)
		ec.restoreContext(frame)
		if err != nil {
			return nil, err
		}
		subNodes, _ := NodesFrom(subResult)
		result = append(result, subNodes...)
	}
	// Per XPath 3.1 §3.3.5: E1/E2 returns nodes in document order.
	// However, when the filter is an ordering function (reverse, sort),
	// preserve the explicit ordering and only deduplicate.
	var deduped []helium.Node
	if filterPreservesOrder(e.Filter) {
		deduped, err = ixpath.DeduplicateNodesPreserveOrder(result, ec.maxNodes)
	} else {
		deduped, err = ixpath.DeduplicateNodes(result, ec.docOrder, ec.maxNodes)
	}
	if err != nil {
		return nil, err
	}
	seq := make(ItemSlice, len(deduped))
	for i, n := range deduped {
		seq[i] = nodeItemFor(ec, n)
	}
	return seq, nil
}

func evalVMPathExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e vmPathExpr) (Sequence, error) {
	base, err := evalFn(ctx, ec, e.Filter)
	if err != nil {
		return nil, err
	}
	if e.Path == nil {
		return base, nil
	}
	baseNodes, ok := NodesFrom(base)
	if !ok {
		return nil, ErrPathNotNodeSet
	}
	result := make([]helium.Node, 0, len(baseNodes))
	for _, n := range baseNodes {
		frame := ec.pushNodeContext(n, 1, 1)
		subResult, err := evalVMLocationPath(evalFn, ctx, ec, *e.Path)
		ec.restoreContext(frame)
		if err != nil {
			return nil, err
		}
		subNodes, _ := NodesFrom(subResult)
		result = append(result, subNodes...)
	}
	var deduped []helium.Node
	if e.PreserveOrder {
		deduped, err = ixpath.DeduplicateNodesPreserveOrder(result, ec.maxNodes)
	} else {
		deduped, err = ixpath.DeduplicateNodes(result, ec.docOrder, ec.maxNodes)
	}
	if err != nil {
		return nil, err
	}
	seq := make(ItemSlice, len(deduped))
	for i, n := range deduped {
		seq[i] = nodeItemFor(ec, n)
	}
	return seq, nil
}

// filterPreservesOrder returns true if the filter expression is a function
// call that explicitly controls sequence order (reverse, sort). In these
// cases, a subsequent path step (/@attr) should preserve the caller's
// ordering rather than re-sorting into document order.
func filterPreservesOrder(e Expr) bool {
	fc, ok := e.(FunctionCall)
	if !ok {
		return false
	}
	if fc.Prefix != "" {
		return false
	}
	return fc.Name == "reverse" || fc.Name == "sort"
}

// evalPathStepExpr evaluates E1/E2 where E2 is a non-axis expression.
// Per XPath 3.1: E1 must produce a node sequence; E2 is evaluated for each node
// with that node as context. If all results are nodes, they are sorted in
// document order and deduplicated.
func evalPathStepExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e PathStepExpr) (Sequence, error) {
	base, err := evalFn(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	baseNodes, ok := NodesFrom(base)
	if !ok {
		return nil, ErrPathNotNodeSet
	}
	allNodes := make([]helium.Node, 0, len(baseNodes))
	allItems := make(ItemSlice, 0, len(baseNodes))
	hasNodes := false
	hasNonNodes := false

	for i, n := range baseNodes {
		frame := ec.pushNodeContext(n, i+1, len(baseNodes))
		r, err := evalFn(ctx, ec, e.Right)
		ec.restoreContext(frame)
		if err != nil {
			return nil, err
		}
		rNodes, nok := NodesFrom(r)
		if nok {
			if len(rNodes) > 0 {
				// XPTY0018: path expression returns mix of nodes and non-nodes.
				if hasNonNodes {
					return nil, fmt.Errorf("XPTY0018: path expression result contains a mix of nodes and non-nodes")
				}
				hasNodes = true
				allNodes = append(allNodes, rNodes...)
			}
		} else {
			// Check for mixed results.
			if hasNodes {
				return nil, fmt.Errorf("XPTY0018: path expression result contains a mix of nodes and non-nodes")
			}
			hasNonNodes = true
			allItems = append(allItems, seqMaterialize(r)...)
		}
	}

	if hasNodes {
		if filterPreservesOrder(e.Left) {
			allNodes, err = ixpath.DeduplicateNodesPreserveOrder(allNodes, ec.maxNodes)
		} else {
			allNodes, err = ixpath.DeduplicateNodes(allNodes, ec.docOrder, ec.maxNodes)
		}
		if err != nil {
			return nil, err
		}
		seq := make(ItemSlice, len(allNodes))
		for i, n := range allNodes {
			seq[i] = nodeItemFor(ec, n)
		}
		return seq, nil
	}
	return allItems, nil
}
