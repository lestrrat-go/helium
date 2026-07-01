package xpath1

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/lestrrat-go/helium/internal/xpath1/number"
)

const (
	maxRecursionDepth = ixpath.DefaultMaxRecursionDepth
	maxNodeSetLength  = ixpath.DefaultMaxNodeSetLength
)

// evalContext holds the evaluation state for an XPath expression.
type evalContext struct {
	node          helium.Node
	position      int
	size          int
	namespaces    map[string]string
	defaultElemNS string
	variables     map[string]any
	functions     map[string]Function
	functionsNS   map[QualifiedName]Function
	depth         int
	opCount       *int // shared across the entire evaluation tree
	opLimit       int  // 0 = unlimited
	docOrder      *ixpath.DocOrderCache
}

func newEvalContextWithConfig(node helium.Node, cfg *evalConfig) *evalContext {
	opCount := 0
	ectx := &evalContext{
		node:     node,
		position: 1,
		size:     1,
		opCount:  &opCount,
		docOrder: &ixpath.DocOrderCache{},
	}
	if cfg != nil {
		ectx.namespaces = cfg.namespaces
		ectx.defaultElemNS = cfg.defaultElemNS
		ectx.variables = cfg.variables
		ectx.opLimit = cfg.opLimit
		ectx.functions = cfg.functions
		ectx.functionsNS = cfg.functionsNS
	}
	return ectx
}

func (ec *evalContext) withNode(n helium.Node, pos, size int) *evalContext {
	return &evalContext{
		node:          n,
		position:      pos,
		size:          size,
		namespaces:    ec.namespaces,
		defaultElemNS: ec.defaultElemNS,
		variables:     ec.variables,
		functions:     ec.functions,
		functionsNS:   ec.functionsNS,
		depth:         ec.depth,
		opCount:       ec.opCount,
		opLimit:       ec.opLimit,
		docOrder:      ec.docOrder,
	}
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

func (ec *evalContext) Node() helium.Node {
	if ec == nil {
		return nil
	}
	return ec.node
}

func (ec *evalContext) Position() int {
	if ec == nil {
		return 0
	}
	return ec.position
}

func (ec *evalContext) Size() int {
	if ec == nil {
		return 0
	}
	return ec.size
}

func (ec *evalContext) Namespace(prefix string) (string, bool) {
	if ec == nil || ec.namespaces == nil {
		return "", false
	}
	uri, ok := ec.namespaces[prefix]
	return uri, ok
}

func (ec *evalContext) Variable(name string) (any, bool) {
	if ec == nil || ec.variables == nil {
		return nil, false
	}
	v, ok := ec.variables[name]
	return v, ok
}

// eval dispatches to the appropriate evaluator for each AST node type.
func eval(ctx context.Context, ec *evalContext, expr Expr) (*Result, error) {
	// Honor cancellation/deadline at every recursion point so a long or
	// looping evaluation aborts promptly with the context error.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ec.depth++
	if ec.depth > maxRecursionDepth {
		return nil, ErrRecursionLimit
	}
	defer func() { ec.depth-- }()
	return dispatchExpr(ctx, ec, expr)
}

// dispatchExpr routes an expression to its evaluator without the depth check.
func dispatchExpr(ctx context.Context, ec *evalContext, expr Expr) (*Result, error) {
	switch e := expr.(type) {
	case *LocationPath:
		return evalLocationPath(ctx, ec, e)
	case BinaryExpr:
		return evalBinaryExpr(ctx, ec, e)
	case UnaryExpr:
		return evalUnaryExpr(ctx, ec, e)
	case LiteralExpr:
		return &Result{Type: StringResult, String: e.Value}, nil
	case NumberExpr:
		return &Result{Type: NumberResult, Number: e.Value}, nil
	case VariableExpr:
		return evalVariableExpr(ec, e)
	case FunctionCall:
		return evalFunctionCall(ctx, ec, e)
	default:
		return dispatchCompoundExpr(ctx, ec, expr)
	}
}

// dispatchCompoundExpr handles compound expression types that combine sub-expressions.
func dispatchCompoundExpr(ctx context.Context, ec *evalContext, expr Expr) (*Result, error) {
	switch e := expr.(type) {
	case FilterExpr:
		return evalFilterExpr(ctx, ec, e)
	case UnionExpr:
		return evalUnionExpr(ctx, ec, e)
	case PathExpr:
		return evalPathExpr(ctx, ec, e)
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedExpr, expr)
	}
}

