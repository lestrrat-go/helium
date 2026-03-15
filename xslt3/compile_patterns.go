package xslt3

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// Pattern is a compiled XSLT match pattern.
type Pattern struct {
	Alternatives   []*PatternAlt
	source         string
	xpathDefaultNS string // xpath-default-namespace at compile site
}

// PatternAlt is one alternative in a union pattern (separated by |).
type PatternAlt struct {
	expr     xpath3.Expr // the parsed XPath AST
	priority float64
}

// compilePattern compiles an XSLT match pattern string.
// XSLT patterns are a restricted subset of XPath expressions.
func compilePattern(s string, nsBindings map[string]string, xpathDefaultNS string) (*Pattern, error) {
	alts := splitPatternUnion(s)
	p := &Pattern{source: s, xpathDefaultNS: xpathDefaultNS}
	for _, alt := range alts {
		alt = strings.TrimSpace(alt)
		if alt == "" {
			continue
		}
		// Reject patterns wrapped in outer parentheses: "(pattern)"
		// These are not valid in the XSLT pattern grammar even though
		// they parse as valid XPath.
		if isOuterParenthesized(alt) {
			return nil, staticError(errCodeXTSE0340, "invalid match pattern %q: parenthesized pattern not allowed", alt)
		}
		// Reject 'union' keyword in patterns — XSLT patterns use '|', not 'union'
		if containsUnionKeyword(alt) {
			return nil, staticError(errCodeXTSE0340, "invalid match pattern %q: 'union' keyword not allowed in pattern", alt)
		}
		ast, err := xpath3.Parse(alt)
		if err != nil {
			return nil, staticError(errCodeXTSE0500, "invalid pattern %q: %v", alt, err)
		}
		if err := validatePatternExpr(ast); err != nil {
			return nil, staticError(errCodeXTSE0340, "invalid match pattern %q: %s", alt, err)
		}
		pa := &PatternAlt{
			expr:     ast,
			priority: computeDefaultPriority(ast),
		}
		p.Alternatives = append(p.Alternatives, pa)
	}
	if len(p.Alternatives) == 0 {
		return nil, staticError(errCodeXTSE0500, "empty pattern %q", s)
	}
	return p, nil
}

// validatePatternExpr validates that a parsed XPath expression is a valid
// XSLT match pattern. XSLT patterns are a restricted subset of XPath —
// certain constructs like variable references, arbitrary function calls,
// and parenthesized union expressions are not allowed.
func validatePatternExpr(expr xpath3.Expr) error {
	return validatePatternExprInner(expr, true)
}

func validatePatternExprInner(expr xpath3.Expr, topLevel bool) error {
	switch e := expr.(type) {
	case xpath3.LocationPath, *xpath3.LocationPath:
		// LocationPath patterns are valid (e.g., a/b/c, //x, /)
		return validateLocationPathPattern(expr)
	case xpath3.RootExpr, *xpath3.RootExpr:
		// "/" is valid
		return nil
	case xpath3.ContextItemExpr:
		// "." is valid
		return nil
	case xpath3.FilterExpr:
		// FilterExpr: a primary expression with predicates.
		// In patterns, only "." with predicates is allowed (e.g., .[pred]).
		// A root expression with predicates like /[doc] is NOT allowed.
		if _, ok := e.Expr.(xpath3.ContextItemExpr); ok {
			return nil // .[pred] is valid
		}
		if _, ok := e.Expr.(xpath3.RootExpr); ok {
			return fmt.Errorf("predicates on '/' are not allowed")
		}
		if _, ok := e.Expr.(*xpath3.RootExpr); ok {
			return fmt.Errorf("predicates on '/' are not allowed")
		}
		return fmt.Errorf("filter expression not allowed in pattern")
	case xpath3.PathExpr:
		// PathExpr: filter/step. Check the filter part is valid.
		return validatePatternPathExpr(e)
	case xpath3.PathStepExpr:
		// E1/E2 or E1//E2 where E2 is a non-axis expression
		return validatePatternPathStepExpr(e)
	case xpath3.VariableExpr:
		// Variable references are allowed in XSLT 3.0 patterns
		return nil
	case xpath3.FunctionCall:
		// Function calls: key(), id(), doc() with literal args are allowed
		if isAllowedPatternFunction(e) {
			return nil
		}
		return fmt.Errorf("function call %s() not allowed in pattern", e.Name)
	case xpath3.UnionExpr:
		// Union inside a pattern: valid in XSLT 3.0 as operands of
		// intersect/except or inside parenthesized path steps
		return nil
	case xpath3.IntersectExceptExpr:
		// intersect/except are valid pattern operators in XSLT 3.0
		if err := validatePatternExprInner(e.Left, false); err != nil {
			return err
		}
		return validatePatternExprInner(e.Right, false)
	case xpath3.BinaryExpr:
		// Binary expressions like "and union or" parse as operator expressions
		return fmt.Errorf("binary operator not allowed in pattern")
	default:
		return fmt.Errorf("expression type %T not allowed in pattern", expr)
	}
}

