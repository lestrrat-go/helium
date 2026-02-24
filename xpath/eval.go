package xpath

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

// evalContext holds the evaluation state for an XPath expression.
type evalContext struct {
	node       helium.Node
	position   int
	size       int
	namespaces map[string]string
	variables  map[string]interface{}
}

func newEvalContext(node helium.Node) *evalContext {
	return &evalContext{
		node:     node,
		position: 1,
		size:     1,
	}
}

func (ctx *evalContext) withNode(n helium.Node, pos, size int) *evalContext {
	return &evalContext{
		node:       n,
		position:   pos,
		size:       size,
		namespaces: ctx.namespaces,
		variables:  ctx.variables,
	}
}

// eval dispatches to the appropriate evaluator for each AST node type.
func eval(ctx *evalContext, expr Expr) (*Result, error) {
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
	case FilterExpr:
		return evalFilterExpr(ctx, e)
	case UnionExpr:
		return evalUnionExpr(ctx, e)
	case PathExpr:
		return evalPathExpr(ctx, e)
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func evalLocationPath(ctx *evalContext, lp *LocationPath) (*Result, error) {
	var nodes []helium.Node

	if lp.Absolute {
		root := documentRoot(ctx.node)
		nodes = []helium.Node{root}
	} else {
		nodes = []helium.Node{ctx.node}
	}

	for _, step := range lp.Steps {
		var next []helium.Node
		for _, n := range nodes {
			candidates := traverseAxis(step.Axis, n)
			for _, c := range candidates {
				if matchNodeTest(step.NodeTest, c, step.Axis, ctx) {
					next = append(next, c)
				}
			}
		}
		nodes = next

		// Apply predicates
		for _, pred := range step.Predicates {
			filtered, err := applyPredicate(ctx, nodes, pred)
			if err != nil {
				return nil, err
			}
			nodes = filtered
		}
	}

	return &Result{Type: NodeSetResult, NodeSet: nodes}, nil
}

func documentRoot(n helium.Node) helium.Node {
	if doc := n.OwnerDocument(); doc != nil {
		return doc
	}
	// Walk up parent chain
	for n.Parent() != nil {
		n = n.Parent()
	}
	return n
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
		// Use type assertion because Attribute.Type() may not be set correctly
		if _, ok := n.(*helium.Attribute); !ok {
			return false
		}
	case AxisNamespace:
		return false // namespace axis not supported
	default:
		// For principal node type of other axes: element
		if n.Type() != helium.ElementNode {
			return false
		}
	}

	if test.Local == "*" {
		if test.Prefix == "" {
			return true
		}
		// prefix:* — match namespace
		return matchPrefix(test.Prefix, n, ctx)
	}

	// Match local name
	localName := localNameOf(n)
	if localName != test.Local {
		return false
	}

	if test.Prefix != "" {
		return matchPrefix(test.Prefix, n, ctx)
	}

	return true
}

func matchPrefix(prefix string, n helium.Node, ctx *evalContext) bool {
	// Resolve prefix to URI via context namespaces
	if ctx.namespaces != nil {
		uri, ok := ctx.namespaces[prefix]
		if ok {
			return nodeNamespaceURI(n) == uri
		}
	}
	// Fall back to node's own prefix
	return nodePrefix(n) == prefix
}

func localNameOf(n helium.Node) string {
	switch v := n.(type) {
	case *helium.Element:
		return v.LocalName()
	case *helium.Attribute:
		// Attribute.Name() returns prefix:local, LocalName returns just local
		return v.LocalName()
	default:
		return n.Name()
	}
}

func nodePrefix(n helium.Node) string {
	type prefixer interface {
		Prefix() string
	}
	if p, ok := n.(prefixer); ok {
		return p.Prefix()
	}
	return ""
}

func nodeNamespaceURI(n helium.Node) string {
	type urier interface {
		URI() string
	}
	if u, ok := n.(urier); ok {
		return u.URI()
	}
	return ""
}

func matchTypeTest(test TypeTest, n helium.Node) bool {
	switch test.Type {
	case NodeTestNode:
		return true
	case NodeTestText:
		return n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode
	case NodeTestComment:
		return n.Type() == helium.CommentNode
	case NodeTestPI:
		return n.Type() == helium.ProcessingInstructionNode
	}
	return false
}

