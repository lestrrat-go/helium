package xpath3

import (
	"context"
	"fmt"
	"math"
	"sort"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

const (
	maxRecursionDepth = ixpath.DefaultMaxRecursionDepth
	maxNodeSetLength  = ixpath.DefaultMaxNodeSetLength
)

// evalContext holds the evaluation state for an XPath 3.1 expression.
type evalContext struct {
	goCtx       context.Context
	node        helium.Node
	contextItem Item // non-nil when context item is not a node (simple map over atomics)
	position    int
	size        int
	vars        map[string]Sequence
	namespaces  map[string]string
	functions   map[string]Function
	fnsNS       map[QualifiedName]Function
	depth       int
	opCount     *int
	opLimit     int
	docOrder    *ixpath.DocOrderCache
	maxNodes    int
}

func newEvalContext(ctx context.Context, node helium.Node) *evalContext {
	opCount := 0
	ec := &evalContext{
		goCtx:    ctx,
		node:     node,
		position: 1,
		size:     1,
		opCount:  &opCount,
		docOrder: &ixpath.DocOrderCache{},
		maxNodes: maxNodeSetLength,
	}
	if xctx := GetContext(ctx); xctx != nil {
		ec.namespaces = xctx.namespaces
		ec.vars = xctx.variables
		ec.opLimit = xctx.opLimit
		ec.functions = xctx.functions
		ec.fnsNS = xctx.functionsNS
	}
	return ec
}

func (ec *evalContext) withNode(n helium.Node, pos, size int) *evalContext {
	return &evalContext{
		goCtx:      ec.goCtx,
		node:       n,
		position:   pos,
		size:       size,
		vars:       ec.vars,
		namespaces: ec.namespaces,
		functions:  ec.functions,
		fnsNS:      ec.fnsNS,
		depth:      ec.depth,
		opCount:    ec.opCount,
		opLimit:    ec.opLimit,
		docOrder:   ec.docOrder,
		maxNodes:   ec.maxNodes,
	}
}

// withContextItem sets a non-node context item (for simple map, etc.)
func (ec *evalContext) withContextItem(item Item, pos, size int) *evalContext {
	return &evalContext{
		goCtx:       ec.goCtx,
		node:        ec.node,
		contextItem: item,
		position:    pos,
		size:        size,
		vars:        ec.vars,
		namespaces:  ec.namespaces,
		functions:   ec.functions,
		fnsNS:       ec.fnsNS,
		depth:       ec.depth,
		opCount:     ec.opCount,
		opLimit:     ec.opLimit,
		docOrder:    ec.docOrder,
		maxNodes:    ec.maxNodes,
	}
}

func (ec *evalContext) withVar(name string, val Sequence) *evalContext {
	newVars := make(map[string]Sequence, len(ec.vars)+1)
	for k, v := range ec.vars {
		newVars[k] = v
	}
	newVars[name] = val
	cp := *ec
	cp.vars = newVars
	return &cp
}

func (ec *evalContext) countOps(n int) error {
	if ec.opLimit <= 0 {
		return nil
	}
	*ec.opCount += n
	if *ec.opCount > ec.opLimit {
		return ErrOpLimit
	}
	return nil
}

// eval dispatches to the appropriate evaluator for each AST node type.
func eval(ec *evalContext, expr Expr) (Sequence, error) {
	ec.depth++
	if ec.depth > maxRecursionDepth {
		return nil, ErrRecursionLimit
	}
	defer func() { ec.depth-- }()

	switch e := expr.(type) {
	case LiteralExpr:
		return evalLiteral(e)
	case VariableExpr:
		return evalVariable(ec, e)
	case ContextItemExpr:
		if ec.contextItem != nil {
			return Sequence{ec.contextItem}, nil
		}
		if ec.node == nil {
			return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
		}
		return Sequence{NodeItem{Node: ec.node}}, nil
	case RootExpr:
		if ec.node == nil {
			return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
		}
		return Sequence{NodeItem{Node: ixpath.DocumentRoot(ec.node)}}, nil
	case SequenceExpr:
		return evalSequenceExpr(ec, e)
	case *LocationPath:
		return evalLocationPath(ec, e)
	case BinaryExpr:
		return evalBinaryExpr(ec, e)
	case UnaryExpr:
		return evalUnaryExpr(ec, e)
	case ConcatExpr:
		return evalConcatExpr(ec, e)
	case SimpleMapExpr:
		return evalSimpleMapExpr(ec, e)
	case RangeExpr:
		return evalRangeExpr(ec, e)
	case UnionExpr:
		return evalUnionExpr(ec, e)
	case IntersectExceptExpr:
		return evalIntersectExceptExpr(ec, e)
	case FilterExpr:
		return evalFilterExpr(ec, e)
	case PathExpr:
		return evalPathExpr(ec, e)
	case LookupExpr:
		return evalLookupExpr(ec, e)
	case UnaryLookupExpr:
		return evalUnaryLookupExpr(ec, e)
	case FLWORExpr:
		return evalFLWOR(ec, e)
	case QuantifiedExpr:
		return evalQuantifiedExpr(ec, e)
	case IfExpr:
		return evalIfExpr(ec, e)
	case TryCatchExpr:
		return evalTryCatchExpr(ec, e)
	case InstanceOfExpr:
		return evalInstanceOfExpr(ec, e)
	case CastExpr:
		return evalCastExpr(ec, e)
	case CastableExpr:
		return evalCastableExpr(ec, e)
	case TreatAsExpr:
		return evalTreatAsExpr(ec, e)
	case FunctionCall:
		return evalFunctionCall(ec, e)
	case DynamicFunctionCall:
		return evalDynamicFunctionCall(ec, e)
	case NamedFunctionRef:
		return evalNamedFunctionRef(ec, e)
	case InlineFunctionExpr:
		return evalInlineFunctionExpr(ec, e)
	case MapConstructorExpr:
		return evalMapConstructorExpr(ec, e)
	case ArrayConstructorExpr:
		return evalArrayConstructorExpr(ec, e)
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedExpr, expr)
	}
}