func validateLocationPathPattern(expr xpath3.Expr) error {
	var steps []xpath3.Step
	switch e := expr.(type) {
	case xpath3.LocationPath:
		steps = e.Steps
	case *xpath3.LocationPath:
		steps = e.Steps
	}
	// Check each step for disallowed constructs in predicates
	for _, step := range steps {
		for _, pred := range step.Predicates {
			if err := validatePredicateExpr(pred); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePatternPathExpr(e xpath3.PathExpr) error {
	// The filter part: only ContextItemExpr, RootExpr, or FunctionCall (key/id/doc)
	switch f := e.Filter.(type) {
	case xpath3.ContextItemExpr:
		// .[pred]/steps is valid
	case xpath3.RootExpr, *xpath3.RootExpr:
		// /steps is valid
	case xpath3.FunctionCall:
		// Only certain functions at the start of a path pattern
		if !isAllowedPatternFunction(f) {
			return fmt.Errorf("function call %s() not allowed at start of path pattern", f.Name)
		}
	case xpath3.FilterExpr:
		// FilterExpr at the start of a path
		return validatePatternExprInner(f, false)
	case xpath3.VariableExpr:
		// Variable references are allowed in XSLT 3.0 patterns
	default:
		return fmt.Errorf("expression type %T not allowed at start of path pattern", e.Filter)
	}
	return nil
}

func validatePatternPathStepExpr(e xpath3.PathStepExpr) error {
	// Check left side
	switch l := e.Left.(type) {
	case xpath3.VariableExpr:
		// Variable references are allowed in XSLT 3.0 patterns
	case xpath3.FunctionCall:
		if !isAllowedPatternFunction(l) {
			return fmt.Errorf("function call %s() not allowed in pattern", l.Name)
		}
	case xpath3.PathExpr:
		if err := validatePatternPathExpr(l); err != nil {
			return err
		}
	case xpath3.PathStepExpr:
		if err := validatePatternPathStepExpr(l); err != nil {
			return err
		}
	}
	// Check right side
	switch r := e.Right.(type) {
	case xpath3.VariableExpr:
		// Variable references are allowed in XSLT 3.0 patterns
	case xpath3.FunctionCall:
		return fmt.Errorf("function call %s() not allowed in middle of pattern", r.Name)
	case xpath3.ArrayConstructorExpr:
		return fmt.Errorf("array constructor not allowed in pattern")
	}
	return nil
}

// isAllowedPatternFunction returns true if a function call is allowed at the
// start of a path pattern. In XSLT 3.0, key(), id(), and doc() are allowed
// with literal arguments.
func isAllowedPatternFunction(fc xpath3.FunctionCall) bool {
	switch fc.Name {
	case "key", "id", "idref", "doc":
		// Check that all arguments are literals
		for _, arg := range fc.Args {
			if !isPatternFunctionArg(arg) {
				return false
			}
		}
		return true
	}
	return false
}

// isPatternFunctionArg returns true if an expression is a valid argument
// to a function in a pattern (must be a literal or variable reference).
func isPatternFunctionArg(expr xpath3.Expr) bool {
	switch expr.(type) {
	case xpath3.LiteralExpr:
		return true
	case xpath3.VariableExpr:
		return true
	}
	return false
}

// validatePredicateExpr validates expressions used inside predicates.
// Predicates in patterns can contain arbitrary XPath expressions.
func validatePredicateExpr(_ xpath3.Expr) error {
	// Predicates can contain any valid XPath expression,
	// so no additional validation needed.
	return nil
}

// containsUnionKeyword checks if a pattern string uses the 'union' keyword
// as an operator (not inside strings, predicates, or as an element name).
// XSLT patterns must use '|' for union, not the 'union' keyword.
func containsUnionKeyword(s string) bool {
	// Simple check: look for ' union ' (surrounded by spaces) outside of
	// strings and predicates/parens.
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'', '"':
			q := s[i]
			i++
			for i < len(s) && s[i] != q {
				i++
			}
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		default:
			if depth == 0 && i+5 < len(s) && s[i:i+5] == "union" {
				// Check word boundaries
				before := i == 0 || !isXPathNameChar(s[i-1])
				after := i+5 >= len(s) || !isXPathNameChar(s[i+5])
				if before && after {
					return true
				}
			}
		}
	}
	return false
}

func isXPathNameChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.'
}

