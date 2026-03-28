package xslt3

import (
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
)

func countDownwardInInstructions(ss *Stylesheet, instructions []instruction) int {
	total := 0
	for _, inst := range instructions {
		// When a for-each-group select ends with a grounding function
		// (e.g. copy-of), the group-by/group-adjacent/sort expressions
		// operate on grounded (in-memory) items, not the streaming source.
		// Only count the select expression itself as a streaming downward
		// selection; skip the grouping key and sort expressions.
		if fg, ok := inst.(*forEachGroupInst); ok {
			if fg.Select != nil && exprEndsWithGrounding(fg.Select.AST()) {
				total += countStreamingDownwardSelections(ss, fg.Select.AST())
				continue
			}
		}
		for _, expr := range getInstructionExprs(inst) {
			total += countStreamingDownwardSelections(ss, expr.AST())
		}
	}
	return total
}

// countCurrentGroupConsumingRefs counts how many times current-group() is used
// in a way that involves downward navigation from the group items.
// Simple uses like count(current-group()) are OK (they don't navigate downward).
// Uses like current-group()/AUTHOR or copy-of(current-group()) ARE consuming.
// In choose branches (when/otherwise), only the max is counted since only one
// branch executes at a time.
func countCurrentGroupConsumingRefs(body []instruction) int {
	count := 0
	for _, inst := range body {
		// For choose instructions, take the max across branches, not the sum.
		if choose, ok := inst.(*chooseInst); ok {
			maxBranch := 0
			for _, when := range choose.When {
				branchCount := countCurrentGroupConsumingRefs(when.Body)
				if branchCount > maxBranch {
					maxBranch = branchCount
				}
			}
			if choose.Otherwise != nil {
				branchCount := countCurrentGroupConsumingRefs(choose.Otherwise)
				if branchCount > maxBranch {
					maxBranch = branchCount
				}
			}
			count += maxBranch
			continue
		}

		// For fork instructions, each branch gets its own copy of the
		// stream, so take the max across branches (like choose).
		if fork, ok := inst.(*forkInst); ok {
			maxBranch := 0
			for _, branch := range fork.Branches {
				branchCount := countCurrentGroupConsumingRefs(branch)
				if branchCount > maxBranch {
					maxBranch = branchCount
				}
			}
			count += maxBranch
			continue
		}

		for _, expr := range getInstructionExprs(inst) {
			if expr == nil {
				continue
			}
			count += countCurrentGroupConsumingInExpr(expr.AST())
		}
		// Check LRE attribute AVTs for current-group() references.
		if lre, ok := inst.(*literalResultElement); ok {
			for _, attr := range lre.Attrs {
				if attr.Value != nil {
					for _, part := range attr.Value.parts {
						if part.expr != nil {
							count += countCurrentGroupConsumingInExpr(part.expr.AST())
						}
					}
				}
			}
		}
		// Check child instructions, but don't recurse into nested for-each-group
		// bodies (their current-group() refers to their own group, not ours).
		if _, ok := inst.(*forEachGroupInst); !ok {
			for _, children := range getChildInstructions(inst) {
				count += countCurrentGroupConsumingRefs(children)
			}
		}
	}
	return count
}

