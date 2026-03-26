package xpath1

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

const (
	maxRecursionDepth = ixpath.DefaultMaxRecursionDepth
	maxNodeSetLength  = ixpath.DefaultMaxNodeSetLength
)

// evalContext holds the evaluation state for an XPath expression.
type evalContext struct {
	goCtx       context.Context
	node        helium.Node
	position    int
	size        int
	namespaces  map[string]string
	variables   map[string]any
	functions   map[string]Function
	functionsNS map[QualifiedName]Function
	depth       int
	opCount     *int // shared across the entire evaluation tree
	opLimit     int  // 0 = unlimited
	docOrder    *ixpath.DocOrderCache
}

func newEvalContext(ctx context.Context, node helium.Node) *evalContext {
	opCount := 0
	ectx := &evalContext{
		goCtx:    ctx,
		node:     node,
		position: 1,
		size:     1,
		opCount:  &opCount,
		docOrder: &ixpath.DocOrderCache{},
	}
	// Pull config from context.Context if present.
	if cfg := getEvalConfig(ctx); cfg != nil {
		ectx.namespaces = cfg.namespaces
		ectx.variables = cfg.variables
		ectx.opLimit = cfg.opLimit
		ectx.functions = cfg.functions
		ectx.functionsNS = cfg.functionsNS
	}
	return ectx
}

func (ctx *evalContext) withNode(n helium.Node, pos, size int) *evalContext {
	return &evalContext{
		goCtx:       ctx.goCtx,
		node:        n,
		position:    pos,
		size:        size,
		namespaces:  ctx.namespaces,
		variables:   ctx.variables,
		functions:   ctx.functions,
		functionsNS: ctx.functionsNS,
		depth:       ctx.depth,
		opCount:     ctx.opCount,
		opLimit:     ctx.opLimit,
		docOrder:    ctx.docOrder,
	}
}

func (ctx *evalContext) countOps(n int) error {
	if ctx.opLimit <= 0 {
		return nil
	}
	*ctx.opCount += n
	if *ctx.opCount > ctx.opLimit {
		return ErrOpLimit
	}
	return nil
}

func (ctx *evalContext) Node() helium.Node {
	if ctx == nil {
		return nil
	}
	return ctx.node
}

func (ctx *evalContext) Position() int {
	if ctx == nil {
		return 0
	}
	return ctx.position
}

func (ctx *evalContext) Size() int {
	if ctx == nil {
		return 0
	}
	return ctx.size
}

func (ctx *evalContext) Namespace(prefix string) (string, bool) {
	if ctx == nil || ctx.namespaces == nil {
		return "", false
	}
	uri, ok := ctx.namespaces[prefix]
	return uri, ok
}

func (ctx *evalContext) Variable(name string) (any, bool) {
	if ctx == nil || ctx.variables == nil {
		return nil, false
	}
	v, ok := ctx.variables[name]
	return v, ok
}

// eval dispatches to the appropriate evaluator for each AST node type.
func eval(ctx *evalContext, expr Expr) (*Result, error) {
	ctx.depth++
	if ctx.depth > maxRecursionDepth {
		return nil, ErrRecursionLimit
	}
	defer func() { ctx.depth-- }()
	return dispatchExpr(ctx, expr)
}

// dispatchExpr routes an expression to its evaluator without the depth check.
func dispatchExpr(ctx *evalContext, expr Expr) (*Result, error) {
	switch e := expr.(type) {
	case *LocationPath:
		return evalLocationPath(ctx, e)
	case BinaryExpr:
		return evalBinaryExpr(ctx, e)
	case UnaryExpr:
		return evalUnaryExpr(ctx, e)
	case LiteralExpr:
		return &Result{Type: StringResult, String: e.Value}, nil
	case NumberExpr:
		return &Result{Type: NumberResult, Number: e.Value}, nil
	case VariableExpr:
		return evalVariableExpr(ctx, e)
	case FunctionCall:
		return evalFunctionCall(ctx, e)
	default:
		return dispatchCompoundExpr(ctx, expr)
	}
}

