package xpath

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

const (
	maxRecursionDepth = 5000
	maxNodeSetLength  = 10_000_000
)

// docOrderCache caches document-order positions for all nodes in a document.
// Built lazily on first use and shared across the entire evaluation tree.
type docOrderCache struct {
	positions map[helium.Node]int
}

func (c *docOrderCache) position(n helium.Node) int {
	if c.positions == nil {
		return -1
	}
	// Namespace nodes are not in the document tree. Position them
	// right after their parent element so they sort correctly.
	if n.Type() == helium.NamespaceNode {
		parent := n.Parent()
		if parent == nil {
			return 0
		}
		return c.position(parent)
	}
	pos, ok := c.positions[n]
	if !ok {
		return -1
	}
	return pos
}

func (c *docOrderCache) buildFrom(root helium.Node) {
	if c.positions != nil {
		return
	}
	c.positions = make(map[helium.Node]int)
	pos := 0
	c.indexWalk(root, &pos)
}

func (c *docOrderCache) indexWalk(cur helium.Node, pos *int) {
	c.positions[cur] = *pos
	*pos++
	if elem, ok := cur.(*helium.Element); ok {
		for _, attr := range elem.Attributes() {
			c.positions[helium.Node(attr)] = *pos
			*pos++
		}
	}
	for child := cur.FirstChild(); child != nil; child = child.NextSibling() {
		c.indexWalk(child, pos)
	}
}

// evalContext holds the evaluation state for an XPath expression.
type evalContext struct {
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
	docOrder    *docOrderCache
}

func newEvalContext(node helium.Node) *evalContext {
	opCount := 0
	return &evalContext{
		node:     node,
		position: 1,
		size:     1,
		opCount:  &opCount,
		docOrder: &docOrderCache{},
	}
}

func (ctx *evalContext) withNode(n helium.Node, pos, size int) *evalContext {
	return &evalContext{
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
		root := documentRoot(ctx.node)
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
	return deduplicateNodes(allFiltered, ctx.docOrder)
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
	return deduplicateNodes(next, ctx.docOrder)
}

// filterByNodeTest returns only those nodes that match the given node test.
func filterByNodeTest(candidates []helium.Node, nt NodeTest, axis AxisType, ctx *evalContext) []helium.Node {
	var matched []helium.Node
	for _, c := range candidates {
		if matchNodeTest(nt, c, axis, ctx) {
			matched = append(matched, c)
		}
	}
	return matched
}

// deduplicateNodes removes duplicate nodes and sorts in document order.
func deduplicateNodes(nodes []helium.Node, cache *docOrderCache) ([]helium.Node, error) {
	if len(nodes) <= 1 {
		return nodes, nil
	}
	seen := make(map[helium.Node]bool, len(nodes))
	nsKeys := make(map[nsNodeKey]bool)
	result := make([]helium.Node, 0, len(nodes))
	for _, n := range nodes {
		if seen[n] {
			continue
		}
		if n.Type() == helium.NamespaceNode {
			key := nsNodeKey{parent: n.Parent(), prefix: n.Name()}
			if nsKeys[key] {
				continue
			}
			nsKeys[key] = true
		}
		seen[n] = true
		result = append(result, n)
	}
	if len(result) > maxNodeSetLength {
		return nil, ErrNodeSetLimit
	}
	cache.buildFrom(documentRoot(result[0]))
	sort.SliceStable(result, func(i, j int) bool {
		return cache.position(result[i]) < cache.position(result[j])
	})
	return result, nil
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
		return matchNameTestNamespaceAxis(test, n)
	default:
		// For principal node type of other axes: element
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
		// prefix:* — match namespace
		return matchPrefix(test.Prefix, n, ctx)
	}

	// Match local name
	if localNameOf(n) != test.Local {
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
	// Node-set vs node-set
	if right.Type == NodeSetResult {
		for _, ln := range leftNodes {
			lv := stringValue(ln)
			for _, rn := range right.NodeSet {
				if compareStrings(op, lv, stringValue(rn)) {
					return true
				}
			}
		}
		return false
	}
	// Node-set vs scalar
	for _, ln := range leftNodes {
		if compareWithScalar(op, stringValue(ln), right) {
			return true
		}
	}
	return false
}

// compareNodeSetRight handles comparisons where only the right operand is a node-set.
func compareNodeSetRight(op TokenType, left *Result, rightNodes []helium.Node) bool {
	rev := reverseOp(op)
	for _, rn := range rightNodes {
		if compareWithScalar(rev, stringValue(rn), left) {
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
	// Relational: always compare as numbers
	return compareNumbers(op, resultToNumber(left), resultToNumber(right))
}

// compareScalarsEqNe compares two scalar results for equality or inequality
// according to XPath 1.0 type coercion rules.
func compareScalarsEqNe(op TokenType, left, right *Result) bool {
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
		return &Result{Type: BooleanResult, Boolean: val}, nil
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
	// Merge and deduplicate, preserving document order
	merged, err := mergeNodeSets(left.NodeSet, right.NodeSet, ctx.docOrder)
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
		result, err = mergeNodeSets(result, subResult.NodeSet, ctx.docOrder)
		if err != nil {
			return nil, err
		}
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
		// XPath spec 5.2: string-value of element/document is the
		// concatenation of string-values of all text node descendants.
		var b strings.Builder
		collectTextDescendants(n, &b)
		return b.String()
	case helium.TextNode, helium.CDATASectionNode:
		return string(n.Content())
	case helium.CommentNode:
		return string(n.Content())
	case helium.ProcessingInstructionNode:
		return string(n.Content())
	case helium.NamespaceNode:
		return string(n.Content())
	}
	return ""
}

// collectTextDescendants recursively collects text from Text and CDATA descendants.
func collectTextDescendants(n helium.Node, b *strings.Builder) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			b.Write(c.Content())
		default:
			collectTextDescendants(c, b)
		}
	}
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

	// Walk backwards past trailing zeros.
	end := len(mantissa)
	for end > dotIdx+1 && mantissa[end-1] == '0' {
		end--
	}
	// If only the dot remains, remove it too.
	if mantissa[end-1] == '.' {
		end--
	}

	return mantissa[:end] + exponent
}

// nsNodeKey identifies a namespace node by its parent element and prefix.
// NamespaceNodeWrapper objects are created fresh each time the namespace axis
// is traversed, so pointer-based identity fails for deduplication. We use
// value-based identity (parent pointer + prefix string) instead.
type nsNodeKey struct {
	parent helium.Node
	prefix string
}

// mergeNodeSets merges two node sets, deduplicating by identity,
// and sorting in document order.
func mergeNodeSets(a, b []helium.Node, cache *docOrderCache) ([]helium.Node, error) {
	seen := make(map[helium.Node]bool, len(a)+len(b))
	nsKeys := make(map[nsNodeKey]bool)
	var result []helium.Node

	addNode := func(n helium.Node) {
		if seen[n] {
			return
		}
		if n.Type() == helium.NamespaceNode {
			key := nsNodeKey{parent: n.Parent(), prefix: n.Name()}
			if nsKeys[key] {
				return
			}
			nsKeys[key] = true
		}
		seen[n] = true
		result = append(result, n)
	}

	for _, n := range a {
		addNode(n)
	}
	for _, n := range b {
		addNode(n)
	}
	if len(result) > maxNodeSetLength {
		return nil, ErrNodeSetLimit
	}
	if len(result) > 0 {
		cache.buildFrom(documentRoot(result[0]))
	}
	sort.SliceStable(result, func(i, j int) bool {
		return cache.position(result[i]) < cache.position(result[j])
	})
	return result, nil
}