func evalLocationPath(ctx context.Context, ec *evalContext, lp *LocationPath) (*Result, error) {
	var nodes []helium.Node

	if lp.Absolute {
		// DocumentRoot dereferences the context node; guard against a
		// nil/typed-nil context so a missing context node yields an
		// evaluation error instead of a panic.
		if ixpath.IsNilNode(ec.node) {
			return nil, ErrNoContextNode
		}
		root := ixpath.DocumentRoot(ec.node)
		nodes = []helium.Node{root}
	} else {
		nodes = []helium.Node{ec.node}
	}

	var err error
	for _, step := range lp.Steps {
		if len(step.Predicates) > 0 {
			nodes, err = evalStepWithPredicates(ctx, ec, nodes, step)
		} else {
			nodes, err = evalStepNoPredicates(ctx, ec, nodes, step)
		}
		if err != nil {
			return nil, err
		}
	}

	return &Result{Type: NodeSetResult, NodeSet: nodes}, nil
}

// evalStepWithPredicates evaluates one location step that has predicates.
// Position() is relative to each parent's candidate set, not the global set.
func evalStepWithPredicates(ctx context.Context, ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var allFiltered []helium.Node
	for _, n := range nodes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		candidates, err := traverseAxis(ctx, step.Axis, n)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(len(candidates)); err != nil {
			return nil, err
		}
		matched := filterByNodeTest(candidates, step.NodeTest, step.Axis, ec)
		for _, pred := range step.Predicates {
			matched, err = applyPredicate(ctx, ec, matched, pred)
			if err != nil {
				return nil, err
			}
		}
		allFiltered = append(allFiltered, matched...)
	}
	return ixpath.DeduplicateNodes(allFiltered, ec.docOrder, maxNodeSetLength)
}

// evalStepNoPredicates evaluates one location step that has no predicates.
func evalStepNoPredicates(ctx context.Context, ec *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var next []helium.Node
	for _, n := range nodes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		candidates, err := traverseAxis(ctx, step.Axis, n)
		if err != nil {
			return nil, err
		}
		if err := ec.countOps(len(candidates)); err != nil {
			return nil, err
		}
		next = append(next, filterByNodeTest(candidates, step.NodeTest, step.Axis, ec)...)
	}
	return ixpath.DeduplicateNodes(next, ec.docOrder, maxNodeSetLength)
}

// filterByNodeTest returns only those nodes that match the given node test.
func filterByNodeTest(candidates []helium.Node, nt NodeTest, axis AxisType, ec *evalContext) []helium.Node {
	matched := make([]helium.Node, 0, len(candidates))
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
		if pi, ok := n.(*helium.ProcessingInstruction); ok {
			return pi.Name() == test.Target
		}
		return false
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
		return matchNameTestNamespaceAxis(test, n)
	default:
		if n.Type() != helium.ElementNode {
			return false
		}
	}

	return matchNameTestByLocalAndPrefix(test, n, ec)
}

// matchNameTestNamespaceAxis matches a name test against a namespace-axis node.
func matchNameTestNamespaceAxis(test NameTest, n helium.Node) bool {
	if n.Type() != helium.NamespaceNode {
		return false
	}
	if test.Local == "*" {
		return true
	}
	return n.Name() == test.Local
}

// matchNameTestByLocalAndPrefix matches a name test's local name and optional prefix
// against a node (used after the principal node type check has passed).
func matchNameTestByLocalAndPrefix(test NameTest, n helium.Node, ec *evalContext) bool {
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

	// XPath 1.0 has no default element namespace: an unprefixed name test
	// matches only nodes with no namespace URI. When a default element namespace
	// is configured (XSD 1.1 @xpathDefaultNamespace), an unprefixed ELEMENT name
	// test matches that URI instead; attributes are never affected (they have no
	// default namespace).
	want := ""
	if ec != nil && ec.defaultElemNS != "" && n.Type() == helium.ElementNode {
		want = ec.defaultElemNS
	}
	return ixpath.NodeNamespaceURI(n) == want
}

