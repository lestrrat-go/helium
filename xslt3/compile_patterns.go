package xslt3

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
)

// pattern is a compiled XSLT match pattern.
type pattern struct {
	Alternatives   []*patternAlt
	source         string
	xpathDefaultNS string            // xpath-default-namespace at compile site
	nsBindings     map[string]string // prefix→URI bindings from compile context
}

// patternAlt is one alternative in a union pattern (separated by |).
type patternAlt struct {
	expr         xpath3.Expr       // the parsed XPath AST
	compiled     *xpath3.Expression // cached compiled expression for runtime matching
	priority     float64
	neverMatches bool // true for syntactically valid but semantically empty patterns (e.g., child::document-node())
}

// isNeverMatchingPattern detects patterns that are syntactically valid but
// semantically can never match any node. For example, child::document-node()
// never matches because document nodes cannot be children; similarly,
// child::attribute() never matches because attributes are not children.
func isNeverMatchingPattern(alt string) bool {
	// NOTE: this is a heuristic based on source text, not AST analysis.
	// It may miss patterns with unusual whitespace around "::" but is
	// safe in the false-negative direction (worst case: we compile a
	// pattern that never matches at runtime).
	s := strings.TrimSpace(alt)
	// child::document-node() — document nodes are never children
	if strings.HasPrefix(s, "child::document-node") || strings.Contains(s, "/child::document-node") {
		return true
	}
	// child::attribute() / child::schema-attribute() — attributes are never children
	if strings.HasPrefix(s, "child::attribute") || strings.Contains(s, "/child::attribute") ||
		strings.HasPrefix(s, "child::schema-attribute") || strings.Contains(s, "/child::schema-attribute") {
		return true
	}
	// child::namespace-node() — namespace nodes are never children
	if strings.HasPrefix(s, "child::namespace-node") || strings.Contains(s, "/child::namespace-node") {
		return true
	}
	return false
}