// countCurrentGroupConsumingInExpr counts consuming uses of current-group()
// in an expression. In streaming, any use of current-group() that requires
// enumerating the group members is consuming. This includes count(),
// current-group()/AUTHOR, copy-of(current-group()), etc.
func countCurrentGroupConsumingInExpr(expr xpath3.Expr) int {
	expr = derefXPathExpr(expr)
	count := 0
	switch e := expr.(type) {
	case xpath3.PathStepExpr:
		// current-group()/child — consuming (navigates into element content)
		// current-group()/@attr — motionless (attributes are immediately available)
		if isCurrentGroupCall(e.Left) {
			if pathStepRightHasDownward(e.Right) || e.DescOrSelf {
				count++
			}
		} else {
			count += countCurrentGroupConsumingInExpr(e.Left)
		}
		count += countCurrentGroupConsumingInExpr(e.Right)
	case xpath3.FunctionCall:
		if e.Prefix == "" {
			switch e.Name {
			case lexicon.FnCurrentGroup:
				// Any reference to current-group() is consuming in streaming
				return 1
			case "current-grouping-key":
				// current-grouping-key() is motionless — it doesn't consume the stream
				return 0
			}
		}
		for _, arg := range e.Args {
			count += countCurrentGroupConsumingInExpr(arg)
		}
	case xpath3.FilterExpr:
		count += countCurrentGroupConsumingInExpr(e.Expr)
		for _, pred := range e.Predicates {
			count += countCurrentGroupConsumingInExpr(pred)
		}
	case xpath3.BinaryExpr:
		count += countCurrentGroupConsumingInExpr(e.Left)
		count += countCurrentGroupConsumingInExpr(e.Right)
	case xpath3.ConcatExpr:
		count += countCurrentGroupConsumingInExpr(e.Left)
		count += countCurrentGroupConsumingInExpr(e.Right)
	case xpath3.UnionExpr:
		count += countCurrentGroupConsumingInExpr(e.Left)
		count += countCurrentGroupConsumingInExpr(e.Right)
	case xpath3.SimpleMapExpr:
		count += countCurrentGroupConsumingInExpr(e.Left)
		count += countCurrentGroupConsumingInExpr(e.Right)
	case xpath3.PathExpr:
		// current-group()/child → consuming (navigates into element content)
		// current-group()/@attr → motionless (attributes are immediately available)
		if isCurrentGroupCall(e.Filter) {
			if e.Path != nil && pathHasDownwardStep(*e.Path) {
				count++
			}
		} else {
			count += countCurrentGroupConsumingInExpr(e.Filter)
		}
		if e.Path != nil {
			for _, step := range e.Path.Steps {
				for _, pred := range step.Predicates {
					count += countCurrentGroupConsumingInExpr(pred)
				}
			}
		}
	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			count += countCurrentGroupConsumingInExpr(item)
		}
	case xpath3.IfExpr:
		count += countCurrentGroupConsumingInExpr(e.Cond)
		thenCount := countCurrentGroupConsumingInExpr(e.Then)
		elseCount := countCurrentGroupConsumingInExpr(e.Else)
		if thenCount > elseCount {
			count += thenCount
		} else {
			count += elseCount
		}
	}
	return count
}

// isCurrentGroupCall returns true if expr is a call to current-group().
func isCurrentGroupCall(expr xpath3.Expr) bool {
	fc, ok := expr.(xpath3.FunctionCall)
	return ok && fc.Prefix == "" && fc.Name == lexicon.FnCurrentGroup
}

// pathHasDownwardStep returns true if a LocationPath has any child or
// descendant axis steps (i.e., navigates into element content).
func pathHasDownwardStep(lp xpath3.LocationPath) bool {
	for _, step := range lp.Steps {
		switch step.Axis {
		case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
			return true
		}
	}
	return false
}

// pathStepRightHasDownward returns true if the right side of a PathStepExpr
// contains a downward step (child/descendant axis).
func pathStepRightHasDownward(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch e := expr.(type) {
	case xpath3.LocationPath:
		return pathHasDownwardStep(e)
	case xpath3.PathStepExpr:
		return pathStepRightHasDownward(e.Left) || pathStepRightHasDownward(e.Right) || e.DescOrSelf
	}
	return false
}