// isOuterParenthesized checks if a pattern string is entirely wrapped in
// outer parentheses, e.g. "(doc|cod)" or "(.[. instance of xs:integer])".
// It returns false for patterns that merely contain parentheses in sub-expressions
// like "element(foo)" or "a[position()=1]".
func isOuterParenthesized(s string) bool {
	if len(s) < 2 || s[0] != '(' {
		return false
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i == len(s)-1
			}
		case '\'', '"':
			// Skip string literals
			q := s[i]
			i++
			for i < len(s) && s[i] != q {
				i++
			}
		}
	}
	return false
}

// splitPatternUnion splits a pattern string by | at the top level.
// It respects brackets, parentheses and string literals.
func splitPatternUnion(s string) []string {
	var parts []string
	depth := 0
	inSingle := false
	inDouble := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '(', '[':
			if !inSingle && !inDouble {
				depth++
			}
		case ')', ']':
			if !inSingle && !inDouble {
				depth--
			}
		case '|':
			if !inSingle && !inDouble && depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// computeDefaultPriority computes the default priority for a pattern
// alternative per XSLT 3.0 Section 6.4.
func computeDefaultPriority(expr xpath3.Expr) float64 {
	var steps []xpath3.Step
	switch e := expr.(type) {
	case xpath3.LocationPath:
		steps = e.Steps
	case *xpath3.LocationPath:
		steps = e.Steps
	case xpath3.RootExpr, *xpath3.RootExpr:
		// "/" matches document nodes; default priority is -0.5
		return -0.5
	case xpath3.ContextItemExpr:
		// "." (self::node()) — wildcard node test, priority -0.5
		return -0.5
	default:
		return 0.5
	}
	if len(steps) == 0 {
		// Absolute path with no steps is "/" — document root, priority -0.5
		if isAbsolute(expr) {
			return -0.5
		}
		return 0.5
	}
	// Per XSLT 3.0 §6.4: path patterns with multiple steps get priority 0.5
	if len(steps) > 1 {
		return 0.5
	}
	lastStep := steps[0]
	// A step with predicates gets priority 0.5 (more specific)
	if len(lastStep.Predicates) > 0 {
		return 0.5
	}
	return stepPriority(lastStep)
}

func isAbsolute(expr xpath3.Expr) bool {
	switch e := expr.(type) {
	case xpath3.LocationPath:
		return e.Absolute
	case *xpath3.LocationPath:
		return e.Absolute
	}
	return false
}

func stepPriority(step xpath3.Step) float64 {
	switch nt := step.NodeTest.(type) {
	case xpath3.TypeTest:
		// node(), text(), comment(), processing-instruction()
		return -0.5
	case xpath3.PITest:
		if nt.Target != "" {
			return 0
		}
		return -0.5
	case xpath3.NameTest:
		if nt.Local == "*" && nt.Prefix == "" && nt.URI == "" {
			return -0.5
		}
		if nt.Local == "*" || nt.Prefix == "*" {
			// prefix:* or *:local — one component is wildcard
			return -0.25
		}
		return 0
	case xpath3.ElementTest:
		if nt.Name == "" || nt.Name == "*" {
			return -0.5 // element() or element(*)
		}
		return 0 // element(specific-name)
	case xpath3.AttributeTest:
		if nt.Name == "" || nt.Name == "*" {
			return -0.5 // attribute() or attribute(*)
		}
		return 0 // attribute(specific-name)
	case xpath3.DocumentTest:
		if nt.Inner == nil {
			return -0.5
		}
		// Priority of document-node(test) derives from the inner test
		return nodeTestPriority(nt.Inner)
	default:
		return 0.5
	}
}

// nodeTestPriority computes the default priority for a node test used inside
// document-node() or similar container tests.
func nodeTestPriority(test xpath3.NodeTest) float64 {
	switch nt := test.(type) {
	case xpath3.ElementTest:
		if nt.Name == "" || nt.Name == "*" {
			return -0.5
		}
		return 0
	case xpath3.AttributeTest:
		if nt.Name == "" || nt.Name == "*" {
			return -0.5
		}
		return 0
	case xpath3.TypeTest:
		return -0.5
	default:
		return 0.5
	}
}

// matchPattern tests whether a node matches the pattern.
func (p *Pattern) matchPattern(ctx *execContext, node helium.Node) bool {
	// Temporarily set xpath-default-namespace from pattern's compile-time value
	saved := ctx.xpathDefaultNS
	savedHas := ctx.hasXPathDefaultNS
	ctx.xpathDefaultNS = p.xpathDefaultNS
	ctx.hasXPathDefaultNS = p.xpathDefaultNS != ""
	defer func() {
		ctx.xpathDefaultNS = saved
		ctx.hasXPathDefaultNS = savedHas
	}()

	for _, alt := range p.Alternatives {
		if matchPatternAlt(ctx, alt, node) {
			return true
		}
	}
	return false
}

// matchPatternAlt tests whether a node matches a single pattern alternative.
// XSLT patterns evaluate by checking if the node would be selected by the
// equivalent XPath expression when evaluated in the right context.
func matchPatternAlt(ctx *execContext, alt *PatternAlt, node helium.Node) bool {
	switch e := alt.expr.(type) {
	case xpath3.LocationPath:
		return matchLocationPath(ctx, e, node)
	case *xpath3.LocationPath:
		return matchLocationPath(ctx, *e, node)
	case xpath3.RootExpr:
		return node.Type() == helium.DocumentNode
	case *xpath3.RootExpr:
		return node.Type() == helium.DocumentNode
	case xpath3.ContextItemExpr:
		// "." matches any node (equivalent to self::node()).
		return true
	default:
		// For complex expressions, try evaluating from document root
		// and checking if node is in the result set.
		return matchByEvaluation(ctx, alt, node)
	}
}

// matchLocationPath matches a LocationPath pattern against a node.
// Patterns match bottom-up: starting from the node, check if there's
// a path from the root that would select this node.
func matchLocationPath(ctx *execContext, path xpath3.LocationPath, node helium.Node) bool {
	if len(path.Steps) == 0 {
		// "/" matches document nodes
		return path.Absolute && node.Type() == helium.DocumentNode
	}

	// Check the last step against the node
	lastStep := path.Steps[len(path.Steps)-1]
	if !nodeMatchesStep(ctx, lastStep, node) {
		return false
	}

	// If there's only one step and it's absolute, check parent is document
	if len(path.Steps) == 1 {
		if path.Absolute {
			parent := node.Parent()
			return parent != nil && parent.Type() == helium.DocumentNode
		}
		return true
	}

	// Match remaining steps upward
	return matchStepsUpward(ctx, path.Steps[:len(path.Steps)-1], path.Absolute, node.Parent())
}

// matchStepsUpward matches remaining pattern steps upward through ancestors.
func matchStepsUpward(ctx *execContext, steps []xpath3.Step, absolute bool, node helium.Node) bool {
	if len(steps) == 0 {
		if absolute {
			return node != nil && node.Type() == helium.DocumentNode
		}
		return true
	}
	if node == nil {
		return false
	}

	lastStep := steps[len(steps)-1]

	switch lastStep.Axis {
	case xpath3.AxisChild:
		// child axis in pattern means "the parent must match"
		if !nodeMatchesStep(ctx, lastStep, node) {
			return false
		}
		return matchStepsUpward(ctx, steps[:len(steps)-1], absolute, node.Parent())
	case xpath3.AxisDescendantOrSelf:
		// // in pattern means any ancestor may match
		// This step is the desc-or-self::node() inserted by //
		// Try matching from this node and any ancestor
		remaining := steps[:len(steps)-1]
		for cur := node; cur != nil; cur = cur.Parent() {
			if matchStepsUpward(ctx, remaining, absolute, cur) {
				return true
			}
		}
		return false
	default:
		if !nodeMatchesStep(ctx, lastStep, node) {
			return false
		}
		return matchStepsUpward(ctx, steps[:len(steps)-1], absolute, node.Parent())
	}
}

// nodeMatchesStep checks if a node matches a single step's node test and predicates.
func nodeMatchesStep(ctx *execContext, step xpath3.Step, node helium.Node) bool {
	// Document nodes are matched by document-node() patterns, or by node()
	// on the self axis (as in match=".").  On the child/descendant axes,
	// document nodes are never selected.
	if node.Type() == helium.DocumentNode {
		if _, ok := step.NodeTest.(xpath3.DocumentTest); !ok {
			// Allow self::node() to match document nodes.
			if step.Axis == xpath3.AxisSelf {
				if tt, ok := step.NodeTest.(xpath3.TypeTest); !ok || tt.Kind != xpath3.NodeKindNode {
					return false
				}
			} else {
				return false
			}
		}
	}
	// Attribute nodes are only matched by the attribute axis (e.g., @*, attribute::node()).
	// The default child axis does not include attributes.
	if node.Type() == helium.AttributeNode && step.Axis != xpath3.AxisAttribute {
		return false
	}
	// Conversely, the attribute axis only matches attribute nodes.
	if node.Type() != helium.AttributeNode && step.Axis == xpath3.AxisAttribute {
		return false
	}
	if !nodeMatchesTest(ctx, step.NodeTest, node) {
		return false
	}
	// Evaluate predicates if any, with chained position filtering
	if len(step.Predicates) > 0 {
		return evaluateChainedPredicates(ctx, step, node)
	}
	return true
}

// nodeMatchesTest checks if a node matches a node test.
func nodeMatchesTest(ctx *execContext, test xpath3.NodeTest, node helium.Node) bool {
	switch nt := test.(type) {
	case xpath3.TypeTest:
		return matchTypeTest(nt, node)
	case xpath3.NameTest:
		return matchNameTest(ctx, nt, node)
	case xpath3.PITest:
		if node.Type() != helium.ProcessingInstructionNode {
			return false
		}
		if nt.Target == "" {
			return true
		}
		pi, ok := node.(*helium.ProcessingInstruction)
		return ok && pi.Name() == nt.Target
	case xpath3.ElementTest:
		return matchElementTest(ctx, nt, node)
	case xpath3.AttributeTest:
		return matchAttributeTest(ctx, nt, node)
	case xpath3.DocumentTest:
		return matchDocumentTest(ctx, nt, node)
	default:
		return false
	}
}

func matchTypeTest(tt xpath3.TypeTest, node helium.Node) bool {
	switch tt.Kind {
	case xpath3.NodeKindNode:
		return true
	case xpath3.NodeKindText:
		return node.Type() == helium.TextNode || node.Type() == helium.CDATASectionNode
	case xpath3.NodeKindComment:
		return node.Type() == helium.CommentNode
	case xpath3.NodeKindProcessingInstruction:
		return node.Type() == helium.ProcessingInstructionNode
	}
	return false
}

func matchNameTest(ctx *execContext, nt xpath3.NameTest, node helium.Node) bool {
	if node.Type() != helium.ElementNode && node.Type() != helium.AttributeNode {
		return false
	}
	elem, isElem := node.(*helium.Element)
	attr, isAttr := node.(*helium.Attribute)

	if nt.Local == "*" && nt.Prefix == "" && nt.URI == "" {
		// * matches all elements/attributes
		return true
	}

	var nodeLocal, nodeURI string
	if isElem {
		nodeLocal = elem.LocalName()
		nodeURI = elem.URI()
	} else if isAttr {
		nodeLocal = attr.LocalName()
		nodeURI = attr.URI()
	} else {
		return false
	}

	if nt.Local == "*" {
		// prefix:* matches any local name in the given namespace
		uri := nt.URI
		if uri == "" && nt.Prefix != "" && ctx != nil {
			uri = ctx.resolvePrefix(nt.Prefix)
		}
		return nodeURI == uri
	}

	// Exact name match
	if nt.URI != "" {
		return nodeLocal == nt.Local && nodeURI == nt.URI
	}

	if nt.Prefix == "*" {
		// *:local matches any namespace
		return nodeLocal == nt.Local
	}
	expectedURI := ""
	if nt.Prefix != "" && ctx != nil {
		expectedURI = ctx.resolvePrefix(nt.Prefix)
	} else if nt.Prefix == "" && ctx != nil && node.Type() == helium.ElementNode {
		// Unprefixed element names use xpath-default-namespace
		expectedURI = ctx.xpathDefaultNS
	}
	return nodeLocal == nt.Local && nodeURI == expectedURI
}

// matchElementTest checks if a node matches an element() test.
func matchElementTest(ctx *execContext, et xpath3.ElementTest, node helium.Node) bool {
	if node.Type() != helium.ElementNode {
		return false
	}
	if et.Name == "" || et.Name == "*" {
		return true
	}
	elem, ok := node.(*helium.Element)
	if !ok {
		return false
	}
	// Check local name match (may contain prefix:local)
	name := et.Name
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		uri := ""
		if ctx != nil {
			uri = ctx.resolvePrefix(prefix)
		}
		return elem.LocalName() == local && elem.URI() == uri
	}
	return elem.LocalName() == name
}