func matchPrefix(prefix string, n helium.Node, ec *evalContext) bool {
	if ec.namespaces != nil {
		uri, ok := ec.namespaces[prefix]
		if ok {
			return ixpath.NodeNamespaceURI(n) == uri
		}
	}
	// The xml prefix is always bound per the XML Namespaces spec.
	if prefix == lexicon.PrefixXML {
		return ixpath.NodeNamespaceURI(n) == lexicon.NamespaceXML
	}
	// Per XPath 1.0, prefix resolution comes from the evaluation namespace
	// context, not the document's lexical prefixes. An unbound prefix cannot
	// match a node merely because it happens to share the lexical prefix.
	return false
}

func matchTypeTest(test TypeTest, n helium.Node) bool {
	switch test.Type {
	case NodeTestNode:
		return true
	case NodeTestText:
		return n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode
	case NodeTestComment:
		return n.Type() == helium.CommentNode
	case NodeTestProcessingInstruction:
		return n.Type() == helium.ProcessingInstructionNode
	}
	return false
}

func applyPredicate(ctx context.Context, ec *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
	if err := ec.countOps(len(nodes)); err != nil {
		return nil, err
	}
	size := len(nodes)
	var result []helium.Node
	for i, n := range nodes {
		pctx := ec.withNode(n, i+1, size)
		r, err := eval(ctx, pctx, pred)
		if err != nil {
			return nil, err
		}
		if predicateTrue(r, i+1) {
			result = append(result, n)
		}
	}
	return result, nil
}

// predicateTrue evaluates a predicate result. Per XPath spec:
// - If result is a number, it's true when equal to position.
// - Otherwise, convert to boolean.
func predicateTrue(r *Result, position int) bool {
	if r.Type == NumberResult {
		return r.Number == float64(position)
	}
	return resultToBoolean(r)
}

func evalBinaryExpr(ctx context.Context, ec *evalContext, e BinaryExpr) (*Result, error) {
	switch e.Op {
	case TokenOr:
		return evalOr(ctx, ec, e)
	case TokenAnd:
		return evalAnd(ctx, ec, e)
	case TokenEquals, TokenNotEquals, TokenLess, TokenLessEq, TokenGreater, TokenGreaterEq:
		return evalComparison(ctx, ec, e)
	case TokenPlus, TokenMinus, TokenStar, TokenDiv, TokenMod:
		return evalArithmetic(ctx, ec, e)
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedBinaryOp, e.Op)
}

func evalOr(ctx context.Context, ec *evalContext, e BinaryExpr) (*Result, error) {
	return evalShortCircuit(ctx, ec, e, true)
}

func evalAnd(ctx context.Context, ec *evalContext, e BinaryExpr) (*Result, error) {
	return evalShortCircuit(ctx, ec, e, false)
}