// exprEndsWithGrounding returns true if the outermost expression grounds its
// result (e.g., ends with copy-of() or snapshot()). This means the crawling
// is followed by grounding, making it streamable for grouping.
func exprEndsWithGrounding(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch e := expr.(type) {
	case xpath3.PathStepExpr:
		// Check if the rightmost step is a grounding function
		return exprEndsWithGrounding(e.Right)
	case xpath3.PathExpr:
		if e.Path != nil {
			lp := *e.Path
			return exprEndsWithGrounding(lp)
		}
		return isGroundingExpr(e.Filter)
	case xpath3.FilterExpr:
		// A predicate filter on a grounded expression is still grounded —
		// it returns a subset of already-grounded nodes.
		return exprEndsWithGrounding(e.Expr)
	case xpath3.FunctionCall:
		return isGroundingExpr(e)
	case xpath3.LocationPath:
		// A LocationPath ending with a grounding step? Not typical.
		// Check last step for function-like patterns.
		return false
	case xpath3.SimpleMapExpr:
		// E1 ! E2: the result is grounded if E2 is grounding
		// (e.g. items ! copy-of(.))
		return exprEndsWithGrounding(e.Right)
	case xpath3.RangeExpr:
		// A range expression (X to Y) always produces integers.
		// Even if its arguments access descendants (e.g., 1 to count(.//*)),
		// the result is atomic — not streamed nodes.
		return true
	}
	return false
}

// countDownwardFromCurrentGroup counts how many distinct downward selections
// occur after current-group() in a single expression. For example,
// current-group()/(AUTHOR||TITLE) has 2 downward selections (AUTHOR and TITLE).
func countDownwardFromCurrentGroup(expr xpath3.Expr) int {
	expr = derefXPathExpr(expr)
	count := 0
	switch e := expr.(type) {
	case xpath3.PathStepExpr:
		if isCurrentGroupCall(e.Left) {
			// Count downward selections in the RHS after current-group()
			count += countDownwardSelectionsInPathRHS(e.Right)
		} else {
			count += countDownwardFromCurrentGroup(e.Left)
		}
		count += countDownwardFromCurrentGroup(e.Right)
	case xpath3.FunctionCall:
		for _, arg := range e.Args {
			count += countDownwardFromCurrentGroup(arg)
		}
	case xpath3.BinaryExpr:
		count += countDownwardFromCurrentGroup(e.Left)
		count += countDownwardFromCurrentGroup(e.Right)
	case xpath3.ConcatExpr:
		count += countDownwardFromCurrentGroup(e.Left)
		count += countDownwardFromCurrentGroup(e.Right)
	case xpath3.FilterExpr:
		count += countDownwardFromCurrentGroup(e.Expr)
	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			count += countDownwardFromCurrentGroup(item)
		}
	case xpath3.IfExpr:
		count += countDownwardFromCurrentGroup(e.Cond)
		thenCount := countDownwardFromCurrentGroup(e.Then)
		elseCount := countDownwardFromCurrentGroup(e.Else)
		if thenCount > elseCount {
			count += thenCount
		} else {
			count += elseCount
		}
	}
	return count
}

// countDownwardSelectionsInPathRHS counts how many distinct downward selections
// are in the right-hand side of a path expression. ConcatExpr with 2 child axes
// counts as 2 selections.
func countDownwardSelectionsInPathRHS(expr xpath3.Expr) int {
	expr = derefXPathExpr(expr)
	switch e := expr.(type) {
	case xpath3.LocationPath:
		for _, step := range e.Steps {
			switch step.Axis {
			case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
				return 1
			}
		}
		return 0
	case xpath3.ConcatExpr:
		return countDownwardSelectionsInPathRHS(e.Left) + countDownwardSelectionsInPathRHS(e.Right)
	case xpath3.BinaryExpr:
		return countDownwardSelectionsInPathRHS(e.Left) + countDownwardSelectionsInPathRHS(e.Right)
	case xpath3.SequenceExpr:
		total := 0
		for _, item := range e.Items {
			total += countDownwardSelectionsInPathRHS(item)
		}
		return total
	case xpath3.FunctionCall:
		total := 0
		for _, arg := range e.Args {
			total += countDownwardSelectionsInPathRHS(arg)
		}
		return total
	case xpath3.PathStepExpr:
		return countDownwardSelectionsInPathRHS(e.Left) + countDownwardSelectionsInPathRHS(e.Right)
	}
	return 0
}

