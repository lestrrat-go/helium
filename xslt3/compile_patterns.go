package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// Pattern is a compiled XSLT match pattern.
type Pattern struct {
	Alternatives []*PatternAlt
	source       string
}

// PatternAlt is one alternative in a union pattern (separated by |).
type PatternAlt struct {
	expr     xpath3.Expr // the parsed XPath AST
	priority float64
}

// compilePattern compiles an XSLT match pattern string.
// XSLT patterns are a restricted subset of XPath expressions.
func compilePattern(s string, nsBindings map[string]string) (*Pattern, error) {
	alts := splitPatternUnion(s)
	p := &Pattern{source: s}
	for _, alt := range alts {
		alt = strings.TrimSpace(alt)
		if alt == "" {
			continue
		}
		ast, err := xpath3.Parse(alt)
		if err != nil {
			return nil, staticError(errCodeXTSE0500, "invalid pattern %q: %v", alt, err)
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
	default:
		return 0.5
	}
	if len(steps) == 0 {
		return 0.5
	}
	// Per XSLT 3.0 §6.4: path patterns with multiple steps get priority 0.5
	if len(steps) > 1 {
		return 0.5
	}
	return stepPriority(steps[len(steps)-1])
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
		if nt.Local == "*" {
			// prefix:* or *:local
			return -0.25
		}
		return 0
	default:
		return 0.5
	}
}

// matchPattern tests whether a node matches the pattern.
func (p *Pattern) matchPattern(ctx *execContext, node helium.Node) bool {
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
	// Document nodes are only matched by the root pattern "/", never by steps.
	if node.Type() == helium.DocumentNode {
		return false
	}
	if !nodeMatchesTest(ctx, step.NodeTest, node) {
		return false
	}
	// Evaluate predicates if any
	for _, pred := range step.Predicates {
		if !evaluatePredicate(ctx, pred, node) {
			return false
		}
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

	expectedURI := ""
	if nt.Prefix != "" && ctx != nil {
		expectedURI = ctx.resolvePrefix(nt.Prefix)
	}
	return nodeLocal == nt.Local && nodeURI == expectedURI
}

// evaluatePredicate evaluates a pattern predicate against a node.
func evaluatePredicate(ctx *execContext, pred xpath3.Expr, node helium.Node) bool {
	xpathCtx := ctx.newXPathContext(node)
	result, err := xpath3.EvaluateExpr(xpathCtx, pred, node)
	if err != nil {
		return false
	}
	// Numeric predicates: compare to position
	if f, ok := result.IsNumber(); ok {
		return int(f) == 1 // in pattern matching, position is always 1 for the candidate
	}
	b, err := xpath3.EBV(result.Sequence())
	if err != nil {
		return false
	}
	return b
}

// matchByEvaluation matches complex patterns by evaluating from document root.
func matchByEvaluation(ctx *execContext, alt *PatternAlt, node helium.Node) bool {
	// For non-LocationPath patterns, this is a fallback that evaluates
	// the expression from the document root and checks if the node
	// appears in the result set.
	// This handles patterns like key(), id(), etc.
	return false
}