func applyPredicate(ctx *evalContext, nodes []helium.Node, pred Expr) ([]helium.Node, error) {
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
// - If result is a number, it's true when equal to position
// - Otherwise, convert to boolean
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
	case TokenEquals:
		return evalComparison(ctx, e)
	case TokenNotEquals:
		return evalComparison(ctx, e)
	case TokenLess:
		return evalComparison(ctx, e)
	case TokenLessEq:
		return evalComparison(ctx, e)
	case TokenGreater:
		return evalComparison(ctx, e)
	case TokenGreaterEq:
		return evalComparison(ctx, e)
	case TokenPlus:
		return evalArithmetic(ctx, e)
	case TokenMinus:
		return evalArithmetic(ctx, e)
	case TokenStar:
		return evalArithmetic(ctx, e)
	case TokenDiv:
		return evalArithmetic(ctx, e)
	case TokenMod:
		return evalArithmetic(ctx, e)
	}
	return nil, fmt.Errorf("unsupported binary operator: %s", e.Op)
}

func evalOr(ctx *evalContext, e BinaryExpr) (*Result, error) {
	left, err := eval(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	if resultToBoolean(left) {
		return &Result{Type: BooleanResult, Boolean: true}, nil
	}
	right, err := eval(ctx, e.Right)
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Boolean: resultToBoolean(right)}, nil
}

func evalAnd(ctx *evalContext, e BinaryExpr) (*Result, error) {
	left, err := eval(ctx, e.Left)
	if err != nil {
		return nil, err
	}
	if !resultToBoolean(left) {
		return &Result{Type: BooleanResult, Boolean: false}, nil
	}
	right, err := eval(ctx, e.Right)
	if err != nil {
		return nil, err
	}
	return &Result{Type: BooleanResult, Boolean: resultToBoolean(right)}, nil
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
	return &Result{Type: BooleanResult, Boolean: b}, nil
}

// compareResults implements XPath comparison semantics including node-set comparisons.
func compareResults(op TokenType, left, right *Result) bool {
	// Node-set vs node-set
	if left.Type == NodeSetResult && right.Type == NodeSetResult {
		for _, ln := range left.NodeSet {
			lv := stringValue(ln)
			for _, rn := range right.NodeSet {
				rv := stringValue(rn)
				if compareStrings(op, lv, rv) {
					return true
				}
			}
		}
		return false
	}

	// Node-set vs other
	if left.Type == NodeSetResult {
		for _, ln := range left.NodeSet {
			sv := stringValue(ln)
			if compareWithScalar(op, sv, right) {
				return true
			}
		}
		return false
	}

	// Other vs node-set
	if right.Type == NodeSetResult {
		for _, rn := range right.NodeSet {
			sv := stringValue(rn)
			if compareWithScalar(reverseOp(op), sv, left) {
				return true
			}
		}
		return false
	}

	// Both scalars
	if op == TokenEquals || op == TokenNotEquals {
		// If either is boolean, compare as booleans
		if left.Type == BooleanResult || right.Type == BooleanResult {
			lb := resultToBoolean(left)
			rb := resultToBoolean(right)
			if op == TokenEquals {
				return lb == rb
			}
			return lb != rb
		}
		// If either is number, compare as numbers
		if left.Type == NumberResult || right.Type == NumberResult {
			ln := resultToNumber(left)
			rn := resultToNumber(right)
			if op == TokenEquals {
				return ln == rn
			}
			return ln != rn
		}
		// Compare as strings
		ls := resultToString(left)
		rs := resultToString(right)
		if op == TokenEquals {
			return ls == rs
		}
		return ls != rs
	}

	// Relational: always compare as numbers
	ln := resultToNumber(left)
	rn := resultToNumber(right)
	return compareNumbers(op, ln, rn)
}