// compilePattern compiles an XSLT match pattern string.
// XSLT patterns are a restricted subset of XPath expressions.
func compilePattern(s string, nsBindings map[string]string, xpathDefaultNS string) (*pattern, error) {
	alts := splitPatternUnion(s)
	p := &pattern{source: s, xpathDefaultNS: xpathDefaultNS, nsBindings: nsBindings}
	for _, alt := range alts {
		alt = strings.TrimSpace(alt)
		if alt == "" {
			continue
		}
		compiled, err := xpath3.NewCompiler().Compile(alt)
		if err != nil {
			return nil, staticError(errCodeXTSE0500, "invalid pattern %q: %v", alt, err)
		}
		// Run static validation (prefix checks) against the stylesheet
		// namespace bindings.
		if valErr := compiled.Validate(nsBindings); valErr != nil {
			return nil, staticError(errCodeXTSE0340, "invalid match pattern %q: %v", alt, valErr)
		}
		ast := compiled.AST()
		// Parenthesized predicate patterns are not allowed: (.[pred])
		// Skip XPath comments (:...:) which start with "(" but aren't parens.
		trimmedAlt := strings.TrimSpace(alt)
		if strings.HasPrefix(trimmedAlt, "(") && !strings.HasPrefix(trimmedAlt, "(:") &&
			strings.HasSuffix(trimmedAlt, ")") {
			if fe, ok := ast.(xpath3.FilterExpr); ok {
				if _, isCtx := fe.Expr.(xpath3.ContextItemExpr); isCtx {
					return nil, staticError(errCodeXTSE0340, "invalid match pattern %q: parenthesized predicate pattern not allowed", alt)
				}
			}
		}
		if err := validatePatternExpr(ast); err != nil {
			return nil, staticError(errCodeXTSE0340, "invalid match pattern %q: %s", alt, err)
		}
		// XTSE3500: current-merge-key() must not be used within a pattern.
		if err := checkPatternForbiddenFunctions(ast); err != nil {
			return nil, err
		}
		compiledExpr, compErr := xpath3.NewCompiler().CompileExpr(ast)
		if compErr != nil {
			return nil, staticError(errCodeXTSE0500, "pattern compile failed %q: %v", alt, compErr)
		}
		pa := &patternAlt{
			expr:         ast,
			compiled:     compiledExpr,
			priority:     computeDefaultPriority(ast),
			neverMatches: isNeverMatchingPattern(alt),
		}
		p.Alternatives = append(p.Alternatives, pa)
	}
	if len(p.Alternatives) == 0 {
		return nil, staticError(errCodeXTSE0500, "empty pattern %q", s)
	}
	// Union patterns (multiple alternatives separated by |) require each
	// alternative to be a PathPattern. PredicatePatterns (FilterExpr like
	// .[pred]) are only valid as standalone patterns, not in unions.
	// Note: FilterExpr wrapping a parenthesized node test with predicates
	// (e.g. (* except xsl:*)[@attr]) is valid in a union.
	if len(p.Alternatives) > 1 {
		for _, pa := range p.Alternatives {
			if fe, ok := pa.expr.(xpath3.FilterExpr); ok {
				if _, isCtx := fe.Expr.(xpath3.ContextItemExpr); isCtx {
					return nil, staticError(errCodeXTSE0340,
						"invalid match pattern %q: predicate pattern not allowed in union", s)
				}
			}
		}
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

func validatePatternExprInner(expr xpath3.Expr, _ bool) error {
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
		// XSLT 3.0 allows various filter patterns:
		//   .[pred], (/)[pred], $var[pred], (union)[pred], etc.
		switch e.Expr.(type) {
		case xpath3.ContextItemExpr:
			return nil // .[pred]
		case xpath3.RootExpr:
			return nil // (/)[pred]
		case xpath3.VariableExpr:
			return nil // $var[pred]
		case xpath3.UnionExpr:
			return nil // (a|b)[pred]
		case xpath3.IntersectExceptExpr:
			return nil // (a except b)[pred]
		case xpath3.FunctionCall:
			return nil // fn()[pred] — e.g., root()[self::A]
		case xpath3.LocationPath:
			return nil // (path)[pred]
		case *xpath3.LocationPath:
			lp := e.Expr.(*xpath3.LocationPath)
			if lp.Absolute && len(lp.Steps) == 0 {
				return nil // (/)[pred]
			}
			return nil // (path)[pred]
		}
		return fmt.Errorf("filter expression not allowed in pattern")
	case xpath3.PathExpr:
		// PathExpr: filter/step. Check the filter part is valid.
		return validatePatternPathExpr(e)
	case xpath3.PathStepExpr:
		// E1/E2 or E1//E2 where E2 is a non-axis expression
		return validatePatternPathStepExpr(e)
	case xpath3.VariableExpr:
		// variable references are allowed in XSLT 3.0 patterns
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
	for _, step := range steps {
		// XTSE0340: reverse axes (parent, ancestor, preceding,
		// preceding-sibling, ancestor-or-self) are not allowed in match
		// patterns. Forward axes are allowed in XSLT 3.0.
		switch step.Axis {
		case xpath3.AxisParent, xpath3.AxisAncestor, xpath3.AxisAncestorOrSelf,
			xpath3.AxisPreceding, xpath3.AxisPrecedingSibling:
			return fmt.Errorf("axis %v not allowed in match pattern", step.Axis)
		}
		// Check each step for disallowed constructs in predicates
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
		// variable references are allowed in XSLT 3.0 patterns
	case xpath3.PathStepExpr:
		// Nested path step (e.g., x/(a|b)/text())
		return validatePatternPathStepExpr(f)
	case *xpath3.PathStepExpr:
		return validatePatternPathStepExpr(*f)
	default:
		return fmt.Errorf("expression type %T not allowed at start of path pattern", e.Filter)
	}
	return nil
}

func validatePatternPathStepExpr(e xpath3.PathStepExpr) error {
	// Check left side
	switch l := e.Left.(type) {
	case xpath3.VariableExpr:
		// variable references are allowed in XSLT 3.0 patterns
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
	// Check right side: must be a valid step (axis step / node test), not
	// an arbitrary expression like a numeric literal.
	if !isValidPatternStep(e.Right) {
		return fmt.Errorf("expression %T not allowed as step in pattern", e.Right)
	}
	switch r := e.Right.(type) {
	case xpath3.VariableExpr:
		// variable references are allowed in XSLT 3.0 patterns
	case xpath3.FunctionCall:
		return fmt.Errorf("function call %s() not allowed in middle of pattern", r.Name)
	case xpath3.ArrayConstructorExpr:
		return fmt.Errorf("array constructor not allowed in pattern")
	}
	return nil
}

// isValidPatternStep checks whether an expression is valid as a step in a
// match pattern (i.e., an axis step or node test, not a literal or arbitrary expr).
func isValidPatternStep(expr xpath3.Expr) bool {
	switch expr.(type) {
	case xpath3.LocationPath, *xpath3.LocationPath:
		return true
	case xpath3.FilterExpr:
		return true
	case xpath3.ContextItemExpr:
		return true
	case xpath3.VariableExpr:
		return true
	case xpath3.FunctionCall:
		return true
	case xpath3.UnionExpr:
		return true
	case xpath3.IntersectExceptExpr:
		return true
	}
	return false
}

// isAllowedPatternFunction returns true if a function call is allowed at the
// start of a path pattern. In XSLT 3.0, key(), id(), and doc() are allowed
// with literal arguments.
func isAllowedPatternFunction(fc xpath3.FunctionCall) bool {
	switch fc.Name {
	case "key", "id", "idref", "doc", "element-with-id", "root":
		// Check that all arguments are literals or variable refs
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

// validatePatternFunctions checks that all function calls in a pattern
// reference known functions (built-in XPath or declared xsl:function).
// Unknown functions like f:special() raise XPST0017 at compile time.
func (c *compiler) validatePatternFunctions(p *pattern, source string) error {
	for _, alt := range p.Alternatives {
		var walkErr error
		xpathstream.WalkExpr(alt.expr, func(e xpath3.Expr) bool {
			if walkErr != nil {
				return false
			}
			fc, ok := e.(xpath3.FunctionCall)
			if !ok {
				return true
			}
			// Unprefixed functions are built-in XPath functions — always OK.
			if fc.Prefix == "" {
				return true
			}
			local := fc.Name
			nsURI := c.nsBindings[fc.Prefix]
			displayName := fc.Prefix + ":" + local
			// Known XPath/XSLT function namespaces are always OK.
			switch nsURI {
			case lexicon.NamespaceFn, lexicon.NamespaceMath, lexicon.NamespaceMap, lexicon.NamespaceArray,
				lexicon.NamespaceXSD, lexicon.NamespaceXSLT:
				return true
			}
			// Check if declared as xsl:function in the stylesheet.
			qn := xpath3.QualifiedName{Name: local, URI: nsURI}
			for fk := range c.stylesheet.functions {
				if fk.Name == qn {
					return true
				}
			}
			walkErr = staticError(errCodeXPST0017,
				"unknown function %s() in match pattern %q", displayName, source)
			return false
		})
		if walkErr != nil {
			return walkErr
		}
	}
	return nil
}

// validatePredicateExpr validates expressions used inside predicates.
// Predicates in patterns can contain arbitrary XPath expressions.
func validatePredicateExpr(_ xpath3.Expr) error {
	// Predicates can contain any valid XPath expression,
	// so no additional validation needed beyond function checks
	// (which are handled separately by validatePatternFunctions).
	return nil
}

// checkPatternForbiddenFunctions checks that a pattern AST does not
// contain calls to functions forbidden in patterns per XSLT 3.0.
// XTSE3500: current-merge-key() must not be used within a pattern.
// XTSE3470: current-merge-group() must not be used within a pattern.
// XTSE1060: current-group()/current-grouping-key() must not be used within a pattern.
func checkPatternForbiddenFunctions(ast xpath3.Expr) error {
	var walkErr error
	xpathstream.WalkExpr(ast, func(e xpath3.Expr) bool {
		if walkErr != nil {
			return false
		}
		fc, ok := e.(xpath3.FunctionCall)
		if !ok {
			return true
		}
		if fc.Prefix != "" {
			return true
		}
		switch fc.Name {
		case "current-merge-key":
			walkErr = staticError(errCodeXTSE3500, "current-merge-key() must not be used within a pattern")
			return false
		case "current-merge-group":
			walkErr = staticError(errCodeXTSE3470, "current-merge-group() must not be used within a pattern")
			return false
		case "current-group":
			walkErr = staticError(errCodeXTSE1060, "current-group() must not be used within a pattern")
			return false
		case "current-grouping-key":
			walkErr = staticError(errCodeXTSE1070, "current-grouping-key() must not be used within a pattern")
			return false
		}
		return true
	})
	return walkErr
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
	case xpath3.UnionExpr:
		// Union pattern (a|b): priority is max of operands' priorities.
		lp := computeDefaultPriority(e.Left)
		rp := computeDefaultPriority(e.Right)
		if lp > rp {
			return lp
		}
		return rp
	case xpath3.FilterExpr:
		// Predicate pattern: .[pred1][pred2]...
		// Priority = 0.5 + (number of predicates * 0.25)
		if _, ok := e.Expr.(xpath3.ContextItemExpr); ok {
			return 0.5 + float64(len(e.Predicates))*0.25
		}
		return 0.5
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
	// Per XSLT 3.0 §6.4: path patterns with multiple steps get priority 0.5.
	// An absolute path like "/doc" counts as multi-step (root + step).
	if len(steps) > 1 || (len(steps) == 1 && isAbsolute(expr)) {
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
		if nt.TypeName != "" {
			if nt.Name == "" || nt.Name == "*" {
				return 0 // element(*, type) — type constraint, priority 0
			}
			return 0.25 // element(name, type)
		}
		if nt.Name == "" || nt.Name == "*" {
			return -0.5 // element() or element(*)
		}
		return 0 // element(specific-name)
	case xpath3.AttributeTest:
		if nt.TypeName != "" {
			if nt.Name == "" || nt.Name == "*" {
				return 0 // attribute(*, type) — type constraint, priority 0
			}
			return 0.25 // attribute(name, type)
		}
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
func (p *pattern) matchPattern(ctx *execContext, node helium.Node) bool {
	// Temporarily set xpath-default-namespace from pattern's compile-time value.
	// Also clear contextItem so predicates evaluate with the candidate node
	// as the context item (not an atomic value from an enclosing instruction
	// like xsl:analyze-string). Per XSLT spec, the set of captured substrings
	// is empty when evaluating patterns: regex-group() must return empty
	// sequence inside a pattern, not groups from an enclosing matching-substring.
	saved := ctx.xpathDefaultNS
	savedHas := ctx.hasXPathDefaultNS
	savedGroups := ctx.regexGroups
	savedContext := ctx.contextNode
	savedCurrent := ctx.currentNode
	savedItem := ctx.contextItem
	savedInPattern := ctx.inPatternMatch
	ctx.xpathDefaultNS = p.xpathDefaultNS
	ctx.hasXPathDefaultNS = p.xpathDefaultNS != ""
	ctx.regexGroups = nil
	ctx.contextNode = node
	ctx.currentNode = node
	ctx.contextItem = nil
	ctx.inPatternMatch = true
	defer func() {
		ctx.xpathDefaultNS = saved
		ctx.hasXPathDefaultNS = savedHas
		ctx.regexGroups = savedGroups
		ctx.contextNode = savedContext
		ctx.currentNode = savedCurrent
		ctx.contextItem = savedItem
		ctx.inPatternMatch = savedInPattern
	}()

	for _, alt := range p.Alternatives {
		if matchPatternAlt(ctx, alt, node) {
			return true
		}
	}
	return false
}

// matchPatternItem tests whether an arbitrary item (node or atomic value)
// matches the pattern. For node items it delegates to matchPattern; for atomic
// values it evaluates the pattern expression with the item as context and
// checks whether it produces a non-empty result. This is needed for
// group-starting-with / group-ending-with over non-node sequences (XSLT 3.0).
func (p *pattern) matchPatternItem(ctx *execContext, item xpath3.Item) bool {
	if ni, ok := item.(xpath3.NodeItem); ok {
		return p.matchPattern(ctx, ni.Node)
	}

	// Atomic item: evaluate each alternative's expression with the item as
	// context. A FilterExpr like .[pred] returns the item when pred is true.
	saved := ctx.xpathDefaultNS
	savedHas := ctx.hasXPathDefaultNS
	savedGroups := ctx.regexGroups
	savedItem := ctx.contextItem
	savedInPattern := ctx.inPatternMatch
	ctx.xpathDefaultNS = p.xpathDefaultNS
	ctx.hasXPathDefaultNS = p.xpathDefaultNS != ""
	ctx.regexGroups = nil
	ctx.contextItem = item
	ctx.inPatternMatch = true
	defer func() {
		ctx.xpathDefaultNS = saved
		ctx.hasXPathDefaultNS = savedHas
		ctx.regexGroups = savedGroups
		ctx.contextItem = savedItem
		ctx.inPatternMatch = savedInPattern
	}()

	for _, alt := range p.Alternatives {
		// variable reference patterns (e.g., match="$var") only match nodes,
		// never atomic items. Per XSLT 3.0 §5.5.3, the semantics require
		// root(N)//(V) which is undefined for non-node items.
		if _, isVar := alt.expr.(xpath3.VariableExpr); isVar {
			continue
		}
		compiled, compErr := xpath3.NewCompiler().CompileExpr(alt.expr)
		if compErr != nil {
			continue
		}
		result, err := ctx.evalXPath(compiled, nil)
		if err != nil {
			continue
		}
		seq := result.Sequence()
		if seq != nil && sequence.Len(seq) > 0 {
			return true
		}
	}
	return false
}

// matchesAttributes returns true if the pattern source could potentially match
// attribute nodes. This is a conservative heuristic used by key table building
// to decide whether to visit attribute nodes during the document walk.
func (p *pattern) matchesAttributes() bool {
	return strings.Contains(p.source, "@") || strings.Contains(p.source, "attribute")
}

// matchPatternAlt tests whether a node matches a single pattern alternative.
// XSLT patterns evaluate by checking if the node would be selected by the
// equivalent XPath expression when evaluated in the right context.
func matchPatternAlt(ctx *execContext, alt *patternAlt, node helium.Node) bool {
	if alt.neverMatches {
		return false
	}
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
	case xpath3.FilterExpr:
		// FilterExpr with root inner: (/)[pred] — matches document node
		// when predicate is true.
		if lp, ok := e.Expr.(xpath3.LocationPath); ok && lp.Absolute && len(lp.Steps) == 0 {
			if node.Type() != helium.DocumentNode {
				return false
			}
			return matchByEvaluation(ctx, alt, node)
		}
		return matchByEvaluation(ctx, alt, node)
	case xpath3.UnionExpr:
		// Union pattern: (a|b) matches if node matches either operand.
		leftAlt := &patternAlt{expr: e.Left}
		rightAlt := &patternAlt{expr: e.Right}
		return matchPatternAlt(ctx, leftAlt, node) || matchPatternAlt(ctx, rightAlt, node)
	case xpath3.IntersectExceptExpr:
		// intersect/except patterns require evaluation-based matching to
		// get correct set semantics. Both sides must be evaluated from the
		// same context; independent matching gives wrong results for cases
		// like "a//* intersect b//*" where a and b are nested.
		return matchByEvaluation(ctx, alt, node)
	case xpath3.PathStepExpr:
		// Path step pattern: E1/E2 or E1//E2 where E2 is a non-axis step.
		// Match bottom-up: check if the candidate matches the right part,
		// then verify the left part matches an ancestor.
		return matchPathStepPattern(ctx, e, node)
	case xpath3.PathExpr:
		// PathExpr: filter/steps. Try bottom-up matching first; fall back
		// to evaluation-based matching for complex cases (key()//union, etc.).
		if matchPathExprPattern(ctx, e, node) {
			return true
		}
		return matchByEvaluation(ctx, alt, node)
	case xpath3.VariableExpr:
		// $var pattern: matches if node is in the variable's value.
		return matchByEvaluation(ctx, alt, node)
	default:
		// For complex expressions, try evaluating from document root
		// and checking if node is in the result set.
		return matchByEvaluation(ctx, alt, node)
	}
}

// matchPathExprPattern matches a PathExpr pattern (Filter/Path) bottom-up.
// First checks if the node matches the path's last step, then works upward
// through the remaining steps, and finally checks the filter against an ancestor.
func matchPathExprPattern(ctx *execContext, e xpath3.PathExpr, node helium.Node) bool {
	if e.Path == nil || len(e.Path.Steps) == 0 {
		// No path steps: just check the filter
		filterAlt := &patternAlt{expr: e.Filter}
		return matchPatternAlt(ctx, filterAlt, node)
	}

	// Match the path steps bottom-up (like matchLocationPath)
	lastStep := e.Path.Steps[len(e.Path.Steps)-1]
	if !nodeMatchesStep(ctx, lastStep, node) {
		return false
	}

	// If there's only one step, check the filter against the parent
	if len(e.Path.Steps) == 1 {
		filterAlt := &patternAlt{expr: e.Filter}
		parent := node.Parent()
		if parent != nil && matchPatternAlt(ctx, filterAlt, parent) {
			return true
		}
		return false
	}

	// Multiple steps: match remaining steps upward, then check filter
	remaining := e.Path.Steps[:len(e.Path.Steps)-1]
	parent := node.Parent()
	if parent == nil {
		return false
	}
	if !matchStepsUpward(ctx, remaining, false, parent) {
		return false
	}
	// Walk up to find where the remaining steps matched, then check filter
	filterAlt := &patternAlt{expr: e.Filter}
	// The filter needs to match the ancestor above the remaining steps
	cur := parent
	for i := len(remaining) - 1; i >= 0; i-- {
		cur = cur.Parent()
		if cur == nil {
			return false
		}
	}
	return matchPatternAlt(ctx, filterAlt, cur)
}

// matchPathStepPattern matches a PathStepExpr pattern (E1/E2 or E1//E2) bottom-up.
// First checks if the candidate node matches the right part (E2), then verifies
// that an ancestor matches the left part (E1).
func matchPathStepPattern(ctx *execContext, e xpath3.PathStepExpr, node helium.Node) bool {
	// Check if node matches the right part.
	rightAlt := &patternAlt{expr: e.Right}
	if !matchPatternAlt(ctx, rightAlt, node) {
		return false
	}
	// Determine if the right part uses descendant-axis steps.
	// When the right part (or any union branch) uses descendant::,
	// the left part may match ANY ancestor, not just the parent.
	checkAllAncestors := e.DescOrSelf || rightPartUsesDescendantAxis(e.Right)
	// Check ancestors for the left part.
	leftAlt := &patternAlt{expr: e.Left}
	if checkAllAncestors {
		for ancestor := node.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
			if matchPatternAlt(ctx, leftAlt, ancestor) {
				return true
			}
		}
	} else {
		// E1/E2 with child axis: check immediate parent.
		parent := node.Parent()
		if parent != nil && matchPatternAlt(ctx, leftAlt, parent) {
			return true
		}
	}
	return false
}

// rightPartUsesDescendantAxis checks if an expression (typically the right
// side of a PathStepExpr) contains any descendant:: or descendant-or-self::
// axis steps. This determines whether ancestor matching should walk all the
// way up the tree.
func rightPartUsesDescendantAxis(expr xpath3.Expr) bool {
	found := false
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		switch lp := e.(type) {
		case xpath3.LocationPath:
			for _, step := range lp.Steps {
				if step.Axis == xpath3.AxisDescendant || step.Axis == xpath3.AxisDescendantOrSelf {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

// matchLocationPath matches a LocationPath pattern against a node.
// Patterns match bottom-up: starting from the node, check if there's
// a path from the root that would select this node.
func matchLocationPath(ctx *execContext, path xpath3.LocationPath, node helium.Node) bool {
	if len(path.Steps) == 0 {
		// "/" matches document nodes
		return path.Absolute && node.Type() == helium.DocumentNode
	}

	// Check the last step against the node.
	// For multi-step patterns with descendant axis and predicates, skip
	// predicate evaluation here — predicates will be evaluated with proper
	// descendant-set position relative to the ancestor later.
	lastStep := path.Steps[len(path.Steps)-1]
	hasDescPreds := len(path.Steps) > 1 &&
		(lastStep.Axis == xpath3.AxisDescendant || lastStep.Axis == xpath3.AxisDescendantOrSelf) &&
		len(lastStep.Predicates) > 0
	if hasDescPreds {
		// Check name/type test without predicates
		stepNoPreds := lastStep
		stepNoPreds.Predicates = nil
		if !nodeMatchesStep(ctx, stepNoPreds, node) {
			return false
		}
	} else if !nodeMatchesStep(ctx, lastStep, node) {
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

	// Match remaining steps upward.
	// The axis of the last step determines how to walk to the preceding step.
	remaining := path.Steps[:len(path.Steps)-1]
	if lastStep.Axis == xpath3.AxisDescendant || lastStep.Axis == xpath3.AxisDescendantOrSelf {
		if len(lastStep.Predicates) > 0 {
			// Position predicates on descendant axis are relative to the full
			// descendant set of the ancestor node. Walk up to find ancestors
			// that match the preceding steps, then evaluate predicates in that context.
			checkDescPreds := func(cur helium.Node, includeSelf bool) bool {
				if !matchStepsUpward(ctx, remaining, path.Absolute, cur) {
					return false
				}
				// Collect all descendants of cur that match the name test.
				// For descendant-or-self, include cur itself if it matches.
				var descendants []helium.Node
				if includeSelf && nodeMatchesTest(ctx, lastStep.NodeTest, cur) {
					descendants = append(descendants, cur)
				}
				descendants = append(descendants, collectDescendants(ctx, lastStep.NodeTest, cur)...)
				// Find position of node in the descendant set
				pos := 0
				for j, d := range descendants {
					if d == node {
						pos = j + 1
						break
					}
				}
				if pos == 0 {
					return false
				}
				// Evaluate predicates with position context
				for _, pred := range lastStep.Predicates {
					if !evaluatePredicateWithPosition(ctx, pred, node, pos, len(descendants)) {
						return false
					}
				}
				return true
			}
			isSelfAxis := lastStep.Axis == xpath3.AxisDescendantOrSelf
			// For descendant-or-self, start from the node itself (it IS a
			// descendant-or-self of itself) and walk upward through ancestors.
			// For descendant (without self), start from parent.
			startNode := node.Parent()
			if isSelfAxis {
				startNode = node
			}
			for cur := startNode; cur != nil; cur = cur.Parent() {
				if checkDescPreds(cur, isSelfAxis) {
					return true
				}
			}
			// Child-or-top: parentless nodes try matching remaining steps
			// from the node itself (XSLT 3.0 §19.2).
			if node.Parent() == nil && node.Type() != helium.DocumentNode {
				if isSelfAxis && !path.Absolute {
					if checkDescPreds(node, true) {
						return true
					}
				}
			}
			return false
		}
		// descendant / descendant-or-self axis: any ancestor may contain the preceding step
		for cur := node.Parent(); cur != nil; cur = cur.Parent() {
			if matchStepsUpward(ctx, remaining, path.Absolute, cur) {
				return true
			}
		}
		// Child-or-top: parentless non-document nodes try matching
		// remaining steps from the node itself.
		if node.Parent() == nil && node.Type() != helium.DocumentNode && !path.Absolute {
			if matchStepsUpward(ctx, remaining, false, node) {
				return true
			}
		}
		return false
	}
	if matchStepsUpward(ctx, remaining, path.Absolute, node.Parent()) {
		return true
	}
	return false
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
	nodeType := node.Type()

	// Document nodes are matched by document-node() patterns, or by node()
	// on the self axis (as in match=".").  On the child/descendant axes,
	// document nodes are never selected.
	if nodeType == helium.DocumentNode {
		if _, isDocTest := step.NodeTest.(xpath3.DocumentTest); !isDocTest {
			// Allow self::node() to match document nodes.
			if step.Axis != xpath3.AxisSelf {
				return false
			}
			if tt, ok := step.NodeTest.(xpath3.TypeTest); !ok || tt.Kind != xpath3.NodeKindNode {
				return false
			}
		}
	}
	// Attribute nodes are only matched by the attribute axis (e.g., @*, attribute::node()).
	// The default child axis does not include attributes.
	// Conversely, the attribute axis only matches attribute nodes.
	isAttr := nodeType == helium.AttributeNode
	if isAttr != (step.Axis == xpath3.AxisAttribute) {
		return false
	}
	// Namespace nodes are only matched by the namespace axis or namespace-node() test.
	isNS := nodeType == helium.NamespaceNode
	if isNS != (step.Axis == xpath3.AxisNamespace) {
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
	case xpath3.NamespaceNodeTest:
		return node.Type() == helium.NamespaceNode
	case xpath3.SchemaElementTest:
		return matchSchemaElementTest(ctx, nt, node)
	case xpath3.SchemaAttributeTest:
		return matchSchemaAttributeTest(ctx, nt, node)
	default:
		return false
	}
}

// matchSchemaElementTest checks if a node matches a schema-element(name) test.
func matchSchemaElementTest(ctx *execContext, t xpath3.SchemaElementTest, node helium.Node) bool {
	if node.Type() != helium.ElementNode {
		return false
	}
	elem := node.(*helium.Element)
	// Resolve the schema element name.
	prefix, local := splitQNamePair(t.Name)
	ns := ""
	if prefix != "" {
		ns = ctx.stylesheet.namespaces[prefix]
	} else if ctx.hasXPathDefaultNS {
		ns = ctx.xpathDefaultNS
	}
	// Check name match (direct or substitution group).
	nameMatch := elem.LocalName() == local && elem.URI() == ns
	if !nameMatch && ctx.schemaRegistry != nil {
		nameMatch = ctx.schemaRegistry.IsSubstitutionGroupMember(elem.LocalName(), elem.URI(), local, ns)
	}
	if !nameMatch {
		return false
	}
	// Check schema declaration exists.
	if ctx.schemaRegistry == nil {
		return false
	}
	declType, found := ctx.schemaRegistry.LookupElement(local, ns)
	if !found {
		return false
	}
	// Check type annotation.
	ann := ""
	if ctx.typeAnnotations != nil {
		ann = ctx.typeAnnotations[node]
	}
	if ann == "" || ann == "xs:untyped" {
		return false // untyped elements have not been validated — no match
	}
	if ann == declType || ctx.schemaRegistry.IsSubtypeOf(ann, declType) {
		return true
	}
	return false
}

// matchSchemaAttributeTest checks if a node matches a schema-attribute(name) test.
func matchSchemaAttributeTest(ctx *execContext, t xpath3.SchemaAttributeTest, node helium.Node) bool {
	if node.Type() != helium.AttributeNode {
		return false
	}
	attr := node.(*helium.Attribute)
	prefix, local := splitQNamePair(t.Name)
	ns := ""
	if prefix != "" {
		ns = ctx.stylesheet.namespaces[prefix]
	}
	if attr.LocalName() != local || attr.URI() != ns {
		return false
	}
	if ctx.schemaRegistry == nil {
		return false
	}
	_, found := ctx.schemaRegistry.LookupAttribute(local, ns)
	return found
}

// splitQNamePair splits "prefix:local" into (prefix, local) or ("", name).
func splitQNamePair(name string) (string, string) {
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		return name[:idx], name[idx+1:]
	}
	return "", name
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
	var nodeLocal, nodeURI string
	var isElem bool
	switch v := node.(type) {
	case *helium.Element:
		isElem = true
		nodeLocal = v.LocalName()
		nodeURI = v.URI()
	case *helium.Attribute:
		nodeLocal = v.LocalName()
		nodeURI = v.URI()
	case *helium.NamespaceNodeWrapper:
		nodeLocal = v.Name()
		nodeURI = ""
	default:
		return false
	}

	if nt.Local == "*" && nt.Prefix == "" && nt.URI == "" {
		// * matches all elements/attributes/namespace-nodes
		return true
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
	} else if nt.Prefix == "" && ctx != nil && isElem {
		// Unprefixed element names use xpath-default-namespace
		expectedURI = ctx.xpathDefaultNS
	}
	if nodeLocal != nt.Local || nodeURI != expectedURI {
		return false
	}
	// XSLT 3.0 §6.6: when mode typed="strict", bare element name patterns
	// are treated as schema-element() — the element must have been validated
	// against the schema declaration for that name.
	if isElem && ctx != nil && nt.Prefix != "*" && isTypedStrictMode(ctx) {
		if ctx.schemaRegistry == nil {
			return false
		}
		declType, found := ctx.schemaRegistry.LookupElement(nodeLocal, nodeURI)
		if !found {
			return false
		}
		ann := ""
		if ctx.typeAnnotations != nil {
			ann = ctx.typeAnnotations[node]
		}
		if ann == "" || ann == "xs:untyped" {
			return false
		}
		if ann != declType && !ctx.schemaRegistry.IsSubtypeOf(ann, declType) {
			return false
		}
	}
	return true
}

// isTypedStrictMode returns true if the current mode has typed="strict" or typed="yes".
func isTypedStrictMode(ctx *execContext) bool {
	mode := ctx.currentMode
	if md := ctx.stylesheet.modeDefs[mode]; md != nil {
		return md.Typed == validationStrict || md.Typed == lexicon.ValueYes || md.Typed == "true" || md.Typed == "1"
	}
	if mode == "" {
		if md := ctx.stylesheet.modeDefs[modeDefault]; md != nil {
			return md.Typed == validationStrict || md.Typed == lexicon.ValueYes || md.Typed == "true" || md.Typed == "1"
		}
	}
	return false
}

// matchElementTest checks if a node matches an element() test.
// isBuiltinXSDType returns true for type names that are built-in XSD types
// (e.g., "string", "integer", "decimal", "untyped", "untypedAtomic").
func isBuiltinXSDType(name string) bool {
	switch name {
	case "string", "boolean", "decimal", "float", "double",
		"duration", "dateTime", "time", "date",
		"gYearMonth", "gYear", "gMonthDay", "gDay", "gMonth",
		"hexBinary", "base64Binary", "anyURI", "QName", "NOTATION",
		"normalizedString", "token", "language", "NMTOKEN", "NMTOKENS",
		"Name", "NCName", "ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES",
		"integer", "nonPositiveInteger", "negativeInteger", "long", "int",
		"short", "byte", "nonNegativeInteger", "unsignedLong", "unsignedInt",
		"unsignedShort", "unsignedByte", "positiveInteger",
		"yearMonthDuration", "dayTimeDuration", "dateTimeStamp",
		"untyped", "untypedAtomic", "anyType", "anySimpleType", "anyAtomicType":
		return true
	}
	return false
}

func matchElementTest(ctx *execContext, et xpath3.ElementTest, node helium.Node) bool {
	if node.Type() != helium.ElementNode {
		return false
	}
	if et.Name == "" || et.Name == "*" {
		if et.TypeName != "" {
			return matchTypeAnnotation(ctx, node, et.TypeName)
		}
		return true
	}
	elem, ok := node.(*helium.Element)
	if !ok {
		return false
	}
	// Check local name match (may contain prefix:local or Q{uri}local)
	name := et.Name
	nameMatch := false
	if strings.HasPrefix(name, "Q{") {
		closeIdx := strings.IndexByte(name, '}')
		if closeIdx > 0 {
			uri := name[2:closeIdx]
			local := name[closeIdx+1:]
			nameMatch = elem.LocalName() == local && elem.URI() == uri
		}
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		uri := ""
		if ctx != nil {
			uri = ctx.resolvePrefix(prefix)
		}
		nameMatch = elem.LocalName() == local && elem.URI() == uri
	} else {
		nameMatch = elem.LocalName() == name
	}
	if !nameMatch {
		return false
	}
	// Check type annotation if specified
	if et.TypeName != "" {
		if !matchTypeAnnotation(ctx, node, et.TypeName) {
			return false
		}
		// element(name, type) without ? does not match nilled elements;
		// element(name, type?) with ? matches even when nilled.
		if !et.Nillable && ctx != nil && ctx.isNilled(elem) {
			return false
		}
		return true
	}
	return true
}

// matchAttributeTest checks if a node matches an attribute() test.
func matchAttributeTest(ctx *execContext, at xpath3.AttributeTest, node helium.Node) bool {
	if node.Type() != helium.AttributeNode {
		return false
	}
	if at.Name == "" || at.Name == "*" {
		if at.TypeName != "" {
			return matchTypeAnnotation(ctx, node, at.TypeName)
		}
		return true
	}
	attr, ok := node.(*helium.Attribute)
	if !ok {
		return false
	}
	// Get the actual local name of the attribute, stripping any prefix
	attrLocal := attr.LocalName()
	if idx := strings.IndexByte(attrLocal, ':'); idx >= 0 {
		attrLocal = attrLocal[idx+1:]
	}
	name := at.Name
	nameMatch := false
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		uri := ""
		if ctx != nil {
			uri = ctx.resolvePrefix(prefix)
		}
		nameMatch = attrLocal == local && attr.URI() == uri
	} else if strings.HasPrefix(name, "Q{") {
		// URIQualifiedName: Q{uri}local
		closeIdx := strings.IndexByte(name, '}')
		if closeIdx >= 0 {
			uri := name[2:closeIdx]
			local := name[closeIdx+1:]
			nameMatch = attrLocal == local && attr.URI() == uri
		}
	} else {
		// Bare name (no prefix): matches only attributes in no namespace.
		nameMatch = attrLocal == name && attr.URI() == ""
	}
	if !nameMatch {
		return false
	}
	// Check type annotation if specified
	if at.TypeName != "" {
		return matchTypeAnnotation(ctx, node, at.TypeName)
	}
	return true
}

// matchTypeAnnotation checks if a node's type annotation matches the given type name.
func matchTypeAnnotation(ctx *execContext, node helium.Node, typeName string) bool {
	// Normalize type name: strip "xs:" prefix if present
	normalize := func(s string) string {
		for _, prefix := range []string{"xs:", "xsd:", "http://www.w3.org/2001/XMLSchema:"} {
			if strings.HasPrefix(s, prefix) {
				return s[len(prefix):]
			}
		}
		return s
	}

	// Resolve prefixed type name to Q{uri}local form
	resolveType := func(s string) string {
		if strings.HasPrefix(s, "Q{") {
			return s
		}
		if idx := strings.IndexByte(s, ':'); idx >= 0 {
			prefix := s[:idx]
			local := s[idx+1:]
			if ctx != nil {
				uri := ctx.resolvePrefix(prefix)
				if uri != "" {
					return "Q{" + uri + "}" + local
				}
			}
		} else if ctx != nil && ctx.xpathDefaultNS != "" {
			// Unprefixed type name: resolve via xpath-default-namespace
			return "Q{" + ctx.xpathDefaultNS + "}" + s
		} else if !isBuiltinXSDType(s) {
			// Unprefixed non-XSD type with no namespace: use Q{} form
			return "Q{}" + s
		}
		return s
	}

	var ann string
	if ctx != nil && ctx.typeAnnotations != nil {
		ann = ctx.typeAnnotations[node]
	}
	if ann == "" {
		switch node.Type() {
		case helium.ElementNode:
			ann = "untyped"
		case helium.AttributeNode:
			ann = "untypedAtomic"
		default:
			return false
		}
	}

	// Try direct comparison after resolving prefixes
	resolvedAnn := resolveType(ann)
	resolvedType := resolveType(typeName)
	if resolvedAnn == resolvedType {
		return true
	}

	normAnn := normalize(ann)
	normType := normalize(typeName)
	if normAnn == normType {
		return true
	}
	// Check subtype relationship using built-in XSD type hierarchy.
	fullAnn := ann
	if !strings.HasPrefix(fullAnn, "xs:") && !strings.HasPrefix(fullAnn, "Q{") {
		fullAnn = "xs:" + fullAnn
	}
	fullType := typeName
	if !strings.HasPrefix(fullType, "xs:") && !strings.HasPrefix(fullType, "Q{") {
		fullType = "xs:" + fullType
	}
	if xpath3.BuiltinIsSubtypeOf(fullAnn, fullType) {
		return true
	}
	// Check via schema declarations for user-defined types.
	if ctx != nil && ctx.schemaRegistry != nil {
		// Also try with resolved QName forms
		if ctx.schemaRegistry.IsSubtypeOf(resolvedAnn, resolvedType) {
			return true
		}
		return ctx.schemaRegistry.IsSubtypeOf(fullAnn, fullType)
	}
	return false
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
	for child := range helium.Children(doc) {
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

// collectDescendants collects all descendant nodes of root that match the
// given node test, in document order.
func collectDescendants(ctx *execContext, test xpath3.NodeTest, root helium.Node) []helium.Node {
	var result []helium.Node
	_ = helium.Walk(root, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n == root {
			return nil // skip the root itself
		}
		if nodeMatchesTest(ctx, test, n) {
			result = append(result, n)
		}
		return nil
	}))
	return result
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

	// For attribute nodes, iterate the element's attributes instead of children
	if node.Type() == helium.AttributeNode {
		if elem, ok := parent.(*helium.Element); ok {
			for _, attr := range elem.Attributes() {
				if nodeMatchesTest(ctx, test, attr) {
					siblings = append(siblings, attr)
				}
			}
			return siblings
		}
	}

	// Iterate through all children of the parent
	for child := range helium.Children(parent) {
		if nodeMatchesTest(ctx, test, child) {
			siblings = append(siblings, child)
		}
	}
	return siblings
}

// evaluatePredicateWithPosition evaluates a pattern predicate with explicit
// position and size context.
func evaluatePredicateWithPosition(ec *execContext, pred xpath3.Expr, node helium.Node, pos, size int) bool {
	compiled, compErr := xpath3.NewCompiler().CompileExpr(pred)
	if compErr != nil {
		return false
	}
	eval := ec.xpathEvaluator().
		Position(pos).
		Size(size)
	result, err := eval.Evaluate(ec.xpathContext(), compiled, node)
	if err != nil {
		// Propagate fatal errors (like XTDE0640 circular key or
		// ErrCircularRef from variable/param evaluation) through
		// the exec context so they can be raised after pattern matching.
		if isXSLTError(err, errCodeXTDE0640) {
			ec.patternMatchErr = err
		} else if errors.Is(err, ErrCircularRef) {
			ec.patternMatchErr = dynamicError(errCodeXTDE0640, "%s", err.Error())
		}
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

// matchByEvaluation matches complex patterns (e.g., id(), key(), doc()) by
// evaluating the pattern expression and checking whether the candidate node
// appears in the result sequence. It tries the candidate node as context first,
// then walks up the ancestor chain trying each ancestor as context.
func matchByEvaluation(ctx *execContext, alt *patternAlt, node helium.Node) bool {
	compiled := alt.compiled
	if compiled == nil {
		var compErr error
		compiled, compErr = xpath3.NewCompiler().CompileExpr(alt.expr)
		if compErr != nil {
			return false
		}
	}

	// Try evaluating with the node itself as context first.
	// Needed for document-node patterns like (/)[doc].
	result, err := ctx.evalXPath(compiled, node)
	if err != nil {
		// Propagate fatal errors (like XTDE0640 circular key) through
		// the exec context so they can be raised after pattern matching.
		if isXSLTError(err, errCodeXTDE0640) {
			ctx.patternMatchErr = err
			return false
		}
	}
	if err == nil {
		for item := range sequence.Items(result.Sequence()) {
			if ni, ok := item.(xpath3.NodeItem); ok && ni.Node == node {
				return true
			}
		}
	}
	// Then try evaluating from each ancestor up to the document root.
	for ancestor := node.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
		result, err := ctx.evalXPath(compiled, ancestor)
		if err != nil {
			if isXSLTError(err, errCodeXTDE0640) {
				ctx.patternMatchErr = err
				return false
			}
			continue
		}
		for item := range sequence.Items(result.Sequence()) {
			if ni, ok := item.(xpath3.NodeItem); ok && ni.Node == node {
				return true
			}
		}
	}
	return false
}
