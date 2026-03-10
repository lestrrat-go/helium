package xpath3

import (
	"fmt"
	"math/big"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

func evalBinaryExpr(ec *evalContext, e BinaryExpr) (Sequence, error) {
	switch e.Op {
	case TokenOr:
		return evalLogicOr(ec, e)
	case TokenAnd:
		return evalLogicAnd(ec, e)
	case TokenEquals, TokenNotEquals, TokenLess, TokenLessEq, TokenGreater, TokenGreaterEq:
		return evalGeneralComparison(ec, e)
	case TokenEq, TokenNe, TokenLt, TokenLe, TokenGt, TokenGe:
		return evalValueComparison(ec, e)
	case TokenIs, TokenNodePre, TokenNodeFol:
		return evalNodeComparison(ec, e)
	case TokenPlus, TokenMinus, TokenStar, TokenDiv, TokenIdiv, TokenMod:
		return evalArithmetic(ec, e)
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedBinaryOp, e.Op)
}

func evalLogicOr(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
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
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	rb, err := EBV(right)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(rb), nil
}

func evalLogicAnd(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
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
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	rb, err := EBV(right)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(rb), nil
}

func evalConcatExpr(ec *evalContext, e ConcatExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	ls := seqToString(left)
	rs := seqToString(right)
	return SingleString(ls + rs), nil
}

func evalSimpleMapExpr(ec *evalContext, e SimpleMapExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	var result Sequence
	size := len(left)
	for i, item := range left {
		var subCtx *evalContext
		if ni, ok := item.(NodeItem); ok {
			subCtx = ec.withNode(ni.Node, i+1, size)
		} else {
			subCtx = ec.withContextItem(item, i+1, size)
		}
		r, err := eval(subCtx, e.Right)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

func evalRangeExpr(ec *evalContext, e RangeExpr) (Sequence, error) {
	startSeq, err := eval(ec, e.Start)
	if err != nil {
		return nil, err
	}
	endSeq, err := eval(ec, e.End)
	if err != nil {
		return nil, err
	}
	if len(startSeq) == 0 || len(endSeq) == 0 {
		return nil, nil
	}
	sa, err := AtomizeItem(startSeq[0])
	if err != nil {
		return nil, err
	}
	ea, err := AtomizeItem(endSeq[0])
	if err != nil {
		return nil, err
	}
	// Cast operands to xs:integer per spec (rejects NaN, Inf, non-numeric)
	saInt, castErr := CastAtomic(sa, TypeInteger)
	if castErr != nil {
		return nil, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("range operand: %v", castErr)}
	}
	eaInt, castErr := CastAtomic(ea, TypeInteger)
	if castErr != nil {
		return nil, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("range operand: %v", castErr)}
	}
	start := saInt.BigInt()
	end := eaInt.BigInt()
	if start.Cmp(end) > 0 {
		return nil, nil
	}
	count := new(big.Int).Sub(end, start)
	count.Add(count, big.NewInt(1))
	if !count.IsInt64() || count.Int64() > int64(ec.maxNodes) {
		return nil, ErrNodeSetLimit
	}
	n := count.Int64()
	result := make(Sequence, 0, n)
	cur := new(big.Int).Set(start)
	for cur.Cmp(end) <= 0 {
		result = append(result, AtomicValue{TypeName: TypeInteger, Value: new(big.Int).Set(cur)})
		cur.Add(cur, big.NewInt(1))
	}
	return result, nil
}

func evalUnionExpr(ec *evalContext, e UnionExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
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
	result := make(Sequence, len(merged))
	for i, n := range merged {
		result[i] = NodeItem{Node: n}
	}
	return result, nil
}

func evalIntersectExceptExpr(ec *evalContext, e IntersectExceptExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	leftNodes, ok1 := NodesFrom(left)
	rightNodes, ok2 := NodesFrom(right)
	if !ok1 || !ok2 {
		return nil, ErrUnionNotNodeSet
	}
	rightSet := make(map[helium.Node]bool, len(rightNodes))
	for _, n := range rightNodes {
		rightSet[n] = true
	}
	var result []helium.Node
	for _, n := range leftNodes {
		inRight := rightSet[n]
		if e.Op == TokenIntersect && inRight {
			result = append(result, n)
		} else if e.Op == TokenExcept && !inRight {
			result = append(result, n)
		}
	}
	seq := make(Sequence, len(result))
	for i, n := range result {
		seq[i] = NodeItem{Node: n}
	}
	return seq, nil
}

func evalFilterExpr(ec *evalContext, e FilterExpr) (Sequence, error) {
	base, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}

	// Try node-set path (optimized for node predicates)
	if nodes, ok := NodesFrom(base); ok {
		for _, pred := range e.Predicates {
			nodes, err = applyPredicate(ec, nodes, pred)
			if err != nil {
				return nil, err
			}
		}
		result := make(Sequence, len(nodes))
		for i, n := range nodes {
			result[i] = NodeItem{Node: n}
		}
		return result, nil
	}

	// General sequence filtering (XPath 3.1)
	seq := base
	for _, pred := range e.Predicates {
		seq, err = applySequencePredicate(ec, seq, pred)
		if err != nil {
			return nil, err
		}
	}
	return seq, nil
}