// matchAttributeTest checks if a node matches an attribute() test.
func matchAttributeTest(ctx *execContext, at xpath3.AttributeTest, node helium.Node) bool {
	if node.Type() != helium.AttributeNode {
		return false
	}
	if at.Name == "" || at.Name == "*" {
		return true
	}
	attr, ok := node.(*helium.Attribute)
	if !ok {
		return false
	}
	name := at.Name
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		uri := ""
		if ctx != nil {
			uri = ctx.resolvePrefix(prefix)
		}
		return attr.LocalName() == local && attr.URI() == uri
	}
	return attr.LocalName() == name
}

// matchDocumentTest checks if a node matches a document-node() test.
func matchDocumentTest(ctx *execContext, dt xpath3.DocumentTest, node helium.Node) bool {
	if node.Type() != helium.DocumentNode {
		return false
	}
	if dt.Inner == nil {
		return true
	}
	// document-node(element(name)) — check that the document has a single
	// element child matching the inner test.
	doc, ok := node.(*helium.Document)
	if !ok {
		return false
	}
	var docElem helium.Node
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			if docElem != nil {
				return false // more than one element child
			}
			docElem = child
		}
	}
	if docElem == nil {
		return false
	}
	return nodeMatchesTest(ctx, dt.Inner, docElem)
}