// --- 3.1: Basic dispatch ---

func evalLiteral(e LiteralExpr) (Sequence, error) {
	switch v := e.Value.(type) {
	case string:
		return SingleString(v), nil
	case float64:
		return SingleDouble(v), nil
	}
	return nil, fmt.Errorf("%w: literal %T", ErrUnsupportedExpr, e.Value)
}

func evalVariable(ec *evalContext, e VariableExpr) (Sequence, error) {
	if ec.vars != nil {
		if v, ok := ec.vars[e.Name]; ok {
			return v, nil
		}
	}
	return nil, fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
}

func evalSequenceExpr(ec *evalContext, e SequenceExpr) (Sequence, error) {
	if len(e.Items) == 0 {
		return nil, nil
	}
	var result Sequence
	for _, item := range e.Items {
		seq, err := eval(ec, item)
		if err != nil {
			return nil, err
		}
		result = append(result, seq...)
	}
	return result, nil
}

// --- 3.2: Location paths ---

func evalLocationPath(ec *evalContext, lp *LocationPath) (Sequence, error) {
	var nodes []helium.Node

	if lp.Absolute {
		if ec.node == nil {
			return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
		}
		root := ixpath.DocumentRoot(ec.node)
		nodes = []helium.Node{root}
	} else {
		if ec.node == nil {
			return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
		}
		nodes = []helium.Node{ec.node}
	}

	var err error
	for _, step := range lp.Steps {
		if len(step.Predicates) > 0 {
			nodes, err = evalStepWithPredicates(ec, nodes, step)
		} else {
			nodes, err = evalStepNoPredicates(ec, nodes, step)
		}
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

func evalStepWithPredicates(ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var allFiltered []helium.Node
	for _, n := range nodes {
		candidates, err := ixpath.TraverseAxis(step.Axis, n, ec.maxNodes)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(len(candidates)); err != nil {
			return nil, err
		}
		matched := filterByNodeTest(candidates, step.NodeTest, step.Axis, ec)
		for _, pred := range step.Predicates {
			matched, err = applyPredicate(ec, matched, pred)
			if err != nil {
				return nil, err
			}
		}
		allFiltered = append(allFiltered, matched...)
	}
	return ixpath.DeduplicateNodes(allFiltered, ec.docOrder, ec.maxNodes)
}

func evalStepNoPredicates(ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var next []helium.Node
	for _, n := range nodes {
		candidates, err := ixpath.TraverseAxis(step.Axis, n, ec.maxNodes)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(len(candidates)); err != nil {
			return nil, err
		}
		next = append(next, filterByNodeTest(candidates, step.NodeTest, step.Axis, ec)...)
	}
	return ixpath.DeduplicateNodes(next, ec.docOrder, ec.maxNodes)
}

func filterByNodeTest(candidates []helium.Node, nt NodeTest, axis AxisType, ec *evalContext) []helium.Node {
	var matched []helium.Node
	for _, c := range candidates {
		if matchNodeTest(nt, c, axis, ec) {
			matched = append(matched, c)
		}
	}
	return matched
}

func matchNodeTest(nt NodeTest, n helium.Node, axis AxisType, ec *evalContext) bool {
	switch test := nt.(type) {
	case NameTest:
		return matchNameTest(test, n, axis, ec)
	case TypeTest:
		return matchTypeTest(test, n)
	case PITest:
		if n.Type() != helium.ProcessingInstructionNode {
			return false
		}
		if test.Target == "" {
			return true
		}
		return n.Name() == test.Target
	case ElementTest:
		if n.Type() != helium.ElementNode {
			return false
		}
		if test.Name != "" && test.Name != "*" {
			if ixpath.LocalNameOf(n) != test.Name {
				return false
			}
		}
		return true
	case AttributeTest:
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
		if test.Name != "" && test.Name != "*" {
			if ixpath.LocalNameOf(n) != test.Name {
				return false
			}
		}
		return true
	case DocumentTest:
		if n.Type() != helium.DocumentNode {
			return false
		}
		if test.Inner != nil {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if matchNodeTest(test.Inner, c, AxisChild, ec) {
					return true
				}
			}
			return false
		}
		return true
	case NamespaceNodeTest:
		return n.Type() == helium.NamespaceNode
	case AnyItemTest:
		return true
	}
	return false
}

func matchNameTest(test NameTest, n helium.Node, axis AxisType, ec *evalContext) bool {
	switch axis {
	case AxisAttribute:
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
	case AxisNamespace:
		if n.Type() != helium.NamespaceNode {
			return false
		}
		if test.Local == "*" {
			return true
		}
		return n.Name() == test.Local
	default:
		if n.Type() != helium.ElementNode {
			return false
		}
	}

	if test.Local == "*" {
		if test.Prefix == "" {
			return true
		}
		return matchPrefix(test.Prefix, n, ec)
	}

	if ixpath.LocalNameOf(n) != test.Local {
		return false
	}

	if test.Prefix != "" {
		return matchPrefix(test.Prefix, n, ec)
	}
	return true
}

func matchPrefix(prefix string, n helium.Node, ec *evalContext) bool {
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[prefix]; ok {
			return ixpath.NodeNamespaceURI(n) == uri
		}
	}
	return ixpath.NodePrefix(n) == prefix
}

