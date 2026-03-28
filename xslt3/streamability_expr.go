package xslt3

import (
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
)

// checkStreamableExpr checks a single XPath expression for non-streamable patterns.
func checkStreamableExpr(ss *Stylesheet, expr *xpath3.Expression) error {
	if expr == nil {
		return nil
	}

	// Parent/ancestor axes are motionless — ancestors of the streaming context
	// are always available. However, preceding/preceding-sibling axes require
	// backward access to already-consumed nodes, which is non-streamable.
	if xpathstream.ExprUsesPrecedingAxis(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q uses preceding/preceding-sibling axis, which is not streamable", expr.String())
	}

	// 1. last() outside grounding context (non-motionless in streaming)
	if exprUsesLastOutsideGrounding(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q uses last(), which is not streamable", expr.String())
	}

	// 3. path() function — not streamable (outside grounding context)
	if exprUsesFunctionOutsideGrounding(expr, "path") {
		return staticError(errCodeXTSE3430,
			"expression %q uses path(), which is not streamable", expr.String())
	}

	// 4. Non-motionless predicates on streaming steps (not on grounded data)
	if exprHasNonMotionlessStreamingPredicate(ss, expr) {
		return staticError(errCodeXTSE3430,
			"expression %q has a non-motionless predicate, which is not streamable", expr.String())
	}

	// 5. Multiple downward selections (consuming the stream twice)
	// Only count selections outside grounding functions (snapshot/copy-of).
	if countStreamingDownwardSelections(ss, expr.AST()) > 1 {
		return staticError(errCodeXTSE3430,
			"expression %q has multiple downward selections, which is not streamable", expr.String())
	}

	// 6. Grounding functions (reverse, innermost) with crawling operands are not
	// streamable because the crawling expression itself requires multi-level descent.
	if exprHasCrawlingGroundingArg(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q uses a grounding function with a crawling operand, which is not streamable", expr.String())
	}

	// 7. Higher-order functions (filter, for-each, fold-left, fold-right) require
	// their sequence argument to be grounded — a consuming (downward) expression
	// is not streamable as the first argument.
	if exprHasHigherOrderWithConsumingArg(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q passes a consuming expression to a higher-order function, which is not streamable", expr.String())
	}

	// 8. accumulator-after() on a parent/ancestor axis is not streamable because
	// the ancestor's post-descent value is not yet available during streaming.
	if exprUsesAccumulatorAfterOnAncestor(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q uses accumulator-after on an ancestor, which is not streamable", expr.String())
	}

	// 9. Calls to shallow-descent functions with a climbing (non-striding) first argument
	// are not streamable. The argument must be in striding posture (., child, etc.),
	// not climbing posture (.., parent::, ancestor::).
	if exprHasShallowDescentCallWithClimbingArg(ss, expr) {
		return staticError(errCodeXTSE3430,
			"expression %q calls a shallow-descent function with a climbing argument, which is not streamable", expr.String())
	}

	// 10. Mixed-posture sequence expressions: a SequenceExpr, UnionExpr, or
	// ArrayConstructorExpr used as the LHS of a path step where some items
	// access the streaming source (crawling/striding) and others are grounded
	// (variable refs, literals) has mixed posture and is not streamable.
	if exprHasMixedPostureSequenceInPath(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q has a mixed-posture sequence expression in a path, which is not streamable", expr.String())
	}

	// 11. Union expressions used as the LHS of a path step: a union of two
	// streaming (non-grounded) expressions is crawling per XSLT 3.0 spec
	// (section 19.8.8.2). Stepping from a crawling expression is not
	// streamable.
	if exprHasUnionInPathStep(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q has a union used as the input of a path step, which is not streamable (union of streaming expressions is crawling)", expr.String())
	}

	// 12. treat as document-node(element(X)) in a path: the type check
	// requires inspecting the document element (consuming), and subsequent
	// path steps also consume, giving multiple consuming operations.
	if exprHasConsumingTreatAsInPath(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q uses treat as document-node(element(...)) in a path, which is not streamable", expr.String())
	}

	// 13. Up-then-down navigation in a single expression: a path that first
	// navigates upward (parent/ancestor) and then downward (child/descendant)
	// is not streamable because the downward step from an ancestor may access
	// nodes that have not yet been streamed.
	if exprHasUpThenDownNavigation(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q navigates upward then downward, which is not streamable", expr.String())
	}

	return nil
}