// evaluateChainedPredicates evaluates a chain of predicates in a pattern step.
// For patterns with multiple predicates like x[P1][P2][P3], each predicate
// filters based on position among siblings matching the node test AND all
// previous predicates. This implements XSLT 3.0 Section 5.5.3.
func evaluateChainedPredicates(ctx *execContext, step xpath3.Step, node helium.Node) bool {
	// Collect all same-test siblings (including the node itself)
	siblings := collectMatchingSiblings(ctx, step.NodeTest, node)

	for i, pred := range step.Predicates {
		// Find position and size of node in current filtered sibling set
		pos := 0
		for j, sib := range siblings {
			if sib == node {
				pos = j + 1
				break
			}
		}
		if pos == 0 {
			return false // node not in the filtered set
		}

		// Evaluate the predicate with position/size context
		if !evaluatePredicateWithPosition(ctx, pred, node, pos, len(siblings)) {
			return false
		}

		// For subsequent predicates, filter the sibling list
		if i < len(step.Predicates)-1 {
			var filtered []helium.Node
			for j, sib := range siblings {
				if evaluatePredicateWithPosition(ctx, pred, sib, j+1, len(siblings)) {
					filtered = append(filtered, sib)
				}
			}
			siblings = filtered
		}
	}
	return true
}