func matchTypeTest(test TypeTest, n helium.Node) bool {
	switch test.Kind {
	case NodeKindNode:
		return true
	case NodeKindText:
		return n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode
	case NodeKindComment:
		return n.Type() == helium.CommentNode
	case NodeKindProcessingInstruction:
		return n.Type() == helium.ProcessingInstructionNode
	}
	return false
}

func applyPredicate(ec *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
	if err := ec.countOps(len(nodes)); err != nil {
		return nil, err
	}
	size := len(nodes)
	var result []helium.Node
	for i, n := range nodes {
		pctx := ec.withNode(n, i+1, size)
		r, err := eval(pctx, pred)
		if err != nil {
			return nil, err
		}
		if predicateTrue(r, i+1) {
			result = append(result, n)
		}
	}
	return result, nil
}

// predicateTrue evaluates a predicate result per XPath spec:
// numeric → compare to position, otherwise → EBV.
func predicateTrue(r Sequence, position int) bool {
	if len(r) == 1 {
		if av, ok := r[0].(AtomicValue); ok && av.IsNumeric() {
			return av.ToFloat64() == float64(position)
		}
	}
	b, _ := EBV(r)
	return b
}

// --- 3.3: Binary operators (except comparison — see compare.go) ---

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

func evalArithmetic(ec *evalContext, e BinaryExpr) (Sequence, error) {
	left, err := eval(ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ec, e.Right)
	if err != nil {
		return nil, err
	}
	if len(left) == 0 || len(right) == 0 {
		return nil, nil // empty sequence
	}
	la, err := AtomizeItem(left[0])
	if err != nil {
		return nil, err
	}
	ra, err := AtomizeItem(right[0])
	if err != nil {
		return nil, err
	}
	ln := promoteToDouble(la)
	rn := promoteToDouble(ra)

	var result float64
	switch e.Op {
	case TokenPlus:
		result = ln + rn
	case TokenMinus:
		result = ln - rn
	case TokenStar:
		result = ln * rn
	case TokenDiv:
		result = ln / rn
	case TokenIdiv:
		if rn == 0 {
			return nil, &XPathError{Code: "FOAR0002", Message: "integer division by zero"}
		}
		result = math.Trunc(ln / rn)
		return SingleInteger(int64(result)), nil
	case TokenMod:
		result = math.Mod(ln, rn)
	}

	// Preserve integer type when both inputs are integer
	if la.TypeName == TypeInteger && ra.TypeName == TypeInteger && e.Op != TokenDiv {
		return SingleInteger(int64(result)), nil
	}
	return SingleDouble(result), nil
}