// exprHasCrawlingGroundingArg returns true if the expression contains a call to
// reverse(), innermost(), or outermost() whose argument has a crawling expression
// on streaming (non-grounded) data.
// exprHasShallowDescentCallWithClimbingArg returns true if the expression
// contains a call to a shallow-descent function where the first argument
// uses upward navigation (parent/ancestor), which is not streamable.
func exprHasShallowDescentCallWithClimbingArg(ss *Stylesheet, expr *xpath3.Expression) bool {
	if expr == nil || ss == nil {
		return false
	}
	found := false
	xpathstream.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if fc, ok := e.(xpath3.FunctionCall); ok && fc.Prefix != "" && len(fc.Args) > 0 {
			cat := lookupFuncStreamability(ss, fc.Name, len(fc.Args))
			if cat == lexicon.StreamShallowDescent {
				if exprHasUpwardAxis(fc.Args[0]) {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

// exprHasMixedPostureSequenceInPath returns true if the expression contains a
// PathExpr or PathStepExpr whose LHS/filter is a SequenceExpr, UnionExpr, or
// ArrayConstructorExpr that mixes items accessing the streaming source
// (crawling/striding) with grounded items (variable references, literals).
// Such mixed-posture expressions are not streamable per XSLT 3.0 spec.
func exprHasMixedPostureSequenceInPath(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	xpathstream.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		e = derefXPathExpr(e)
		switch v := e.(type) {
		case xpath3.PathExpr:
			if v.Path != nil && filterHasMixedPosture(v.Filter) {
				found = true
				return false
			}
		case xpath3.PathStepExpr:
			if filterHasMixedPosture(v.Left) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// filterHasMixedPosture checks whether a filter expression (LHS of a path)
// is a SequenceExpr, UnionExpr, or ArrayConstructorExpr with mixed postures
// (some items grounded, some crawling).
func filterHasMixedPosture(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch v := expr.(type) {
	case xpath3.SequenceExpr:
		return seqHasMixedPosture(v)
	case xpath3.UnionExpr:
		return itemsHaveMixedPosture([]xpath3.Expr{v.Left, v.Right})
	case xpath3.ArrayConstructorExpr:
		return itemsHaveMixedPosture(v.Items)
	case xpath3.LookupExpr:
		// Array lookup like [$a, //B]?* — check the base expression.
		return filterHasMixedPosture(v.Expr)
	}
	return false
}

// exprHasUnionInPathStep returns true if a UnionExpr where both operands
// access the streaming source (non-grounded) is used as the LHS of a path
// step. Per XSLT 3.0, a union of two streaming expressions is crawling, and
// stepping from a crawling expression is not streamable.
func exprHasUnionInPathStep(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	xpathstream.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		e = derefXPathExpr(e)
		switch v := e.(type) {
		case xpath3.PathExpr:
			if v.Path != nil {
				if u, ok := derefXPathExpr(v.Filter).(xpath3.UnionExpr); ok {
					if !exprIsGrounded(u.Left) && !exprIsGrounded(u.Right) {
						found = true
						return false
					}
				}
			}
		case xpath3.PathStepExpr:
			if u, ok := derefXPathExpr(v.Left).(xpath3.UnionExpr); ok {
				if !exprIsGrounded(u.Left) && !exprIsGrounded(u.Right) {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

// exprIsGrounded returns true if an expression is grounded — it accesses
// only in-memory data (variable references, literals) and does not navigate
// the streaming source.
func exprIsGrounded(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch expr.(type) {
	case xpath3.VariableExpr, xpath3.LiteralExpr:
		return true
	}
	return false
}

// seqHasMixedPosture returns true if a SequenceExpr has items with different
// postures — specifically, some items are crawling (use descendant or
// descendant-or-self axes) while others are non-crawling (grounded variable
// refs, striding paths, etc.). Grounded + striding is fine; crawling mixed
// with non-crawling is not streamable.
func seqHasMixedPosture(seq xpath3.SequenceExpr) bool {
	return itemsHaveMixedPosture(seq.Items)
}

// itemsHaveMixedPosture returns true if a list of expressions has mixed
// postures — some crawling (descendant/descendant-or-self axes on the stream)
// and some non-crawling (grounded variables, striding paths).
func itemsHaveMixedPosture(items []xpath3.Expr) bool {
	if len(items) < 2 {
		return false
	}
	hasCrawling := false
	hasNonCrawling := false
	for _, item := range items {
		if seqItemIsCrawling(item) {
			hasCrawling = true
		} else {
			hasNonCrawling = true
		}
		if hasCrawling && hasNonCrawling {
			return true
		}
	}
	return false
}

// seqItemIsCrawling returns true if a single expression has crawling posture,
// meaning it uses descendant or descendant-or-self axes from the streaming
// context. variable references, literals, and striding expressions (child-only
// paths) are not crawling.
func seqItemIsCrawling(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch expr.(type) {
	case xpath3.VariableExpr, xpath3.LiteralExpr:
		return false
	}
	found := false
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		// Skip sub-expressions rooted in a variable — those navigate
		// in-memory data, not the stream.
		switch v := e.(type) {
		case xpath3.PathExpr:
			if _, isVar := derefXPathExpr(v.Filter).(xpath3.VariableExpr); isVar {
				return false
			}
		case xpath3.PathStepExpr:
			if _, isVar := derefXPathExpr(v.Left).(xpath3.VariableExpr); isVar {
				return false
			}
		}
		if lp, ok := e.(xpath3.LocationPath); ok {
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

func exprHasCrawlingGroundingArg(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	xpathstream.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" && (fc.Name == "reverse" || fc.Name == "innermost") { //nolint:goconst
				for _, arg := range fc.Args {
					if argHasStreamingCrawl(arg) {
						found = true
						return false
					}
				}
			}
		}
		return true
	})
	return found
}

// exprHasHigherOrderWithConsumingArg returns true if the expression contains a
// call to filter() or fold-right() whose first argument (the sequence) has a
// downward (consuming) selection that is not inside a grounding function.
// These higher-order functions require their sequence argument to be grounded
// because they need random access to the full sequence. In contrast, for-each()
// and fold-left() can process items sequentially and accept striding operands.
func exprHasHigherOrderWithConsumingArg(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	xpathstream.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		fc, ok := e.(xpath3.FunctionCall)
		if !ok || fc.Prefix != "" || len(fc.Args) < 2 {
			return true
		}
		switch fc.Name {
		case lexicon.StreamFilter, "fold-right":
			if argHasStreamingDownwardUngrounded(fc.Args[0]) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// argHasStreamingDownwardUngrounded returns true if the expression navigates
// downward into the streaming document (child/descendant axis) and is not
// wrapped in a grounding function like copy-of() or snapshot().
func argHasStreamingDownwardUngrounded(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	// If the outermost expression is a grounding function, the result is
	// grounded and safe for higher-order functions.
	if isGroundingExpr(expr) {
		return false
	}
	// If the outermost expression produces atomic values (e.g., data(),
	// string(), count()), the result is grounded — atomic values are not
	// streaming nodes and can be accessed randomly.
	if isAtomicResultExpr(expr) {
		return false
	}
	// If the expression is a path ending in an atomizing function
	// (e.g., ./path/to/nodes/data()), the result is atomic and safe.
	if ps, ok := expr.(xpath3.PathStepExpr); ok {
		if isAtomicResultExpr(ps.Right) {
			return false
		}
	}
	hasDown := false
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
		if hasDown {
			return false
		}
		// Skip into grounding function arguments — they are grounded.
		if fc, ok := e.(xpath3.FunctionCall); ok && isGroundingExpr(fc) {
			return false
		}
		switch v := e.(type) {
		case xpath3.LocationPath:
			for _, step := range v.Steps {
				switch step.Axis {
				case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
					hasDown = true
					return false
				}
			}
		}
		return true
	})
	return hasDown
}

// accRuleMatchesElement returns true if the accumulator rule's match pattern
// can match element nodes (as opposed to only text, attribute, or other node
// types). This is used to determine whether the context item in the select
// expression would be consuming in streaming mode.
func accRuleMatchesElement(rule *accumulatorRule) bool {
	if rule.Match == nil {
		return false
	}
	for _, alt := range rule.Match.Alternatives {
		// Check if the pattern's last step matches elements.
		// If the pattern is just a name test (e.g., "fig"), it matches elements.
		// If it's "text()" or "attribute(...)", it doesn't match elements.
		if !patternMatchesNonElement(alt.expr) {
			return true
		}
	}
	return false
}

// patternMatchesNonElement returns true if the pattern exclusively matches
// non-element nodes (text, attribute, comment, PI, document).
func patternMatchesNonElement(expr xpath3.Expr) bool {
	// Walk to find the last/most-specific step and check its node test.
	var lastTest xpath3.NodeTest
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
		switch v := e.(type) {
		case xpath3.LocationPath:
			if len(v.Steps) > 0 {
				lastTest = v.Steps[len(v.Steps)-1].NodeTest
			}
		}
		return true
	})
	if lastTest == nil {
		return false
	}
	if tt, ok := lastTest.(xpath3.TypeTest); ok {
		switch tt.Kind {
		case xpath3.NodeKindText, xpath3.NodeKindComment, xpath3.NodeKindProcessingInstruction:
			return true
		}
	}
	return false
}

// exprUsesAccumulatorAfterOnAncestor returns true if the expression contains
// a path expression like ../accumulator-after(...) where accumulator-after
// is applied to an ancestor node. In streaming, the ancestor's post-descent
// accumulator value is not yet available.
func exprUsesAccumulatorAfterOnAncestor(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	xpathstream.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		// Check for PathStepExpr: Left / Right where Left uses upward axis
		// and Right is accumulator-after function call.
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if hasUpwardAxis(ps.Left) && exprCallsFunction(ps.Right, "accumulator-after") {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// hasUpwardAxis checks if an expression contains a parent or ancestor axis step.
func hasUpwardAxis(expr xpath3.Expr) bool {
	found := false
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if lp, ok := e.(xpath3.LocationPath); ok {
			for _, step := range lp.Steps {
				if step.Axis == xpath3.AxisParent || step.Axis == xpath3.AxisAncestor || step.Axis == xpath3.AxisAncestorOrSelf {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

// exprCallsFunction checks if an expression is or contains a specific function call.
func exprCallsFunction(expr xpath3.Expr, name string) bool {
	found := false
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" && fc.Name == name {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// argHasStreamingCrawl returns true if an expression has a descendant-or-self
// axis step that operates on streaming (non-grounded) data. Crawling inside
// a grounding function like snapshot() is fine and not flagged.
func argHasStreamingCrawl(expr xpath3.Expr) bool {
	return argHasStreamingCrawlInner(expr, false)
}

func argHasStreamingCrawlInner(expr xpath3.Expr, grounded bool) bool {
	expr = derefXPathExpr(expr)
	if grounded {
		return false
	}
	switch v := expr.(type) {
	case xpath3.LocationPath:
		for _, step := range v.Steps {
			if step.Axis == xpath3.AxisDescendant || step.Axis == xpath3.AxisDescendantOrSelf {
				return true
			}
		}
	case xpath3.PathStepExpr:
		if v.DescOrSelf {
			// Check if the LHS is grounding
			if !isGroundingExpr(v.Left) {
				return true
			}
		}
		leftGrounding := isGroundingExpr(v.Left) || isAtomicResultExpr(v.Left)
		if argHasStreamingCrawlInner(v.Left, false) {
			return true
		}
		return argHasStreamingCrawlInner(v.Right, leftGrounding)
	case xpath3.PathExpr:
		filterGrounding := isGroundingExpr(v.Filter)
		if argHasStreamingCrawlInner(v.Filter, false) {
			return true
		}
		if v.Path != nil {
			lp := *v.Path
			return argHasStreamingCrawlInner(lp, filterGrounding)
		}
	case xpath3.FunctionCall:
		g := isGroundingExpr(v)
		for _, arg := range v.Args {
			if argHasStreamingCrawlInner(arg, g) {
				return true
			}
		}
	case xpath3.FilterExpr:
		return argHasStreamingCrawlInner(v.Expr, false)
	}
	return false
}

// walkExprWithGrounding walks an expression tree, tracking whether we are
// inside a grounding function (snapshot, copy-of, copy).
func walkExprWithGrounding(expr xpath3.Expr, insideGrounding bool, fn func(xpath3.Expr, bool) bool) {
	if expr == nil {
		return
	}
	expr = derefXPathExpr(expr)
	if !fn(expr, insideGrounding) {
		return
	}

	// Check if this is a grounding function call
	newGrounding := insideGrounding
	if fc, ok := expr.(xpath3.FunctionCall); ok {
		if fc.Prefix == "" && isGroundingFuncName(fc.Name) {
			newGrounding = true
		}
	}

	// Walk children with appropriate grounding state
	switch e := expr.(type) {
	case xpath3.FunctionCall:
		for _, arg := range e.Args {
			walkExprWithGrounding(arg, newGrounding, fn)
		}
	case xpath3.PathExpr:
		walkExprWithGrounding(e.Filter, insideGrounding, fn)
		if e.Path != nil {
			g := insideGrounding || isGroundingExpr(e.Filter) || isAtomicResultExpr(e.Filter)
			lp := *e.Path
			walkExprWithGrounding(lp, g, fn)
		}
	case xpath3.PathStepExpr:
		walkExprWithGrounding(e.Left, insideGrounding, fn)
		// If left side produces grounded or atomic result, right side operates on non-streaming data
		if isGroundingExpr(e.Left) || isAtomicResultExpr(e.Left) {
			walkExprWithGrounding(e.Right, true, fn)
		} else {
			walkExprWithGrounding(e.Right, insideGrounding, fn)
		}
	case xpath3.FilterExpr:
		walkExprWithGrounding(e.Expr, insideGrounding, fn)
		g := insideGrounding || isGroundingExpr(e.Expr) || isAtomicResultExpr(e.Expr)
		for _, pred := range e.Predicates {
			walkExprWithGrounding(pred, g, fn)
		}
	case xpath3.LocationPath:
		for _, step := range e.Steps {
			for _, pred := range step.Predicates {
				walkExprWithGrounding(pred, insideGrounding, fn)
			}
		}
	case xpath3.BinaryExpr:
		walkExprWithGrounding(e.Left, insideGrounding, fn)
		walkExprWithGrounding(e.Right, insideGrounding, fn)
	case xpath3.UnaryExpr:
		walkExprWithGrounding(e.Operand, insideGrounding, fn)
	case xpath3.ConcatExpr:
		walkExprWithGrounding(e.Left, insideGrounding, fn)
		walkExprWithGrounding(e.Right, insideGrounding, fn)
	case xpath3.SimpleMapExpr:
		walkExprWithGrounding(e.Left, insideGrounding, fn)
		g := insideGrounding || isGroundingExpr(e.Left)
		walkExprWithGrounding(e.Right, g, fn)
	case xpath3.UnionExpr:
		walkExprWithGrounding(e.Left, insideGrounding, fn)
		walkExprWithGrounding(e.Right, insideGrounding, fn)
	case xpath3.IfExpr:
		walkExprWithGrounding(e.Cond, insideGrounding, fn)
		walkExprWithGrounding(e.Then, insideGrounding, fn)
		walkExprWithGrounding(e.Else, insideGrounding, fn)
	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			walkExprWithGrounding(item, insideGrounding, fn)
		}
	case xpath3.FLWORExpr:
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case xpath3.ForClause:
				walkExprWithGrounding(c.Expr, insideGrounding, fn)
			case xpath3.LetClause:
				walkExprWithGrounding(c.Expr, insideGrounding, fn)
			}
		}
		walkExprWithGrounding(e.Return, insideGrounding, fn)
	}
}

// isGroundingFuncName returns true if the given function name is a grounding
// function (produces grounded, non-streaming output).
func isGroundingFuncName(name string) bool {
	switch name {
	case "snapshot", "copy-of", "copy", "current-group", //nolint:goconst
		"outermost", "innermost", "parse-xml", "parse-xml-fragment",
		"doc", "document", "sort", "reverse", "head":
		return true
	}
	return false
}

// isGroundingExpr returns true if an expression grounds its input,
// meaning subsequent operations on its result work on non-streaming data.
// This is different from functions that merely produce atomic output (like count/sum).
func isGroundingExpr(expr xpath3.Expr) bool {
	return isGroundingExprSS(nil, expr)
}

func isGroundingExprSS(ss *Stylesheet, expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	// A range expression (X to Y) always produces a sequence of integers —
	// it never navigates the streaming source, so it is inherently grounded.
	if _, ok := expr.(xpath3.RangeExpr); ok {
		return true
	}
	if fc, ok := expr.(xpath3.FunctionCall); ok {
		if fc.Prefix == "" {
			return isGroundingFuncName(fc.Name)
		}
		// Check user-defined absorbing/unclassified functions
		cat := lookupFuncStreamability(ss, fc.Name, len(fc.Args))
		if cat == "absorbing" || cat == "unclassified" {
			return true
		}
	}
	// A FilterExpr wrapping a grounding expression is still grounding.
	// E.g. current-group()[1] is a predicate on grounded data.
	// Only unwrap if the inner expression is a FunctionCall to avoid
	// treating arbitrary filtered expressions as grounded.
	if fe, ok := expr.(xpath3.FilterExpr); ok {
		if fc, ok2 := derefXPathExpr(fe.Expr).(xpath3.FunctionCall); ok2 {
			if fc.Prefix == "" {
				return isGroundingFuncName(fc.Name)
			}
		}
	}
	return false
}

// exprProducesAtomicResult returns true if the outermost expression produces an
// atomic result (string, number, boolean) rather than streaming nodes. This
// includes function calls like string(), number(), count(), etc., as well as
// arithmetic expressions, comparisons, and other operators that always produce
// atomic values.
func exprProducesAtomicResult(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch e := expr.(type) {
	case xpath3.FunctionCall:
		return isAtomicResultExpr(e) || isGroundingExpr(e)
	case xpath3.BinaryExpr:
		// Arithmetic and comparison operators produce atomic results
		return true
	case xpath3.LiteralExpr:
		return true
	}
	return false
}

// isAtomicResultExpr returns true if an expression produces an atomic result
// (string, number, boolean). Operations AFTER such a function operate on atomic
// data, but the function's ARGUMENTS are still streaming.
func isAtomicResultExpr(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	if fc, ok := expr.(xpath3.FunctionCall); ok {
		if fc.Prefix == "" {
			switch fc.Name {
			case "tokenize", "string", "data", "number", "boolean", //nolint:goconst
				"name", "local-name", "namespace-uri", "string-length",
				"normalize-space", "count", "sum", "avg", "min", "max",
				"string-join", "concat", "contains", "starts-with",
				"ends-with", "matches", "replace", "translate",
				"substring", "substring-before", "substring-after",
				"upper-case", "lower-case", "round", "floor", "ceiling",
				"abs", "not", "true", "false", "position", lexicon.FnLast,
				"empty", "exists", "head", "tail":
				return true
			}
		}
	}
	return false
}

// exprUsesLastOutsideGrounding returns true if the expression uses last()
// outside a grounding context (snapshot, copy-of, current-group, etc.).
func exprUsesLastOutsideGrounding(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	walkExprWithGrounding(expr.AST(), false, func(e xpath3.Expr, insideGrounding bool) bool {
		if found {
			return false
		}
		if insideGrounding {
			return true
		}
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" && fc.Name == "last" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// exprUsesFunctionOutsideGrounding returns true if the expression calls the
// named function outside a grounding context.
func exprUsesFunctionOutsideGrounding(expr *xpath3.Expression, name string) bool {
	if expr == nil {
		return false
	}
	found := false
	walkExprWithGrounding(expr.AST(), false, func(e xpath3.Expr, insideGrounding bool) bool {
		if found {
			return false
		}
		if insideGrounding {
			return true
		}
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" && fc.Name == name {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// exprHasNonMotionlessStreamingPredicate checks for non-motionless predicates
// on streaming steps, but NOT on grounded data (after snapshot/copy-of/tokenize etc.).
func exprHasNonMotionlessStreamingPredicate(ss *Stylesheet, expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	walkExprCheckPredicates(ss, expr.AST(), false, &found)
	return found
}

// walkExprCheckPredicates walks the AST looking for non-motionless predicates
// on streaming (non-grounded) steps.
func walkExprCheckPredicates(ss *Stylesheet, expr xpath3.Expr, grounded bool, found *bool) {
	if expr == nil || *found {
		return
	}
	expr = derefXPathExpr(expr)

	switch e := expr.(type) {
	case xpath3.LocationPath:
		if !grounded {
			for _, step := range e.Steps {
				for _, pred := range step.Predicates {
					if predicateIsNonMotionlessSS(ss, pred, &step) {
						*found = true
						return
					}
				}
			}
		}
	case xpath3.FilterExpr:
		// variable references are always grounded — they hold materialized
		// sequences, not streaming nodes. Treat them like grounding exprs.
		_, isVarRef := derefXPathExpr(e.Expr).(xpath3.VariableExpr)
		g := grounded || isGroundingExprSS(ss, e.Expr) || isAtomicResultExpr(e.Expr) || isVarRef
		walkExprCheckPredicates(ss, e.Expr, grounded, found)
		if !g {
			for _, pred := range e.Predicates {
				if predicateIsNonMotionlessSS(ss, pred, nil) {
					*found = true
					return
				}
			}
		}
	case xpath3.SimpleMapExpr:
		walkExprCheckPredicates(ss, e.Left, grounded, found)
		// RHS of ! gets individual items — check if LHS produces grounded data
		// Atomic result functions like tokenize() also ground the RHS
		g := grounded || isGroundingExprSS(ss, e.Left) || isAtomicResultExpr(e.Left)
		walkExprCheckPredicates(ss, e.Right, g, found)
	case xpath3.PathStepExpr:
		walkExprCheckPredicates(ss, e.Left, grounded, found)
		g := grounded || isGroundingExprSS(ss, e.Left) || isAtomicResultExpr(e.Left)
		walkExprCheckPredicates(ss, e.Right, g, found)
	case xpath3.FunctionCall:
		// Function arguments are still streaming — only grounding functions
		// (snapshot, copy-of) ground their input.
		g := grounded || isGroundingExprSS(ss, e)
		for _, arg := range e.Args {
			walkExprCheckPredicates(ss, arg, g, found)
		}
		// For user-defined inspection functions, predicates on the call
		// are motionless since the result is atomic/upward.
		if e.Prefix != "" {
			cat := lookupFuncStreamability(ss, e.Name, len(e.Args))
			if cat == "inspection" || cat == "ascent" {
				return
			}
		}
	case xpath3.PathExpr:
		walkExprCheckPredicates(ss, e.Filter, grounded, found)
		if e.Path != nil {
			g := grounded || isGroundingExprSS(ss, e.Filter)
			lp := *e.Path
			walkExprCheckPredicates(ss, lp, g, found)
		}
	case xpath3.BinaryExpr:
		walkExprCheckPredicates(ss, e.Left, grounded, found)
		walkExprCheckPredicates(ss, e.Right, grounded, found)
	case xpath3.UnaryExpr:
		walkExprCheckPredicates(ss, e.Operand, grounded, found)
	case xpath3.InstanceOfExpr:
		walkExprCheckPredicates(ss, e.Expr, grounded, found)
	case xpath3.CastExpr:
		walkExprCheckPredicates(ss, e.Expr, grounded, found)
	case xpath3.CastableExpr:
		walkExprCheckPredicates(ss, e.Expr, grounded, found)
	case xpath3.TreatAsExpr:
		walkExprCheckPredicates(ss, e.Expr, grounded, found)
	case xpath3.ConcatExpr:
		walkExprCheckPredicates(ss, e.Left, grounded, found)
		walkExprCheckPredicates(ss, e.Right, grounded, found)
	case xpath3.UnionExpr:
		walkExprCheckPredicates(ss, e.Left, grounded, found)
		walkExprCheckPredicates(ss, e.Right, grounded, found)
	case xpath3.IfExpr:
		walkExprCheckPredicates(ss, e.Cond, grounded, found)
		walkExprCheckPredicates(ss, e.Then, grounded, found)
		walkExprCheckPredicates(ss, e.Else, grounded, found)
	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			walkExprCheckPredicates(ss, item, grounded, found)
		}
	case xpath3.FLWORExpr:
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case xpath3.ForClause:
				walkExprCheckPredicates(ss, c.Expr, grounded, found)
			case xpath3.LetClause:
				walkExprCheckPredicates(ss, c.Expr, grounded, found)
			}
		}
		walkExprCheckPredicates(ss, e.Return, grounded, found)
	}
}

// predicateIsNonMotionless returns true if a predicate expression navigates
// downward (uses child/descendant axes), uses last(), or accesses the context item.
func predicateIsNonMotionless(pred xpath3.Expr) bool {
	return predicateIsNonMotionlessSS(nil, pred, nil)
}

// stepContextIsAtomic returns true if the context item in a predicate on this
// step has its string value available without navigating to child nodes.
// Attribute, text, comment, PI, and namespace nodes have atomic values;
// element and document nodes require reading descendant text.
func stepContextIsAtomic(axis xpath3.AxisType, nt xpath3.NodeTest) bool {
	// Attribute axis always selects attribute nodes — atomic value.
	if axis == xpath3.AxisAttribute {
		return true
	}
	switch v := nt.(type) {
	case xpath3.TypeTest:
		switch v.Kind {
		case xpath3.NodeKindText, xpath3.NodeKindComment, xpath3.NodeKindProcessingInstruction:
			return true
		}
	case xpath3.AttributeTest:
		return true
	case xpath3.NamespaceNodeTest:
		return true
	case xpath3.PITest:
		return true
	}
	return false
}

func predicateIsNonMotionlessSS(ss *Stylesheet, pred xpath3.Expr, step *xpath3.Step) bool {
	nonMotionless := false
	var walkPred func(e xpath3.Expr, underNonContext bool)
	walkPred = func(e xpath3.Expr, underNonContext bool) {
		if nonMotionless || e == nil {
			return
		}
		e = derefXPathExpr(e)
		switch v := e.(type) {
		case xpath3.LocationPath:
			if !underNonContext {
				for _, s := range v.Steps {
					switch s.Axis {
					case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
						nonMotionless = true
						return
					}
				}
			}
			// Walk predicates within steps.
			for _, s := range v.Steps {
				for _, p := range s.Predicates {
					walkPred(p, underNonContext)
				}
			}
		case xpath3.PathStepExpr:
			if v.DescOrSelf && !underNonContext {
				nonMotionless = true
				return
			}
			leftIsNonContext := isNonContextRoot(v.Left)
			walkPred(v.Left, underNonContext)
			walkPred(v.Right, underNonContext || leftIsNonContext)
		case xpath3.PathExpr:
			filterIsNonContext := isNonContextRoot(v.Filter)
			walkPred(v.Filter, underNonContext)
			if v.Path != nil {
				lp := *v.Path
				walkPred(lp, underNonContext || filterIsNonContext)
			}
		case xpath3.FunctionCall:
			if v.Prefix == "" && v.Name == "last" {
				nonMotionless = true
				return
			}
			// current-group() and current-grouping-key() always return
			// materialized (grounded) data.  Navigation into them (e.g.
			// current-group()[1]/Date) is motionless w.r.t. the stream.
			if v.Prefix == "" && (v.Name == "current-group" || v.Name == "current-grouping-key") {
				return // skip children — result is grounded
			}
			// Property-access functions are motionless.
			if v.Prefix == "" {
				switch v.Name {
				case "name", "local-name", "namespace-uri", "node-name",
					"self", "generate-id", "base-uri", "document-uri",
					"nilled", "has-children", "string-length":
					return // skip children
				}
			}
			// User-defined functions with motionless streamability annotations.
			if v.Prefix != "" {
				cat := lookupFuncStreamability(ss, v.Name, len(v.Args))
				if cat == "inspection" || cat == "ascent" || cat == "filter" {
					return // skip children
				}
			}
			for _, arg := range v.Args {
				walkPred(arg, underNonContext)
			}
		case xpath3.ContextItemExpr:
			if step == nil || !stepContextIsAtomic(step.Axis, step.NodeTest) {
				nonMotionless = true
				return
			}
		case xpath3.FilterExpr:
			walkPred(v.Expr, underNonContext)
			for _, p := range v.Predicates {
				walkPred(p, underNonContext)
			}
		case xpath3.BinaryExpr:
			walkPred(v.Left, underNonContext)
			walkPred(v.Right, underNonContext)
		case xpath3.UnaryExpr:
			walkPred(v.Operand, underNonContext)
		case xpath3.ConcatExpr:
			walkPred(v.Left, underNonContext)
			walkPred(v.Right, underNonContext)
		case xpath3.UnionExpr:
			walkPred(v.Left, underNonContext)
			walkPred(v.Right, underNonContext)
		case xpath3.IfExpr:
			walkPred(v.Cond, underNonContext)
			walkPred(v.Then, underNonContext)
			walkPred(v.Else, underNonContext)
		case xpath3.SequenceExpr:
			for _, item := range v.Items {
				walkPred(item, underNonContext)
			}
		case xpath3.LiteralExpr, xpath3.VariableExpr, xpath3.RootExpr, xpath3.PlaceholderExpr:
			// leaf nodes — nothing to walk
		case xpath3.RangeExpr:
			walkPred(v.Start, underNonContext)
			walkPred(v.End, underNonContext)
		case xpath3.SimpleMapExpr:
			walkPred(v.Left, underNonContext)
			walkPred(v.Right, underNonContext)
		case xpath3.IntersectExceptExpr:
			walkPred(v.Left, underNonContext)
			walkPred(v.Right, underNonContext)
		default:
			// VM-internal types (vmLocationPathExpr, vmPathExpr) can't be
			// matched in xslt3.  Use the old WalkExpr-based approach for
			// the CHILDREN only (not the node itself) to avoid infinite
			// recursion.
			first := true
			xpathstream.WalkExpr(e, func(child xpath3.Expr) bool {
				if nonMotionless {
					return false
				}
				// Skip the first call (WalkExpr calls fn with the root node first).
				if first {
					first = false
					return true // continue to walk children
				}
				walkPred(child, underNonContext)
				return false // we handle recursion ourselves
			})
		}
	}
	walkPred(pred, false)
	return nonMotionless
}

// exprHasUpThenDownNavigation returns true if any location path in the
// expression first navigates upward (parent/ancestor axis) and then downward
// (child/descendant axis). This up-then-down pattern is not streamable because
// the downward step from an ancestor may access nodes that have not yet been
// streamed.
func exprHasUpThenDownNavigation(expr *xpath3.Expression) bool {
	return xpathstream.ExprHasUpThenDownNavigation(expr)
}