// applySequencePredicate filters a sequence by a predicate expression.
// Each item becomes the context item; numeric predicates select by position.
func applySequencePredicate(ec *evalContext, seq Sequence, pred Expr) (Sequence, error) {
	var result Sequence
	size := len(seq)
	for i, item := range seq {
		var subCtx *evalContext
		if ni, ok := item.(NodeItem); ok {
			subCtx = ec.withNode(ni.Node, i+1, size)
		} else {
			subCtx = ec.withContextItem(item, i+1, size)
		}
		r, err := eval(subCtx, pred)
		if err != nil {
			return nil, err
		}
		// Numeric predicate: position match
		if len(r) == 1 {
			if a, ok := r[0].(AtomicValue); ok && a.IsNumeric() {
				f := a.ToFloat64()
				if f == float64(i+1) {
					result = append(result, item)
				}
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
	}
	return result, nil
}

func evalPathExpr(ec *evalContext, e PathExpr) (Sequence, error) {
	base, err := eval(ec, e.Filter)
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
	var result []helium.Node
	for _, n := range baseNodes {
		subCtx := ec.withNode(n, 1, 1)
		subResult, err := evalLocationPath(subCtx, e.Path)
		if err != nil {
			return nil, err
		}
		subNodes, _ := NodesFrom(subResult)
		result, err = ixpath.MergeNodeSets(result, subNodes, ec.docOrder, ec.maxNodes)
		if err != nil {
			return nil, err
		}
	}
	seq := make(Sequence, len(result))
	for i, n := range result {
		seq[i] = NodeItem{Node: n}
	}
	return seq, nil
}

// evalPathStepExpr evaluates E1/E2 where E2 is a non-axis expression.
// Per XPath 3.1: E1 must produce a node sequence; E2 is evaluated for each node
// with that node as context. If all results are nodes, they are sorted in
// document order and deduplicated.
func evalPathStepExpr(ec *evalContext, e PathStepExpr) (Sequence, error) {
	base, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	baseNodes, ok := NodesFrom(base)
	if !ok {
		return nil, ErrPathNotNodeSet
	}
	var allNodes []helium.Node
	var allItems Sequence
	isNodeResult := true

	for i, n := range baseNodes {
		subCtx := ec.withNode(n, i+1, len(baseNodes))
		r, err := eval(subCtx, e.Right)
		if err != nil {
			return nil, err
		}
		if isNodeResult {
			rNodes, nok := NodesFrom(r)
			if nok {
				allNodes, err = ixpath.MergeNodeSets(allNodes, rNodes, ec.docOrder, ec.maxNodes)
				if err != nil {
					return nil, err
				}
				continue
			}
			// First non-node result — switch to item mode
			isNodeResult = false
			// Convert previously collected nodes to items
			for _, pn := range allNodes {
				allItems = append(allItems, NodeItem{Node: pn})
			}
			allNodes = nil
		}
		allItems = append(allItems, r...)
	}

	if isNodeResult {
		seq := make(Sequence, len(allNodes))
		for i, n := range allNodes {
			seq[i] = NodeItem{Node: n}
		}
		return seq, nil
	}
	return allItems, nil
}