// evalShortCircuit evaluates a boolean "or" (shortOn==true) or "and"
// (shortOn==false) expression. The left operand is always evaluated; when its
// boolean value equals shortOn, that value is returned without evaluating the
// right operand. Otherwise the right operand (and its error) is evaluated and
// its boolean value returned.
func evalShortCircuit(ctx context.Context, ec *evalContext, e BinaryExpr, shortOn bool) (*Result, error) {
	left, err := eval(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	if resultToBoolean(left) == shortOn {
		return &Result{Type: BooleanResult, Bool: shortOn}, nil
	}
	right, err := eval(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Bool: resultToBoolean(right)}, nil
}

func evalComparison(ctx context.Context, ec *evalContext, e BinaryExpr) (*Result, error) {
	left, err := eval(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	b, err := compareResults(ctx, ec, e.Op, left, right)
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Bool: b}, nil
}

// compareResults implements XPath comparison semantics including node-set
// comparisons. Node-set comparisons iterate over (possibly large) node-sets, so
// they charge the operation counter and honor context cancellation just like
// the axis-iteration loops; a cancelled context or an exceeded op limit aborts
// promptly with the same error the rest of the evaluator returns.
func compareResults(ctx context.Context, ec *evalContext, op TokenType, left, right *Result) (bool, error) {
	if left.Type == NodeSetResult {
		return compareNodeSet(ctx, ec, op, left.NodeSet, right)
	}
	if right.Type == NodeSetResult {
		return compareNodeSetRight(ctx, ec, op, left, right.NodeSet)
	}
	return compareScalars(op, left, right), nil
}

// compareNodeSet handles comparisons where the left operand is a node-set.
func compareNodeSet(ctx context.Context, ec *evalContext, op TokenType, leftNodes []helium.Node, right *Result) (bool, error) {
	if right.Type == NodeSetResult {
		// A node-set vs node-set general comparison is always false when
		// either side is empty: there is no pair of nodes to compare. Return
		// before doing any per-node string-value work (which would otherwise
		// be O(n) or O(m) with a zero op charge, escaping the op-limit and
		// cancellation bound).
		if len(leftNodes) == 0 || len(right.NodeSet) == 0 {
			return false, nil
		}
		// Honor a context cancelled before the comparison begins so no
		// string-value work runs after cancellation, even when the first pair
		// would otherwise match and return before the periodic re-check below.
		if err := ctx.Err(); err != nil {
			return false, err
		}
		// Charge an op (and periodically honor cancellation) BEFORE materializing
		// EITHER side's string value, so the bound is complete: if the operand
		// evaluation already spent the budget, the first per-pair charge fails
		// before any string-value walk or large allocation runs.
		//
		// The right-hand string values are cached LAZILY in a sparse map, grown
		// only for the indices actually reached. A dense len(right)-sized cache
		// would do O(m) allocation up front — escaping the op-limit/cancellation
		// bound for a huge right-hand node-set (e.g. one returned by a custom
		// function) even when the budget is already exhausted.
		rightVals := make(map[int]string)
		// Hoist the cancellation channel once so the per-pair check below is a
		// cheap non-blocking channel receive (cancelCtx.Err() takes a mutex; a
		// receive on this hoisted channel does not). A nil channel is fine: the
		// select's default branch always wins, matching the rest of the evaluator's
		// nil-context tolerance.
		done := ctx.Done()
		for _, ln := range leftNodes {
			// lv is materialized lazily, only AFTER the first successful op
			// charge for this left node, so a pre-exhausted budget aborts before
			// walking/copying a potentially large left subtree.
			var lv string
			lvComputed := false
			for i, rn := range right.NodeSet {
				if err := ec.countOps(1); err != nil {
					return false, err
				}
				// Honor cancellation on EVERY pair, before either side's
				// string-value walk runs, so a context cancelled mid-loop aborts
				// promptly even for comparisons far smaller than any periodic
				// re-check window.
				select {
				case <-done:
					return false, ctx.Err()
				default:
				}
				if !lvComputed {
					lv = ixpath.StringValue(ln)
					lvComputed = true
				}
				rv, ok := rightVals[i]
				if !ok {
					rv = ixpath.StringValue(rn)
					rightVals[i] = rv
				}
				if compareStrings(op, lv, rv) {
					return true, nil
				}
			}
		}
		return false, nil
	}
	// Per XPath 1.0 REC 3.4, comparing a node-set with a boolean converts the
	// whole node-set to a boolean (true iff non-empty), then compares booleans.
	if right.Type == BooleanResult {
		nsBool := len(leftNodes) > 0
		if op == TokenEquals {
			return nsBool == right.Bool, nil
		}
		if op == TokenNotEquals {
			return nsBool != right.Bool, nil
		}
		return compareNumbers(op, boolToNumber(nsBool), boolToNumber(right.Bool)), nil
	}
	for _, ln := range leftNodes {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if err := ec.countOps(1); err != nil {
			return false, err
		}
		if compareWithScalar(op, ixpath.StringValue(ln), right) {
			return true, nil
		}
	}
	return false, nil
}

// compareNodeSetRight handles comparisons where only the right operand is a node-set.
func compareNodeSetRight(ctx context.Context, ec *evalContext, op TokenType, left *Result, rightNodes []helium.Node) (bool, error) {
	// Per XPath 1.0 REC 3.4, comparing a boolean with a node-set converts the
	// whole node-set to a boolean (true iff non-empty), then compares booleans.
	if left.Type == BooleanResult {
		nsBool := len(rightNodes) > 0
		if op == TokenEquals {
			return left.Bool == nsBool, nil
		}
		if op == TokenNotEquals {
			return left.Bool != nsBool, nil
		}
		return compareNumbers(op, boolToNumber(left.Bool), boolToNumber(nsBool)), nil
	}
	rev := reverseOp(op)
	for _, rn := range rightNodes {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if err := ec.countOps(1); err != nil {
			return false, err
		}
		if compareWithScalar(rev, ixpath.StringValue(rn), left) {
			return true, nil
		}
	}
	return false, nil
}

// boolToNumber converts a boolean to its XPath number value (true=1, false=0).
func boolToNumber(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// compareScalars compares two non-node-set results using XPath type coercion rules.
func compareScalars(op TokenType, left, right *Result) bool {
	if op == TokenEquals || op == TokenNotEquals {
		return compareScalarsEqNe(op, left, right)
	}
	return compareNumbers(op, resultToNumber(left), resultToNumber(right))
}

// compareScalarsEqNe compares two scalar results for equality or inequality
// according to XPath 1.0 type coercion rules.
func compareScalarsEqNe(op TokenType, left, right *Result) bool {
	if left.Type == BooleanResult || right.Type == BooleanResult {
		lb := resultToBoolean(left)
		rb := resultToBoolean(right)
		if op == TokenEquals {
			return lb == rb
		}
		return lb != rb
	}
	if left.Type == NumberResult || right.Type == NumberResult {
		ln := resultToNumber(left)
		rn := resultToNumber(right)
		if op == TokenEquals {
			return ln == rn
		}
		return ln != rn
	}
	ls := resultToString(left)
	rs := resultToString(right)
	if op == TokenEquals {
		return ls == rs
	}
	return ls != rs
}

func compareWithScalar(op TokenType, nodeStr string, scalar *Result) bool {
	switch scalar.Type {
	case BooleanResult:
		nb := len(nodeStr) > 0
		if op == TokenEquals {
			return nb == scalar.Bool
		}
		if op == TokenNotEquals {
			return nb != scalar.Bool
		}
		var ln, rn float64
		if nb {
			ln = 1
		}
		if scalar.Bool {
			rn = 1
		}
		return compareNumbers(op, ln, rn)
	case NumberResult:
		nn := stringToNumber(nodeStr)
		return compareNumbers(op, nn, scalar.Number)
	default:
		if op == TokenEquals || op == TokenNotEquals {
			return compareStrings(op, nodeStr, scalar.String)
		}
		nn := stringToNumber(nodeStr)
		sn := stringToNumber(scalar.String)
		return compareNumbers(op, nn, sn)
	}
}

func compareStrings(op TokenType, a, b string) bool {
	switch op {
	case TokenEquals:
		return a == b
	case TokenNotEquals:
		return a != b
	default:
		return compareNumbers(op, stringToNumber(a), stringToNumber(b))
	}
}

func compareNumbers(op TokenType, a, b float64) bool {
	switch op {
	case TokenEquals:
		return a == b
	case TokenNotEquals:
		return a != b
	case TokenLess:
		return a < b
	case TokenLessEq:
		return a <= b
	case TokenGreater:
		return a > b
	case TokenGreaterEq:
		return a >= b
	}
	return false
}

func reverseOp(op TokenType) TokenType {
	switch op {
	case TokenLess:
		return TokenGreater
	case TokenLessEq:
		return TokenGreaterEq
	case TokenGreater:
		return TokenLess
	case TokenGreaterEq:
		return TokenLessEq
	}
	return op
}

func evalArithmetic(ctx context.Context, ec *evalContext, e BinaryExpr) (*Result, error) {
	left, err := eval(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	ln := resultToNumber(left)
	rn := resultToNumber(right)

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
	case TokenMod:
		result = math.Mod(ln, rn)
	}
	return &Result{Type: NumberResult, Number: result}, nil
}

func evalUnaryExpr(ctx context.Context, ec *evalContext, e UnaryExpr) (*Result, error) {
	r, err := eval(ctx, ec, e.Operand)
	if err != nil {
		return nil, err
	}
	n := resultToNumber(r)
	return &Result{Type: NumberResult, Number: -n}, nil
}

func evalVariableExpr(ec *evalContext, e VariableExpr) (*Result, error) {
	if ec.variables == nil {
		return nil, fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
	}
	v, ok := ec.variables[e.Name]
	if !ok {
		return nil, fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
	}
	switch val := v.(type) {
	case []helium.Node:
		// Caller-provided slices may contain nil entries; drop them so
		// DeduplicateNodes (which dereferences each node) cannot panic.
		// IsNilNode also catches typed-nil concrete pointers, which are
		// non-nil at the interface level but still panic on dereference.
		clean := make([]helium.Node, 0, len(val))
		for _, n := range val {
			if !ixpath.IsNilNode(n) {
				clean = append(clean, n)
			}
		}
		nodes, err := ixpath.DeduplicateNodes(clean, ec.docOrder, maxNodeSetLength)
		if err != nil {
			return nil, err
		}
		return &Result{Type: NodeSetResult, NodeSet: nodes}, nil
	case string:
		return &Result{Type: StringResult, String: val}, nil
	case float64:
		return &Result{Type: NumberResult, Number: val}, nil
	case bool:
		return &Result{Type: BooleanResult, Bool: val}, nil
	default:
		return nil, fmt.Errorf("%w: $%s is %T", ErrUnsupportedVariableType, e.Name, v)
	}
}

func evalFilterExpr(ctx context.Context, ec *evalContext, e FilterExpr) (*Result, error) {
	r, err := eval(ctx, ec, e.Expr)
	if err != nil {
		return nil, err
	}
	if r.Type != NodeSetResult {
		return nil, ErrFilterNotNodeSet
	}
	nodes := r.NodeSet
	for _, pred := range e.Predicates {
		nodes, err = applyPredicate(ctx, ec, nodes, pred)
		if err != nil {
			return nil, err
		}
	}
	return &Result{Type: NodeSetResult, NodeSet: nodes}, nil
}

func evalUnionExpr(ctx context.Context, ec *evalContext, e UnionExpr) (*Result, error) {
	left, err := eval(ctx, ec, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ctx, ec, e.Right)
	if err != nil {
		return nil, err
	}
	if left.Type != NodeSetResult || right.Type != NodeSetResult {
		return nil, ErrUnionNotNodeSet
	}
	merged, err := ixpath.MergeNodeSets(left.NodeSet, right.NodeSet, ec.docOrder, maxNodeSetLength)
	if err != nil {
		return nil, err
	}
	return &Result{Type: NodeSetResult, NodeSet: merged}, nil
}

func evalPathExpr(ctx context.Context, ec *evalContext, e PathExpr) (*Result, error) {
	r, err := eval(ctx, ec, e.Filter)
	if err != nil {
		return nil, err
	}
	if e.Path == nil {
		return r, nil
	}
	if r.Type != NodeSetResult {
		return nil, ErrPathNotNodeSet
	}

	var result []helium.Node
	for _, n := range r.NodeSet {
		subCtx := ec.withNode(n, 1, 1)
		subResult, err := evalLocationPath(ctx, subCtx, e.Path)
		if err != nil {
			return nil, err
		}
		result, err = ixpath.MergeNodeSets(result, subResult.NodeSet, ec.docOrder, maxNodeSetLength)
		if err != nil {
			return nil, err
		}
	}
	return &Result{Type: NodeSetResult, NodeSet: result}, nil
}

// documentRoot returns the owning Document or the topmost ancestor.
func documentRoot(n helium.Node) helium.Node {
	return ixpath.DocumentRoot(n)
}

// stringValue returns the string-value of a node per XPath spec.
func stringValue(n helium.Node) string {
	return ixpath.StringValue(n)
}

// localNameOf returns the local name of any node type.
func localNameOf(n helium.Node) string {
	return ixpath.LocalNameOf(n)
}

// nodeNamespaceURI returns the namespace URI of any node type.
func nodeNamespaceURI(n helium.Node) string {
	return ixpath.NodeNamespaceURI(n)
}

// resultToString converts any Result to a string per XPath spec.
func resultToString(r *Result) string {
	switch r.Type {
	case StringResult:
		return r.String
	case BooleanResult:
		if r.Bool {
			return "true"
		}
		return "false"
	case NumberResult:
		return number.ToString(r.Number)
	case NodeSetResult:
		if len(r.NodeSet) == 0 {
			return ""
		}
		return ixpath.StringValue(r.NodeSet[0])
	}
	return ""
}

// resultToNumber converts any Result to a number per XPath spec.
func resultToNumber(r *Result) float64 {
	switch r.Type {
	case NumberResult:
		return r.Number
	case StringResult:
		return stringToNumber(r.String)
	case BooleanResult:
		if r.Bool {
			return 1
		}
		return 0
	case NodeSetResult:
		return stringToNumber(resultToString(r))
	}
	return math.NaN()
}

// resultToBoolean converts any Result to a boolean per XPath spec.
func resultToBoolean(r *Result) bool {
	switch r.Type {
	case BooleanResult:
		return r.Bool
	case NumberResult:
		return r.Number != 0 && !math.IsNaN(r.Number)
	case StringResult:
		return len(r.String) > 0
	case NodeSetResult:
		return len(r.NodeSet) > 0
	}
	return false
}

func stringToNumber(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return math.NaN()
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}