// collectMatchingSiblings collects all siblings (including the node itself)
// that match the given node test, in document order.
func collectMatchingSiblings(ctx *execContext, test xpath3.NodeTest, node helium.Node) []helium.Node {
	var siblings []helium.Node

	// Find the parent to iterate children
	parent := node.Parent()
	if parent == nil {
		// No parent means we can only check the node itself
		return []helium.Node{node}
	}

	// Iterate through all children of the parent
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if nodeMatchesTest(ctx, test, child) {
			siblings = append(siblings, child)
		}
	}
	return siblings
}

// evaluatePredicateWithPosition evaluates a pattern predicate with explicit
// position and size context.
func evaluatePredicateWithPosition(ctx *execContext, pred xpath3.Expr, node helium.Node, pos, size int) bool {
	xpathCtx := ctx.newXPathContext(node)
	xpathCtx = xpath3.WithPosition(xpathCtx, pos)
	xpathCtx = xpath3.WithSize(xpathCtx, size)
	result, err := xpath3.EvaluateExpr(xpathCtx, pred, node)
	if err != nil {
		return false
	}
	// Numeric predicates: compare to the provided position
	if f, ok := result.IsNumber(); ok {
		return int(f) == pos
	}
	b, err := xpath3.EBV(result.Sequence())
	if err != nil {
		return false
	}
	return b
}

// matchByEvaluation matches complex patterns by evaluating from document root.
//
// TODO(xslt3): implement evaluation-based pattern matching for non-LocationPath
// patterns (e.g., key(), id(), doc()). This requires evaluating the pattern
// expression from the document root and checking whether the candidate node
// appears in the result sequence. Currently returns false, so templates with
// these pattern forms will never match. This is a known limitation of the
// initial XSLT 3.0 implementation — not a bug to be fixed in isolation.
func matchByEvaluation(ctx *execContext, alt *PatternAlt, node helium.Node) bool {
	compiled := xpath3.CompileExpr(alt.expr)

	// Try evaluating from each ancestor up to the document root.
	// This handles nodes in variable trees where the document root
	// has a synthetic wrapper element.
	for ancestor := node.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
		xpathCtx := ctx.newXPathContext(ancestor)
		result, err := compiled.Evaluate(xpathCtx, ancestor)
		if err != nil {
			continue
		}
		for _, item := range result.Sequence() {
			if ni, ok := item.(xpath3.NodeItem); ok && ni.Node == node {
				return true
			}
		}
	}
	return false
}
