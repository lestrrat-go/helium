package xpath3

import (
	"context"
	"fmt"
	"math/big"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
	ls, err := concatToString(ctx, left)
	if err != nil {
		return nil, err
	}
	rs, err := concatToString(ctx, right)
	if err != nil {
		return nil, err
	}
	return SingleString(ls + rs), nil
}

// concatToString converts a sequence to string for the || operator. Arrays are
// flattened to their members during atomization; only function and map items,
// which have no string value, raise FOTY0014.
func concatToString(ctx context.Context, seq Sequence) (string, error) {
	if seqLen(seq) == 0 {
		return "", nil
	}
	// seqToStringErr atomizes first (flattening arrays, so an empty-array member
	// contributes nothing) and applies cardinality afterwards. Function and map
	// items have no atomized string value for ||, so reject them up front.
	for item := range seqItems(seq) {
		switch item.(type) {
		case FunctionItem, MapItem:
			return "", &XPathError{Code: errCodeFOTY0014, Message: "cannot get string value of function or map item"}
		}
	}
	return seqToStringErr(ctx, seq)
}

// appendBounded appends src to dst, enforcing the configured sequence/node-set
// size limit. Returns ErrNodeSetLimit once the accumulated length would exceed
// maxNodes. Used by accumulation sites (simple-map, FLWOR return, lookup) that
// concatenate per-item sub-sequences and could otherwise materialize unbounded
// output regardless of maxNodes.
func appendBounded(dst ItemSlice, src []Item, maxNodes int) (ItemSlice, error) {
	if maxNodes > 0 && len(dst)+len(src) > maxNodes {
		return nil, ErrNodeSetLimit
	}
	return append(dst, src...), nil
}

// appendBoundedSeq appends the items of src to dst one at a time, enforcing the
// configured sequence/node-set size limit before each append. Unlike
// appendBounded (which takes an already-materialized []Item), it iterates src
// lazily via seqItems so a callback returning an unbounded lazy Sequence is
// rejected with ErrNodeSetLimit as soon as the accumulated length would exceed
// maxNodes — without ever materializing the whole source sequence.
//
// It also charges one op (and honors cancellation) per appended item via
// fnCountOp, so draining a large lazy source respects OpLimit / context
// cancellation, and never runs to completion unbounded.
func appendBoundedSeq(ctx context.Context, ec *evalContext, dst ItemSlice, src Sequence, maxNodes int) (ItemSlice, error) {
	for item := range seqItems(src) {
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
		if maxNodes > 0 && len(dst)+1 > maxNodes {
			return nil, ErrNodeSetLimit
		}
		dst = append(dst, item)
	}
	return dst, nil
}

// appendBoundedClonedSeq behaves like appendBoundedSeq but appends a defensive
// deep clone of each item, and never the item itself. It is used by lookup /
// map:get paths that drain a BORROWED (no-clone) stored map value via get0:
// draining lazily charges op (and honors cancellation) and enforces maxNodes
// BEFORE each item is appended, so a borrowed lazy value whose Materialize is
// unbounded/panics trips the bound first; cloning each appended item then keeps
// value semantics so mutating the exposed result cannot reach the source map.
func appendBoundedClonedSeq(ctx context.Context, ec *evalContext, dst ItemSlice, src Sequence, maxNodes int) (ItemSlice, error) {
	for item := range seqItems(src) {
		if err := fnCountOp(ctx, ec); err != nil {
			return nil, err
		}
		if maxNodes > 0 && len(dst)+1 > maxNodes {
			return nil, ErrNodeSetLimit
		}
		dst = append(dst, deepCloneItem(item))
	}
	return dst, nil
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
		// Accumulate the right-hand sequence lazily so the aggregate maxNodes /
		// OpLimit / cancellation bound is enforced item-by-item BEFORE the whole
		// sub-sequence is materialized — a per-item right expression producing a
		// large lazy Sequence is rejected without first allocating it in full.
		result, err = appendBoundedSeq(ctx, ec, result, r, ec.maxNodes)
		if err != nil {
			return nil, err
		}
		i++
	}
	return result, nil
}