func compareWithScalar(op TokenType, nodeStr string, scalar *Result) bool {
	switch scalar.Type {
	case BooleanResult:
		nb := len(nodeStr) > 0
		if op == TokenEquals {
			return nb == scalar.Boolean
		}
		if op == TokenNotEquals {
			return nb != scalar.Boolean
		}
		// Relational with boolean: convert both to number
		var ln, rn float64
		if nb {
			ln = 1
		}
		if scalar.Boolean {
			rn = 1
		}
		return compareNumbers(op, ln, rn)
	case NumberResult:
		nn := stringToNumber(nodeStr)
		return compareNumbers(op, nn, scalar.Number)
	default: // StringResult
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
		if rn == 0 {
			if ln == 0 {
				result = math.NaN()
			} else if ln > 0 {
				result = math.Inf(1)
			} else {
				result = math.Inf(-1)
			}
		} else {
			result = ln / rn
		}
	case TokenMod:
		if rn == 0 {
			result = math.NaN()
		} else {
			result = math.Mod(ln, rn)
		}
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
		return nil, fmt.Errorf("undefined variable: $%s", e.Name)
	}
	v, ok := ctx.variables[e.Name]
	if !ok {
		return nil, fmt.Errorf("undefined variable: $%s", e.Name)
	}
	switch val := v.(type) {
	case []helium.Node:
		return &Result{Type: NodeSetResult, NodeSet: val}, nil
	case string:
		return &Result{Type: StringResult, String: val}, nil
	case float64:
		return &Result{Type: NumberResult, Number: val}, nil
	case bool:
		return &Result{Type: BooleanResult, Boolean: val}, nil
	default:
		return nil, fmt.Errorf("unsupported variable type for $%s: %T", e.Name, v)
	}
}

func evalFilterExpr(ctx *evalContext, e FilterExpr) (*Result, error) {
	r, err := eval(ctx, e.Expr)
	if err != nil {
		return nil, err
	}
	if r.Type != NodeSetResult {
		return nil, fmt.Errorf("filter expression requires a node-set")
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
		return nil, fmt.Errorf("union operator requires node-sets")
	}
	// Merge and deduplicate, preserving document order
	merged := mergeNodeSets(left.NodeSet, right.NodeSet)
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
		return nil, fmt.Errorf("path expression requires a node-set")
	}

	var result []helium.Node
	for _, n := range r.NodeSet {
		subCtx := ctx.withNode(n, 1, 1)
		subResult, err := evalLocationPath(subCtx, e.Path)
		if err != nil {
			return nil, err
		}
		result = mergeNodeSets(result, subResult.NodeSet)
	}
	return &Result{Type: NodeSetResult, NodeSet: result}, nil
}

// stringValue returns the string-value of a node per XPath spec.
func stringValue(n helium.Node) string {
	// Check Attribute by type assertion first since etype may not be set
	if attr, ok := n.(*helium.Attribute); ok {
		return attr.Value()
	}
	switch n.Type() {
	case helium.DocumentNode, helium.ElementNode:
		return string(n.Content())
	case helium.TextNode, helium.CDATASectionNode:
		return string(n.Content())
	case helium.CommentNode:
		return string(n.Content())
	case helium.ProcessingInstructionNode:
		return string(n.Content())
	}
	return ""
}

// resultToString converts any Result to a string per XPath spec.
func resultToString(r *Result) string {
	switch r.Type {
	case StringResult:
		return r.String
	case BooleanResult:
		if r.Boolean {
			return "true"
		}
		return "false"
	case NumberResult:
		return numberToString(r.Number)
	case NodeSetResult:
		if len(r.NodeSet) == 0 {
			return ""
		}
		return stringValue(r.NodeSet[0])
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
		if r.Boolean {
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
		return r.Boolean
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
	// Use the shortest representation
	s := strconv.FormatFloat(f, 'f', -1, 64)
	return s
}

// mergeNodeSets merges two node sets, deduplicating by identity,
// and sorting in document order.
func mergeNodeSets(a, b []helium.Node) []helium.Node {
	seen := make(map[helium.Node]bool, len(a)+len(b))
	var result []helium.Node
	for _, n := range a {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	for _, n := range b {
		if !seen[n] {
			seen[n] = true
			result = append(result, n)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return documentPosition(result[i]) < documentPosition(result[j])
	})
	return result
}

// documentPosition returns an integer representing document order.
// This is a simple implementation that counts traversal steps from root.
func documentPosition(n helium.Node) int {
	pos := 0
	// Count ancestors
	var ancestors []helium.Node
	for p := n.Parent(); p != nil; p = p.Parent() {
		ancestors = append(ancestors, p)
	}
	// From root down, count sibling position at each level
	for i := len(ancestors) - 1; i >= 0; i-- {
		parent := ancestors[i]
		idx := 0
		for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
			idx++
			if i > 0 && c == ancestors[i-1] {
				break
			}
			if i == 0 && c == n {
				break
			}
		}
		pos = pos*1000 + idx
	}
	return pos
}