func evalUnaryExpr(ec *evalContext, e UnaryExpr) (Sequence, error) {
	r, err := eval(ec, e.Operand)
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	a, err := AtomizeItem(r[0])
	if err != nil {
		return nil, err
	}
	n := promoteToDouble(a)
	if a.TypeName == TypeInteger {
		return SingleInteger(-int64(n)), nil
	}
	return SingleDouble(-n), nil
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
	ls := atomizeToString(left)
	rs := atomizeToString(right)
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
	start := int64(promoteToDouble(sa))
	end := int64(promoteToDouble(ea))
	if start > end {
		return nil, nil
	}
	count := end - start + 1
	if count > int64(ec.maxNodes) {
		return nil, ErrNodeSetLimit
	}
	result := make(Sequence, 0, count)
	for i := start; i <= end; i++ {
		result = append(result, AtomicValue{TypeName: TypeInteger, Value: i})
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
	nodes, ok := NodesFrom(base)
	if !ok {
		return nil, ErrFilterNotNodeSet
	}
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

// --- Lookup ---

func evalLookupExpr(ec *evalContext, e LookupExpr) (Sequence, error) {
	base, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	var result Sequence
	for _, item := range base {
		r, err := lookupItem(ec, item, e.Key, e.All)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

func evalUnaryLookupExpr(ec *evalContext, e UnaryLookupExpr) (Sequence, error) {
	if ec.node == nil {
		return nil, &XPathError{Code: "XPDY0002", Message: "context item is absent"}
	}
	return lookupItem(ec, NodeItem{Node: ec.node}, e.Key, e.All)
}

func lookupItem(ec *evalContext, item Item, keyExpr Expr, all bool) (Sequence, error) {
	switch v := item.(type) {
	case MapItem:
		if all {
			var result Sequence
			_ = v.ForEach(func(_ AtomicValue, val Sequence) error {
				result = append(result, val...)
				return nil
			})
			return result, nil
		}
		keySeq, err := eval(ec, keyExpr)
		if err != nil {
			return nil, err
		}
		if len(keySeq) == 0 {
			return nil, nil
		}
		ka, err := AtomizeItem(keySeq[0])
		if err != nil {
			return nil, err
		}
		val, ok := v.Get(ka)
		if !ok {
			return nil, nil
		}
		return val, nil
	case ArrayItem:
		if all {
			var result Sequence
			for _, m := range v.Members() {
				result = append(result, m...)
			}
			return result, nil
		}
		keySeq, err := eval(ec, keyExpr)
		if err != nil {
			return nil, err
		}
		if len(keySeq) == 0 {
			return nil, nil
		}
		ka, err := AtomizeItem(keySeq[0])
		if err != nil {
			return nil, err
		}
		idx := int(promoteToDouble(ka))
		return v.Get(idx)
	default:
		return nil, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("lookup requires map or array, got %T", item)}
	}
}

// --- 3.4: FLWOR + control flow ---

func evalFLWOR(ec *evalContext, e FLWORExpr) (Sequence, error) {
	// Build tuples by processing clauses
	tuples := []flworTuple{{vars: copyVars(ec.vars)}}

	for _, clause := range e.Clauses {
		switch c := clause.(type) {
		case ForClause:
			var newTuples []flworTuple
			for _, tup := range tuples {
				subCtx := &evalContext{
					goCtx: ec.goCtx, node: ec.node, position: ec.position,
					size: ec.size, vars: tup.vars, namespaces: ec.namespaces,
					functions: ec.functions, fnsNS: ec.fnsNS, depth: ec.depth,
					opCount: ec.opCount, opLimit: ec.opLimit, docOrder: ec.docOrder,
					maxNodes: ec.maxNodes,
				}
				domain, err := eval(subCtx, c.Expr)
				if err != nil {
					return nil, err
				}
				for _, item := range domain {
					newVars := copyVars(tup.vars)
					newVars[c.Var] = Sequence{item}
					newTuples = append(newTuples, flworTuple{vars: newVars})
				}
			}
			tuples = newTuples
		case LetClause:
			for i := range tuples {
				subCtx := &evalContext{
					goCtx: ec.goCtx, node: ec.node, position: ec.position,
					size: ec.size, vars: tuples[i].vars, namespaces: ec.namespaces,
					functions: ec.functions, fnsNS: ec.fnsNS, depth: ec.depth,
					opCount: ec.opCount, opLimit: ec.opLimit, docOrder: ec.docOrder,
					maxNodes: ec.maxNodes,
				}
				val, err := eval(subCtx, c.Expr)
				if err != nil {
					return nil, err
				}
				tuples[i].vars = copyVars(tuples[i].vars)
				tuples[i].vars[c.Var] = val
			}
		case WhereClause:
			var filtered []flworTuple
			for _, tup := range tuples {
				subCtx := &evalContext{
					goCtx: ec.goCtx, node: ec.node, position: ec.position,
					size: ec.size, vars: tup.vars, namespaces: ec.namespaces,
					functions: ec.functions, fnsNS: ec.fnsNS, depth: ec.depth,
					opCount: ec.opCount, opLimit: ec.opLimit, docOrder: ec.docOrder,
					maxNodes: ec.maxNodes,
				}
				r, err := eval(subCtx, c.Predicate)
				if err != nil {
					return nil, err
				}
				b, err := EBV(r)
				if err != nil {
					return nil, err
				}
				if b {
					filtered = append(filtered, tup)
				}
			}
			tuples = filtered
		case OrderByClause:
			sorted, sortErr := sortTuples(ec, tuples, c)
			if sortErr != nil {
				return nil, sortErr
			}
			tuples = sorted
		}
	}

	// Evaluate return expression for each tuple
	var result Sequence
	for _, tup := range tuples {
		retCtx := &evalContext{
			goCtx: ec.goCtx, node: ec.node, position: ec.position,
			size: ec.size, vars: tup.vars, namespaces: ec.namespaces,
			functions: ec.functions, fnsNS: ec.fnsNS, depth: ec.depth,
			opCount: ec.opCount, opLimit: ec.opLimit, docOrder: ec.docOrder,
			maxNodes: ec.maxNodes,
		}
		r, err := eval(retCtx, e.Return)
		if err != nil {
			return nil, err
		}
		result = append(result, r...)
	}
	return result, nil
}

type flworTuple struct {
	vars map[string]Sequence
}

func sortTuples(ec *evalContext, tuples []flworTuple, ob OrderByClause) ([]flworTuple, error) {
	type sortKey struct {
		values []AtomicValue
	}
	keys := make([]sortKey, len(tuples))
	for i, tup := range tuples {
		var vals []AtomicValue
		for _, spec := range ob.Specs {
			subCtx := &evalContext{
				goCtx: ec.goCtx, node: ec.node, position: ec.position,
				size: ec.size, vars: tup.vars, namespaces: ec.namespaces,
				functions: ec.functions, fnsNS: ec.fnsNS, depth: ec.depth,
				opCount: ec.opCount, opLimit: ec.opLimit, docOrder: ec.docOrder,
				maxNodes: ec.maxNodes,
			}
			r, err := eval(subCtx, spec.Expr)
			if err != nil {
				return nil, err
			}
			if len(r) > 0 {
				a, err := AtomizeItem(r[0])
				if err != nil {
					return nil, err
				}
				vals = append(vals, a)
			} else {
				vals = append(vals, AtomicValue{})
			}
		}
		keys[i] = sortKey{values: vals}
	}
	sort.SliceStable(tuples, func(i, j int) bool {
		for k, spec := range ob.Specs {
			cmp := compareAtomicOrder(keys[i].values[k], keys[j].values[k])
			if cmp == 0 {
				continue
			}
			if spec.Descending {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
	return tuples, nil
}

func compareAtomicOrder(a, b AtomicValue) int {
	af := promoteToDouble(a)
	bf := promoteToDouble(b)
	if af < bf {
		return -1
	}
	if af > bf {
		return 1
	}
	return 0
}

func copyVars(m map[string]Sequence) map[string]Sequence {
	cp := make(map[string]Sequence, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func evalQuantifiedExpr(ec *evalContext, e QuantifiedExpr) (Sequence, error) {
	domain, err := eval(ec, e.Domain)
	if err != nil {
		return nil, err
	}
	for _, item := range domain {
		subCtx := ec.withVar(e.Var, Sequence{item})
		r, err := eval(subCtx, e.Satisfies)
		if err != nil {
			return nil, err
		}
		b, err := EBV(r)
		if err != nil {
			return nil, err
		}
		if e.Some && b {
			return SingleBoolean(true), nil
		}
		if !e.Some && !b {
			return SingleBoolean(false), nil
		}
	}
	if e.Some {
		return SingleBoolean(false), nil
	}
	return SingleBoolean(true), nil
}

func evalIfExpr(ec *evalContext, e IfExpr) (Sequence, error) {
	cond, err := eval(ec, e.Cond)
	if err != nil {
		return nil, err
	}
	b, err := EBV(cond)
	if err != nil {
		return nil, err
	}
	if b {
		return eval(ec, e.Then)
	}
	return eval(ec, e.Else)
}

func evalTryCatchExpr(ec *evalContext, e TryCatchExpr) (Sequence, error) {
	result, err := eval(ec, e.Try)
	if err == nil {
		return result, nil
	}
	xpErr, ok := err.(*XPathError)
	if !ok {
		return nil, err // non-XPath errors propagate through
	}
	for _, catch := range e.Catches {
		if len(catch.Codes) == 0 {
			// Wildcard catch
			catchCtx := ec.withVar("err:code", SingleString(xpErr.Code))
			catchCtx = catchCtx.withVar("err:description", SingleString(xpErr.Message))
			return eval(catchCtx, catch.Expr)
		}
		for _, code := range catch.Codes {
			if code == xpErr.Code {
				catchCtx := ec.withVar("err:code", SingleString(xpErr.Code))
				catchCtx = catchCtx.withVar("err:description", SingleString(xpErr.Message))
				return eval(catchCtx, catch.Expr)
			}
		}
	}
	return nil, err // no matching catch
}

// --- 3.5: Type expressions ---

func evalInstanceOfExpr(ec *evalContext, e InstanceOfExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	return SingleBoolean(matchesSequenceType(seq, e.Type, ec)), nil
}

func evalCastExpr(ec *evalContext, e CastExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if len(seq) == 0 {
		if e.AllowEmpty {
			return nil, nil
		}
		return nil, &XPathError{Code: "XPTY0004", Message: "cast requires non-empty sequence"}
	}
	if len(seq) > 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "cast requires singleton"}
	}
	av, err := AtomizeItem(seq[0])
	if err != nil {
		return nil, err
	}
	targetType := resolveAtomicTypeName(e.Type, ec)
	result, err := CastAtomic(av, targetType)
	if err != nil {
		return nil, err
	}
	return SingleAtomic(result), nil
}

func evalCastableExpr(ec *evalContext, e CastableExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if len(seq) == 0 {
		return SingleBoolean(e.AllowEmpty), nil
	}
	if len(seq) > 1 {
		return SingleBoolean(false), nil
	}
	av, err := AtomizeItem(seq[0])
	if err != nil {
		return SingleBoolean(false), nil
	}
	targetType := resolveAtomicTypeName(e.Type, ec)
	_, castErr := CastAtomic(av, targetType)
	return SingleBoolean(castErr == nil), nil
}

func evalTreatAsExpr(ec *evalContext, e TreatAsExpr) (Sequence, error) {
	seq, err := eval(ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if !matchesSequenceType(seq, e.Type, ec) {
		return nil, &XPathError{Code: "XPDY0050", Message: "treat as type mismatch"}
	}
	return seq, nil
}

func resolveAtomicTypeName(tn AtomicTypeName, ec *evalContext) string {
	if tn.Prefix == "" {
		return "xs:" + tn.Name
	}
	if tn.Prefix == "xs" || tn.Prefix == "xsd" {
		return "xs:" + tn.Name
	}
	// Resolve via namespace context
	if ec.namespaces != nil {
		if uri, ok := ec.namespaces[tn.Prefix]; ok {
			if uri == "http://www.w3.org/2001/XMLSchema" {
				return "xs:" + tn.Name
			}
		}
	}
	return tn.Prefix + ":" + tn.Name
}

func matchesSequenceType(seq Sequence, st SequenceType, ec *evalContext) bool {
	if st.Void {
		return len(seq) == 0
	}
	switch st.Occurrence {
	case OccurrenceExactlyOne:
		if len(seq) != 1 {
			return false
		}
	case OccurrenceZeroOrOne:
		if len(seq) > 1 {
			return false
		}
	case OccurrenceOneOrMore:
		if len(seq) == 0 {
			return false
		}
	case OccurrenceZeroOrMore:
		// any length ok
	}
	for _, item := range seq {
		if !matchesItemType(item, st.ItemTest, ec) {
			return false
		}
	}
	return true
}

func matchesItemType(item Item, test NodeTest, ec *evalContext) bool {
	if test == nil {
		return true
	}
	switch t := test.(type) {
	case AnyItemTest:
		return true
	case TypeTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchTypeTest(t, ni.Node)
	case NameTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNameTest(t, ni.Node, AxisChild, ec)
	case ElementTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNodeTest(t, ni.Node, AxisChild, ec)
	case AttributeTest:
		ni, ok := item.(NodeItem)
		if !ok {
			return false
		}
		return matchNodeTest(t, ni.Node, AxisAttribute, ec)
	case AtomicOrUnionType:
		av, ok := item.(AtomicValue)
		if !ok {
			return false
		}
		targetType := "xs:" + t.Name
		if t.Prefix != "" {
			targetType = t.Prefix + ":" + t.Name
		}
		if targetType == TypeAnyAtomicType {
			return true
		}
		return av.TypeName == targetType
	case FunctionTest:
		_, ok := item.(FunctionItem)
		return ok
	case MapTest:
		_, ok := item.(MapItem)
		return ok
	case ArrayTest:
		_, ok := item.(ArrayItem)
		return ok
	}
	return false
}

// --- 3.6: Function infrastructure ---

func evalFunctionCall(ec *evalContext, e FunctionCall) (Sequence, error) {
	// Evaluate arguments
	args := make([]Sequence, len(e.Args))
	hasPlaceholders := false
	for i, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			hasPlaceholders = true
			continue
		}
		a, err := eval(ec, argExpr)
		if err != nil {
			return nil, err
		}
		args[i] = a
	}

	// Partial application: if any args are placeholders, return FunctionItem
	if hasPlaceholders {
		return partialApply(ec, e, args)
	}

	// Resolve function
	fn, err := resolveFunction(ec, e.Prefix, e.Name, len(args))
	if err != nil {
		return nil, err
	}

	fctx := withFnContext(ec.goCtx, ec)
	return fn.Call(fctx, args)
}

func evalDynamicFunctionCall(ec *evalContext, e DynamicFunctionCall) (Sequence, error) {
	funcSeq, err := eval(ec, e.Func)
	if err != nil {
		return nil, err
	}
	if len(funcSeq) != 1 {
		return nil, &XPathError{Code: "XPTY0004", Message: "dynamic function call requires single function item"}
	}
	fi, ok := funcSeq[0].(FunctionItem)
	if !ok {
		return nil, &XPathError{Code: "XPTY0004", Message: fmt.Sprintf("dynamic function call requires function item, got %T", funcSeq[0])}
	}
	args := make([]Sequence, len(e.Args))
	for i, argExpr := range e.Args {
		a, err := eval(ec, argExpr)
		if err != nil {
			return nil, err
		}
		args[i] = a
	}
	if fi.Arity >= 0 && len(args) != fi.Arity {
		return nil, fmt.Errorf("%w: expected %d arguments, got %d", ErrArityMismatch, fi.Arity, len(args))
	}
	return fi.Invoke(ec.goCtx, args)
}

func evalNamedFunctionRef(ec *evalContext, e NamedFunctionRef) (Sequence, error) {
	fn, err := resolveFunction(ec, e.Prefix, e.Name, e.Arity)
	if err != nil {
		return nil, err
	}
	fi := FunctionItem{
		Arity: e.Arity,
		Name:  e.Name,
		Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
			return fn.Call(ctx, args)
		},
	}
	return Sequence{fi}, nil
}

func evalInlineFunctionExpr(ec *evalContext, e InlineFunctionExpr) (Sequence, error) {
	// Capture current variable scope snapshot
	closedVars := copyVars(ec.vars)
	fi := FunctionItem{
		Arity: len(e.Params),
		Invoke: func(ctx context.Context, args []Sequence) (Sequence, error) {
			innerCtx := &evalContext{
				goCtx:      ctx,
				node:       ec.node,
				position:   ec.position,
				size:       ec.size,
				vars:       copyVars(closedVars),
				namespaces: ec.namespaces,
				functions:  ec.functions,
				fnsNS:      ec.fnsNS,
				opCount:    ec.opCount,
				opLimit:    ec.opLimit,
				docOrder:   ec.docOrder,
				maxNodes:   ec.maxNodes,
			}
			for i, param := range e.Params {
				if i < len(args) {
					innerCtx.vars[param.Name] = args[i]
				}
			}
			return eval(innerCtx, e.Body)
		},
	}
	return Sequence{fi}, nil
}

func partialApply(ec *evalContext, e FunctionCall, fixedArgs []Sequence) (Sequence, error) {
	// Count placeholders to determine new arity
	var placeholderIndices []int
	for i, argExpr := range e.Args {
		if _, ok := argExpr.(PlaceholderExpr); ok {
			placeholderIndices = append(placeholderIndices, i)
		}
	}

	fn, err := resolveFunction(ec, e.Prefix, e.Name, len(e.Args))
	if err != nil {
		return nil, err
	}

	fi := FunctionItem{
		Arity: len(placeholderIndices),
		Name:  e.Name,
		Invoke: func(ctx context.Context, partialArgs []Sequence) (Sequence, error) {
			fullArgs := make([]Sequence, len(e.Args))
			copy(fullArgs, fixedArgs)
			pi := 0
			for _, idx := range placeholderIndices {
				if pi < len(partialArgs) {
					fullArgs[idx] = partialArgs[pi]
					pi++
				}
			}
			return fn.Call(ctx, fullArgs)
		},
	}
	return Sequence{fi}, nil
}

// --- Constructor evaluation ---

func evalMapConstructorExpr(ec *evalContext, e MapConstructorExpr) (Sequence, error) {
	entries := make([]MapEntry, len(e.Pairs))
	for i, pair := range e.Pairs {
		keySeq, err := eval(ec, pair.Key)
		if err != nil {
			return nil, err
		}
		if len(keySeq) != 1 {
			return nil, &XPathError{Code: "XPTY0004", Message: "map key must be a single atomic value"}
		}
		ka, err := AtomizeItem(keySeq[0])
		if err != nil {
			return nil, err
		}
		valSeq, err := eval(ec, pair.Value)
		if err != nil {
			return nil, err
		}
		entries[i] = MapEntry{Key: ka, Value: valSeq}
	}
	return Sequence{NewMap(entries)}, nil
}

func evalArrayConstructorExpr(ec *evalContext, e ArrayConstructorExpr) (Sequence, error) {
	if e.SquareBracket {
		// [a, b, c] — each expr is one member
		members := make([]Sequence, len(e.Items))
		for i, item := range e.Items {
			seq, err := eval(ec, item)
			if err != nil {
				return nil, err
			}
			members[i] = seq
		}
		return Sequence{NewArray(members)}, nil
	}
	// array { expr } — evaluate as sequence, each item is singleton member
	if len(e.Items) == 0 {
		return Sequence{NewArray(nil)}, nil
	}
	seq, err := eval(ec, e.Items[0])
	if err != nil {
		return nil, err
	}
	members := make([]Sequence, len(seq))
	for i, item := range seq {
		members[i] = Sequence{item}
	}
	return Sequence{NewArray(members)}, nil
}

// --- Helpers ---

func promoteToDouble(a AtomicValue) float64 {
	switch a.TypeName {
	case TypeDouble, TypeFloat:
		return a.Value.(float64)
	case TypeInteger:
		return float64(a.Value.(int64))
	case TypeDecimal:
		var f float64
		if _, err := fmt.Sscanf(a.Value.(string), "%f", &f); err != nil {
			return math.NaN()
		}
		return f
	case TypeUntypedAtomic, TypeString:
		s := a.Value.(string)
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
			return math.NaN()
		}
		return f
	case TypeBoolean:
		if a.Value.(bool) {
			return 1
		}
		return 0
	}
	return math.NaN()
}

func atomizeToString(seq Sequence) string {
	if len(seq) == 0 {
		return ""
	}
	a, err := AtomizeItem(seq[0])
	if err != nil {
		return ""
	}
	s, _ := atomicToString(a)
	return s
}