// simpleMapNodes evaluates E1 ! E2 for the E1 that evaluates to the node
// items of nodes, as evalSimpleMapExpr does: each node in turn is the
// context node.
func simpleMapNodes(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e SimpleMapExpr, nodes []helium.Node) (Sequence, error) {
	var result ItemSlice
	for i, n := range nodes {
		frame := ec.pushNodeContext(n, i+1, len(nodes))
		r, err := evalFn(ctx, ec, e.Right)
		ec.restoreContext(frame)
		if err != nil {
			return nil, err
		}
		result, err = appendBoundedSeq(ctx, ec, result, r, ec.maxNodes)
		if err != nil {
			return nil, err
		}
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
	// Atomize through the stream so a schema-typed list/union node expands; the
	// at-most-one cardinality is checked on the atomized result.
	sAtoms, err := atomizeSingletonOperand(startSeq)
	if err != nil {
		return nil, err
	}
	eAtoms, err := atomizeSingletonOperand(endSeq)
	if err != nil {
		return nil, err
	}
	if len(sAtoms) == 0 || len(eAtoms) == 0 {
		return validNilSequence, nil
	}
	// XPath 1.0 compatibility mode: a range bound keeps only its first atom rather
	// than raising a cardinality error.
	if ec.xpath10CompatMode() {
		sAtoms = sAtoms[:1]
		eAtoms = eAtoms[:1]
	}
	if len(sAtoms) > 1 || len(eAtoms) > 1 {
		return nil, &XPathError{Code: lexicon.ErrXPTY0004, Message: "to operator operands must each be xs:integer? (at most one item)"}
	}
	sa := sAtoms[0]
	ea := eAtoms[0]
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
		// Compute the span with big.Int: ev-sv+1 can overflow int64 even when
		// both bounds fit (e.g. minInt64 .. maxInt64), which would wrap the
		// count and bypass the limit check or build a negative-length range.
		span := new(big.Int).Sub(big.NewInt(ev), big.NewInt(sv))
		span.Add(span, big.NewInt(1))
		if !span.IsInt64() || span.Int64() > int64(ec.maxNodes) {
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
	// Gather both operands into one buffer, left then right, so the union
	// can return it as is when it is already in document order.
	nodes, ok := appendSequenceNodes(make([]helium.Node, 0, seqLen(left)+seqLen(right)), left)
	if !ok {
		return nil, ErrUnionNotNodeSet
	}
	split := len(nodes)
	nodes, ok = appendSequenceNodes(nodes, right)
	if !ok {
		return nil, ErrUnionNotNodeSet
	}
	merged, err := ixpath.UnionNodeSets(nodes, split, ec.docOrder, ec.maxNodes)
	if err != nil {
		return nil, err
	}
	result := make(ItemSlice, len(merged))
	for i, n := range merged {
		result[i] = nodeItemFor(ctx, ec, n)
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
	result, err := intersectExceptNodes(ec, e.Op, leftNodes, rightNodes)
	if err != nil {
		return nil, err
	}
	return nodeItemsFor(ctx, ec, result), nil
}

// intersectExceptNodes returns the nodes of leftNodes that are (op is
// TokenIntersect) or are not (TokenExcept) in rightNodes, in document order
// and free of duplicates.
func intersectExceptNodes(ec *evalContext, op TokenType, leftNodes, rightNodes []helium.Node) ([]helium.Node, error) {
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
		if op == TokenIntersect && inRight {
			result = append(result, n)
		} else if op == TokenExcept && !inRight {
			result = append(result, n)
		}
	}
	// XPath requires intersect/except results in document order
	return ixpath.DeduplicateNodes(result, ec.docOrder, ec.maxNodes)
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
			result[i] = nodeItemFor(ctx, ec, n)
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
	if filterPreservesOrder(ctx, ec, e.Filter) {
		deduped, err = ixpath.DeduplicateNodesPreserveOrder(result, ec.maxNodes)
	} else {
		deduped, err = ixpath.DeduplicateNodes(result, ec.docOrder, ec.maxNodes)
	}
	if err != nil {
		return nil, err
	}
	seq := make(ItemSlice, len(deduped))
	for i, n := range deduped {
		seq[i] = nodeItemFor(ctx, ec, n)
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
	nodes, err := vmPathStepNodes(evalFn, ctx, ec, e, baseNodes)
	if err != nil {
		return nil, err
	}
	return nodeItemsFor(ctx, ec, nodes), nil
}

// vmPathStepNodes evaluates the location path e.Path from every node of
// baseNodes and returns the results merged in document order, or in the
// order of baseNodes when e.OrderFilter is fn:reverse or fn:sort.
func vmPathStepNodes(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e vmPathExpr, baseNodes []helium.Node) ([]helium.Node, error) {
	result := make([]helium.Node, 0, len(baseNodes))
	for _, n := range baseNodes {
		frame := ec.pushNodeContext(n, 1, 1)
		subNodes, err := evalVMLocationPathNodes(evalFn, ctx, ec, *e.Path)
		ec.restoreContext(frame)
		if err != nil {
			return nil, err
		}
		result = append(result, subNodes...)
	}
	if filterPreservesOrder(ctx, ec, e.OrderFilter) {
		return ixpath.DeduplicateNodesPreserveOrder(result, ec.maxNodes)
	}
	return ixpath.DeduplicateNodes(result, ec.docOrder, ec.maxNodes)
}

// filterPreservesOrder returns true if the filter expression is a call to the
// built-in fn:reverse or fn:sort, which explicitly control sequence order. In
// these cases, a subsequent path step (/@attr) should preserve the caller's
// ordering, re-sorting into no document order.
//
// The decision is made against the runtime function resolver so that a rebound
// "fn" prefix or a user function overriding the built-in is not mistaken for
// the order-controlling built-in. (Do not use the lexical streamability helper
// StreamFnLocalName here — that is a static spelling check, not a binding.)
func filterPreservesOrder(ctx context.Context, ec *evalContext, e Expr) bool {
	fc, ok := e.(FunctionCall)
	if !ok {
		return false
	}
	r, err := resolveFunctionInfo(ctx, ec, fc.Prefix, fc.Name, len(fc.Args))
	if err != nil {
		return false
	}
	if !r.isBuiltin || r.uri != NSFn {
		return false
	}
	return r.name == "reverse" || r.name == "sort"
}

// evalPathStepExpr evaluates E1/E2 where E2 is a non-axis expression.
// Per XPath 3.1 §3.3.1.1: E1 must produce a node sequence (XPTY0019); E2 is
// evaluated for each node with that node as context. If all results are nodes,
// they are sorted in document order and deduplicated; if all are non-nodes,
// they keep their order. Any mix of nodes and non-nodes, within one result of
// E2 or across the results for different context nodes, raises XPTY0018. An
// empty result counts as either kind.
func evalPathStepExpr(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e PathStepExpr) (Sequence, error) {
	base, err := evalFn(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	baseNodes, ok := NodesFrom(base)
	if !ok {
		return nil, ErrPathNotNodeSet
	}
	return pathStepFromNodes(evalFn, ctx, ec, e, baseNodes)
}

// pathStepFromNodes evaluates E1/E2 for a non-axis E2 and the E1 whose
// nodes are baseNodes.
func pathStepFromNodes(evalFn exprEvaluator, ctx context.Context, ec *evalContext, e PathStepExpr, baseNodes []helium.Node) (Sequence, error) {
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
		if seqLen(r) == 0 {
			continue
		}
		rNodes, nok := NodesFrom(r)
		if nok && !hasNonNodes {
			hasNodes = true
			allNodes = append(allNodes, rNodes...)
			continue
		}
		// r is all nodes after earlier non-nodes, or it holds a non-node after
		// earlier nodes or next to a node of its own.
		if nok || hasNodes || sequenceHasNode(r) {
			return nil, &XPathError{Code: errCodeXPTY0018, Message: errMsgPathMixedResult}
		}
		hasNonNodes = true
		// Accumulate through the bounded helper so maxNodes / OpLimit /
		// cancellation are enforced on the atomic branch just as they are on
		// the node branch below; otherwise a per-node right expression such as
		// (1 to 10000000) could grow allItems past ec.maxNodes unbounded.
		allItems, err = appendBoundedSeq(ctx, ec, allItems, r, ec.maxNodes)
		if err != nil {
			return nil, err
		}
	}

	if hasNodes {
		var err error
		if filterPreservesOrder(ctx, ec, e.Left) {
			allNodes, err = ixpath.DeduplicateNodesPreserveOrder(allNodes, ec.maxNodes)
		} else {
			allNodes, err = ixpath.DeduplicateNodes(allNodes, ec.docOrder, ec.maxNodes)
		}
		if err != nil {
			return nil, err
		}
		return nodeItemsFor(ctx, ec, allNodes), nil
	}
	return allItems, nil
}

// sequenceHasNode reports whether seq holds at least one node.
func sequenceHasNode(seq Sequence) bool {
	for item := range seqItems(seq) {
		if _, ok := item.(NodeItem); ok {
			return true
		}
	}
	return false
}