// dispatchCompoundExpr handles compound expression types that combine sub-expressions.
func dispatchCompoundExpr(ctx *evalContext, expr Expr) (*Result, error) {
	switch e := expr.(type) {
	case FilterExpr:
		return evalFilterExpr(ctx, e)
	case UnionExpr:
		return evalUnionExpr(ctx, e)
	case PathExpr:
		return evalPathExpr(ctx, e)
	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedExpr, expr)
	}
}

func evalLocationPath(ctx *evalContext, lp *LocationPath) (*Result, error) {
	var nodes []helium.Node

	if lp.Absolute {
		root := ixpath.DocumentRoot(ctx.node)
		nodes = []helium.Node{root}
	} else {
		nodes = []helium.Node{ctx.node}
	}

	var err error
	for _, step := range lp.Steps {
		if len(step.Predicates) > 0 {
			nodes, err = evalStepWithPredicates(ctx, nodes, step)
		} else {
			nodes, err = evalStepNoPredicates(ctx, nodes, step)
		}
		if err != nil {
			return nil, err
		}
	}

	return &Result{Type: NodeSetResult, NodeSet: nodes}, nil
}

// evalStepWithPredicates evaluates one location step that has predicates.
// Position() is relative to each parent's candidate set, not the global set.
func evalStepWithPredicates(ctx *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var allFiltered []helium.Node
	for _, n := range nodes {
		candidates, err := traverseAxis(step.Axis, n)
		if err != nil {
			return nil, err
		}
		if err := ctx.countOps(len(candidates)); err != nil {
			return nil, err
		}
		matched := filterByNodeTest(candidates, step.NodeTest, step.Axis, ctx)
		for _, pred := range step.Predicates {
			matched, err = applyPredicate(ctx, matched, pred)
			if err != nil {
				return nil, err
			}
		}
		allFiltered = append(allFiltered, matched...)
	}
	return ixpath.DeduplicateNodes(allFiltered, ctx.docOrder, maxNodeSetLength)
}

// evalStepNoPredicates evaluates one location step that has no predicates.
func evalStepNoPredicates(ctx *evalContext, nodes []helium.Node, step Step) ([]helium.Node, error) {
	var next []helium.Node
	for _, n := range nodes {
		candidates, err := traverseAxis(step.Axis, n)
		if err != nil {
			return nil, err
		}
		if err := ctx.countOps(len(candidates)); err != nil {
			return nil, err
		}
		next = append(next, filterByNodeTest(candidates, step.NodeTest, step.Axis, ctx)...)
	}
	return ixpath.DeduplicateNodes(next, ctx.docOrder, maxNodeSetLength)
}

// filterByNodeTest returns only those nodes that match the given node test.
func filterByNodeTest(candidates []helium.Node, nt NodeTest, axis AxisType, ctx *evalContext) []helium.Node {
	matched := make([]helium.Node, 0, len(candidates))
	for _, c := range candidates {
		if matchNodeTest(nt, c, axis, ctx) {
			matched = append(matched, c)
		}
	}
	return matched
}