// returnNavigatesFromVars checks whether an expression contains a path
// that navigates from one of the given variable names (e.g., $x/child::*).
// This is used to detect FLWOR patterns like "for $x in .//foo return $x/bar"
// where the return clause requires buffering the bound node.
func returnNavigatesFromVars(expr xpath3.Expr, vars map[string]struct{}) bool {
	found := false
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		var root xpath3.Expr
		switch t := e.(type) {
		case xpath3.PathStepExpr:
			root = pathRoot(t)
		case xpath3.PathExpr:
			root = t.Filter
		default:
			return true
		}
		if ve, ok := root.(xpath3.VariableExpr); ok {
			if _, match := vars[ve.Name]; match {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// pathRoot returns the leftmost expression in a PathStepExpr/PathExpr chain.
func pathRoot(expr xpath3.Expr) xpath3.Expr {
	for {
		switch e := expr.(type) {
		case xpath3.PathStepExpr:
			expr = e.Left
		case xpath3.PathExpr:
			return e.Filter
		default:
			return expr
		}
	}
}

// exprHasContextOnlyDownward is like exprHasContextDownward but skips downward
// steps that are reached via a path from a function call or variable reference.
// For example, current-group()/node() has a child step (node()), but it
// navigates from current-group(), not from the context item.  Similarly,
// $var/* navigates from a variable, not from context.
func exprHasContextOnlyDownward(expr xpath3.Expr) bool {
	found := false
	var walk func(e xpath3.Expr, underNonContext bool)
	walk = func(e xpath3.Expr, underNonContext bool) {
		if found {
			return
		}
		e = derefXPathExpr(e)
		switch v := e.(type) {
		case xpath3.PathStepExpr:
			// Check if the left side is a function call or variable — if so,
			// the right side's downward steps are NOT from context.
			leftIsNonContext := isNonContextRoot(v.Left)
			walk(v.Left, underNonContext)
			walk(v.Right, underNonContext || leftIsNonContext)
			return
		case xpath3.PathExpr:
			// PathExpr: Filter/Path — if Filter is non-context, Path steps
			// are from the filter result, not the context item.
			filterIsNonContext := isNonContextRoot(v.Filter)
			walk(v.Filter, underNonContext)
			if v.Path != nil {
				lp := *v.Path
				walk(lp, underNonContext || filterIsNonContext)
			}
			return
		case xpath3.SimpleMapExpr:
			// SimpleMapExpr (E1 ! E2): if the left side is a non-context root
			// (e.g., current-group()), the right side's steps navigate from the
			// map result, not from the context item.
			leftIsNonContext := isNonContextRoot(v.Left)
			walk(v.Left, underNonContext)
			walk(v.Right, underNonContext || leftIsNonContext)
			return
		case xpath3.LocationPath:
			if underNonContext {
				return // skip — this is a step from a non-context root
			}
			for _, step := range v.Steps {
				switch step.Axis {
				case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
					found = true
					return
				}
			}
			return
		}
		// For other node types, walk children normally.
		xpathstream.WalkExpr(e, func(child xpath3.Expr) bool {
			if found {
				return false
			}
			child = derefXPathExpr(child)
			// Don't recurse here; handle via our custom walker for path types.
			switch child.(type) {
			case xpath3.PathStepExpr, xpath3.PathExpr:
				walk(child, underNonContext)
				return false
			case xpath3.LocationPath:
				walk(child, underNonContext)
				return false
			}
			return true
		})
	}
	walk(expr, false)
	return found
}

// isNonContextRoot returns true if the expression is a function call or
// variable reference (i.e. not the implicit context item).
func isNonContextRoot(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch expr := expr.(type) {
	case xpath3.FunctionCall:
		return true
	case xpath3.VariableExpr:
		return true
	case xpath3.FilterExpr:
		// FilterExpr wraps a primary expression with predicates.
		// If the inner expression is a function call or variable, it's non-context.
		return isNonContextRoot(expr.Expr)
	}
	return false
}

// checkStreamableFunctionBody checks a function declared with streamability
// for violations of its declared streamability category.