func matchNodeTest(nt NodeTest, n helium.Node, axis AxisType, ctx *evalContext) bool {
	switch test := nt.(type) {
	case NameTest:
		return matchNameTest(test, n, axis, ctx)
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

func matchNameTest(test NameTest, n helium.Node, axis AxisType, ctx *evalContext) bool {
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

	return matchNameTestByLocalAndPrefix(test, n, ctx)
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
func matchNameTestByLocalAndPrefix(test NameTest, n helium.Node, ctx *evalContext) bool {
	if test.Local == "*" {
		if test.Prefix == "" {
			return true
		}
		return matchPrefix(test.Prefix, n, ctx)
	}

	if ixpath.LocalNameOf(n) != test.Local {
		return false
	}

	if test.Prefix != "" {
		return matchPrefix(test.Prefix, n, ctx)
	}

	return true
}

func matchPrefix(prefix string, n helium.Node, ctx *evalContext) bool {
	if ctx.namespaces != nil {
		uri, ok := ctx.namespaces[prefix]
		if ok {
			return ixpath.NodeNamespaceURI(n) == uri
		}
	}
	return ixpath.NodePrefix(n) == prefix
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

func applyPredicate(ctx *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
	if err := ctx.countOps(len(nodes)); err != nil {
		return nil, err
	}
	size := len(nodes)
	var result []helium.Node
	for i, n := range nodes {
		pctx := ctx.withNode(n, i+1, size)
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

// predicateTrue evaluates a predicate result. Per XPath spec:
// - If result is a number, it's true when equal to position.
// - Otherwise, convert to boolean.
func predicateTrue(r *Result, position int) bool {
	if r.Type == NumberResult {
		return r.Number == float64(position)
	}
	return resultToBoolean(r)
}

func evalBinaryExpr(ctx *evalContext, e BinaryExpr) (*Result, error) {
	switch e.Op {
	case TokenOr:
		return evalOr(ctx, e)
	case TokenAnd:
		return evalAnd(ctx, e)
	case TokenEquals, TokenNotEquals, TokenLess, TokenLessEq, TokenGreater, TokenGreaterEq:
		return evalComparison(ctx, e)
	case TokenPlus, TokenMinus, TokenStar, TokenDiv, TokenMod:
		return evalArithmetic(ctx, e)
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedBinaryOp, e.Op)
}

func evalOr(ctx *evalContext, e BinaryExpr) (*Result, error) {
	left, err := eval(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	if resultToBoolean(left) {
		return &Result{Type: BooleanResult, Bool: true}, nil
	}
	right, err := eval(ctx, e.Right)
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Bool: resultToBoolean(right)}, nil
}

func evalAnd(ctx *evalContext, e BinaryExpr) (*Result, error) {
	left, err := eval(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	if !resultToBoolean(left) {
		return &Result{Type: BooleanResult, Bool: false}, nil
	}
	right, err := eval(ctx, e.Right)
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Bool: resultToBoolean(right)}, nil
}

func evalComparison(ctx *evalContext, e BinaryExpr) (*Result, error) {
	left, err := eval(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ctx, e.Right)
	if err != nil {
		return nil, err
	}
	b := compareResults(e.Op, left, right)
	return &Result{Type: BooleanResult, Bool: b}, nil
}

// compareResults implements XPath comparison semantics including node-set comparisons.
func compareResults(op TokenType, left, right *Result) bool {
	if left.Type == NodeSetResult {
		return compareNodeSet(op, left.NodeSet, right)
	}
	if right.Type == NodeSetResult {
		return compareNodeSetRight(op, left, right.NodeSet)
	}
	return compareScalars(op, left, right)
}

// compareNodeSet handles comparisons where the left operand is a node-set.
func compareNodeSet(op TokenType, leftNodes []helium.Node, right *Result) bool {
	if right.Type == NodeSetResult {
		// Pre-compute string values for the right-hand node-set to
		// avoid recomputing them for every left-hand node.
		rightVals := make([]string, len(right.NodeSet))
		for i, rn := range right.NodeSet {
			rightVals[i] = ixpath.StringValue(rn)
		}
		for _, ln := range leftNodes {
			lv := ixpath.StringValue(ln)
			for _, rv := range rightVals {
				if compareStrings(op, lv, rv) {
					return true
				}
			}
		}
		return false
	}
	for _, ln := range leftNodes {
		if compareWithScalar(op, ixpath.StringValue(ln), right) {
			return true
		}
	}
	return false
}

// compareNodeSetRight handles comparisons where only the right operand is a node-set.
func compareNodeSetRight(op TokenType, left *Result, rightNodes []helium.Node) bool {
	rev := reverseOp(op)
	for _, rn := range rightNodes {
		if compareWithScalar(rev, ixpath.StringValue(rn), left) {
			return true
		}
	}
	return false
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

func evalArithmetic(ctx *evalContext, e BinaryExpr) (*Result, error) {
	left, err := eval(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ctx, e.Right)
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

func evalUnaryExpr(ctx *evalContext, e UnaryExpr) (*Result, error) {
	r, err := eval(ctx, e.Operand)
	if err != nil {
		return nil, err
	}
	n := resultToNumber(r)
	return &Result{Type: NumberResult, Number: -n}, nil
}

func evalVariableExpr(ctx *evalContext, e VariableExpr) (*Result, error) {
	if ctx.variables == nil {
		return nil, fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
	}
	v, ok := ctx.variables[e.Name]
	if !ok {
		return nil, fmt.Errorf("%w: $%s", ErrUndefinedVariable, e.Name)
	}
	switch val := v.(type) {
	case []helium.Node:
		return &Result{Type: NodeSetResult, NodeSet: val}, nil
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

func evalFilterExpr(ctx *evalContext, e FilterExpr) (*Result, error) {
	r, err := eval(ctx, e.Expr)
	if err != nil {
		return nil, err
	}
	if r.Type != NodeSetResult {
		return nil, ErrFilterNotNodeSet
	}
	nodes := r.NodeSet
	for _, pred := range e.Predicates {
		nodes, err = applyPredicate(ctx, nodes, pred)
		if err != nil {
			return nil, err
		}
	}
	return &Result{Type: NodeSetResult, NodeSet: nodes}, nil
}

func evalUnionExpr(ctx *evalContext, e UnionExpr) (*Result, error) {
	left, err := eval(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	right, err := eval(ctx, e.Right)
	if err != nil {
		return nil, err
	}
	if left.Type != NodeSetResult || right.Type != NodeSetResult {
		return nil, ErrUnionNotNodeSet
	}
	merged, err := ixpath.MergeNodeSets(left.NodeSet, right.NodeSet, ctx.docOrder, maxNodeSetLength)
	if err != nil {
		return nil, err
	}
	return &Result{Type: NodeSetResult, NodeSet: merged}, nil
}

func evalPathExpr(ctx *evalContext, e PathExpr) (*Result, error) {
	r, err := eval(ctx, e.Filter)
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
		subCtx := ctx.withNode(n, 1, 1)
		subResult, err := evalLocationPath(subCtx, e.Path)
		if err != nil {
			return nil, err
		}
		result, err = ixpath.MergeNodeSets(result, subResult.NodeSet, ctx.docOrder, maxNodeSetLength)
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
		return numberToString(r.Number)
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

// numberToString converts a float64 to its XPath string representation,
// matching libxml2's xmlXPathFormatNumber behavior.
//
// libxml2 uses three formatting branches:
//  1. Integers within int32 range: decimal integer format (%d)
//  2. |value| >= 1e9 or |value| < 1e-5: scientific notation with trailing zero stripping
//  3. Otherwise: fixed notation with trailing zero stripping
//
// Both scientific and fixed branches use DBL_DIG (15) significant digits.
func numberToString(f float64) string {
	if math.IsNaN(f) {
		return "NaN"
	}
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	if f == 0 {
		return "0"
	}

	// Match libxml2: integers within int32 range use %d format.
	if f > math.MinInt32 && f < math.MaxInt32 && f == math.Trunc(f) {
		return strconv.Itoa(int(f))
	}

	abs := math.Abs(f)

	const (
		upperDouble = 1e9
		lowerDouble = 1e-5
		dblDig      = 15
	)

	var s string
	if abs >= upperDouble || abs < lowerDouble {
		// Scientific notation matching libxml2's %*.*e branch.
		s = strconv.FormatFloat(f, 'e', dblDig-1, 64)
	} else {
		// Fixed notation matching libxml2's %0.*f branch.
		intPlace := 1 + int(math.Log10(abs))
		fracPlace := (dblDig - 1) - intPlace
		if fracPlace < 0 {
			fracPlace = 0
		}
		s = strconv.FormatFloat(f, 'f', fracPlace, 64)
	}

	return trimNumberTrailingZeros(s)
}

// trimNumberTrailingZeros strips trailing zeros after the decimal point
// in a formatted number string. Handles both fixed and scientific notation.
// Matches libxml2's post-format zero stripping logic.
func trimNumberTrailingZeros(s string) string {
	eIdx := strings.IndexByte(s, 'e')

	var mantissa, exponent string
	if eIdx >= 0 {
		mantissa = s[:eIdx]
		exponent = s[eIdx:]
	} else {
		mantissa = s
	}

	dotIdx := strings.IndexByte(mantissa, '.')
	if dotIdx < 0 {
		return s
	}

	end := len(mantissa)
	for end > dotIdx+1 && mantissa[end-1] == '0' {
		end--
	}
	if mantissa[end-1] == '.' {
		end--
	}

	return mantissa[:end] + exponent
}
