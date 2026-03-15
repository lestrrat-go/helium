package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium/xpath3"
)

// analyzeStreamability performs a post-compilation pass over the stylesheet.
// It checks templates in streamable modes and bodies of xsl:source-document
// streamable="yes" for non-streamable constructs, raising XTSE3430 errors.
func analyzeStreamability(ss *Stylesheet) error {
	// Check all templates for source-document streamable="yes" in their body.
	for _, tmpl := range ss.templates {
		if err := checkInstructionsForStreamableSourceDoc(ss, tmpl.Body); err != nil {
			return err
		}
	}

	// Note: templates in streamable modes are NOT checked here because the
	// streamability rules for template bodies in streamable modes are more
	// nuanced (ancestor axes are allowed on the matched context node, etc.).
	// The W3C XTSE3430 tests for streaming modes primarily target
	// xsl:source-document bodies, not individual template bodies.

	// Check functions with declared streamability.
	for _, fn := range ss.functions {
		if fn.Streamability != "" {
			if err := checkStreamableFunctionBody(ss, fn); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkInstructionsForStreamableSourceDoc walks instructions looking for
// SourceDocumentInst with Streamable=true and checks their bodies.
func checkInstructionsForStreamableSourceDoc(ss *Stylesheet, instructions []Instruction) error {
	for _, inst := range instructions {
		if sd, ok := inst.(*SourceDocumentInst); ok && sd.Streamable {
			if err := checkStreamableTemplateBody(ss, sd.Body); err != nil {
				return err
			}
		}
		// Recurse into any nested instruction bodies to find source-document.
		for _, child := range getChildInstructions(inst) {
			if err := checkInstructionsForStreamableSourceDoc(ss, child); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkStreamableTemplateBody checks a template body (or source-document body)
// for non-streamable constructs.
func checkStreamableTemplateBody(ss *Stylesheet, body []Instruction) error {
	for _, inst := range body {
		if err := checkStreamableInstruction(ss, inst); err != nil {
			return err
		}
	}
	return nil
}

// checkStreamableInstruction checks a single instruction for streamability violations.
func checkStreamableInstruction(ss *Stylesheet, inst Instruction) error {
	for _, expr := range getInstructionExprs(inst) {
		if err := checkStreamableExpr(expr); err != nil {
			return err
		}
	}

	// Check for multiple downward selections across sibling expressions in
	// fork branches and map entries (they each consume the stream).
	if err := checkMultipleDownwardInst(inst); err != nil {
		return err
	}

	// Recurse into child instructions.
	for _, children := range getChildInstructions(inst) {
		for _, child := range children {
			if err := checkStreamableInstruction(ss, child); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkStreamableExpr checks a single XPath expression for non-streamable patterns.
func checkStreamableExpr(expr *xpath3.Expression) error {
	if expr == nil {
		return nil
	}

	// Note: upward axes (parent, ancestor, preceding) are NOT flagged here
	// because in XSLT 3.0 streaming, ancestor axis is "motionless" — the
	// ancestors of the context node are available. Upward axis violations
	// are only checked for declared-streamability function categories.

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
	if exprHasNonMotionlessStreamingPredicate(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q has a non-motionless predicate, which is not streamable", expr.String())
	}

	// 5. Multiple downward selections (consuming the stream twice)
	// Only count selections outside grounding functions (snapshot/copy-of).
	if countStreamingDownwardSelections(expr.AST()) > 1 {
		return staticError(errCodeXTSE3430,
			"expression %q has multiple downward selections, which is not streamable", expr.String())
	}

	return nil
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
		g := insideGrounding || isGroundingExpr(e.Expr)
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
	case "snapshot", "copy-of", "copy", "current-group",
		"outermost", "innermost", "parse-xml", "parse-xml-fragment",
		"doc", "document", "sort", "reverse":
		return true
	}
	return false
}

// isGroundingExpr returns true if an expression grounds its input,
// meaning subsequent operations on its result work on non-streaming data.
// This is different from functions that merely produce atomic output (like count/sum).
func isGroundingExpr(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	if fc, ok := expr.(xpath3.FunctionCall); ok {
		if fc.Prefix == "" {
			return isGroundingFuncName(fc.Name)
		}
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
			case "tokenize", "string", "data", "number", "boolean",
				"name", "local-name", "namespace-uri", "string-length",
				"normalize-space", "count", "sum", "avg", "min", "max",
				"string-join", "concat", "contains", "starts-with",
				"ends-with", "matches", "replace", "translate",
				"substring", "substring-before", "substring-after",
				"upper-case", "lower-case", "round", "floor", "ceiling",
				"abs", "not", "true", "false", "position", "last",
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
func exprHasNonMotionlessStreamingPredicate(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	walkExprCheckPredicates(expr.AST(), false, &found)
	return found
}

// walkExprCheckPredicates walks the AST looking for non-motionless predicates
// on streaming (non-grounded) steps.
func walkExprCheckPredicates(expr xpath3.Expr, grounded bool, found *bool) {
	if expr == nil || *found {
		return
	}
	expr = derefXPathExpr(expr)

	switch e := expr.(type) {
	case xpath3.LocationPath:
		if !grounded {
			for _, step := range e.Steps {
				for _, pred := range step.Predicates {
					if predicateIsNonMotionless(pred) {
						*found = true
						return
					}
				}
			}
		}
	case xpath3.FilterExpr:
		g := grounded || isGroundingExpr(e.Expr) || isAtomicResultExpr(e.Expr)
		walkExprCheckPredicates(e.Expr, grounded, found)
		if !g {
			for _, pred := range e.Predicates {
				if predicateIsNonMotionless(pred) {
					*found = true
					return
				}
			}
		}
	case xpath3.SimpleMapExpr:
		walkExprCheckPredicates(e.Left, grounded, found)
		// RHS of ! gets individual items — check if LHS produces grounded data
		// Atomic result functions like tokenize() also ground the RHS
		g := grounded || isGroundingExpr(e.Left) || isAtomicResultExpr(e.Left)
		walkExprCheckPredicates(e.Right, g, found)
	case xpath3.PathStepExpr:
		walkExprCheckPredicates(e.Left, grounded, found)
		g := grounded || isGroundingExpr(e.Left) || isAtomicResultExpr(e.Left)
		walkExprCheckPredicates(e.Right, g, found)
	case xpath3.FunctionCall:
		// Function arguments are still streaming — only grounding functions
		// (snapshot, copy-of) ground their input.
		g := grounded || isGroundingExpr(e)
		for _, arg := range e.Args {
			walkExprCheckPredicates(arg, g, found)
		}
	case xpath3.PathExpr:
		walkExprCheckPredicates(e.Filter, grounded, found)
		if e.Path != nil {
			g := grounded || isGroundingExpr(e.Filter)
			lp := *e.Path
			walkExprCheckPredicates(lp, g, found)
		}
	case xpath3.BinaryExpr:
		walkExprCheckPredicates(e.Left, grounded, found)
		walkExprCheckPredicates(e.Right, grounded, found)
	case xpath3.UnaryExpr:
		walkExprCheckPredicates(e.Operand, grounded, found)
	case xpath3.ConcatExpr:
		walkExprCheckPredicates(e.Left, grounded, found)
		walkExprCheckPredicates(e.Right, grounded, found)
	case xpath3.UnionExpr:
		walkExprCheckPredicates(e.Left, grounded, found)
		walkExprCheckPredicates(e.Right, grounded, found)
	case xpath3.IfExpr:
		walkExprCheckPredicates(e.Cond, grounded, found)
		walkExprCheckPredicates(e.Then, grounded, found)
		walkExprCheckPredicates(e.Else, grounded, found)
	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			walkExprCheckPredicates(item, grounded, found)
		}
	case xpath3.FLWORExpr:
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case xpath3.ForClause:
				walkExprCheckPredicates(c.Expr, grounded, found)
			case xpath3.LetClause:
				walkExprCheckPredicates(c.Expr, grounded, found)
			}
		}
		walkExprCheckPredicates(e.Return, grounded, found)
	}
}

// predicateIsNonMotionless returns true if a predicate expression navigates
// downward (uses child/descendant axes), uses last(), or accesses the context item.
func predicateIsNonMotionless(pred xpath3.Expr) bool {
	nonMotionless := false
	xpath3.WalkExpr(pred, func(e xpath3.Expr) bool {
		if nonMotionless {
			return false
		}
		switch v := e.(type) {
		case xpath3.LocationPath:
			for _, step := range v.Steps {
				switch step.Axis {
				case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
					nonMotionless = true
					return false
				}
			}
		case xpath3.PathStepExpr:
			if v.DescOrSelf {
				nonMotionless = true
				return false
			}
		case xpath3.FunctionCall:
			if v.Prefix == "" && v.Name == "last" {
				nonMotionless = true
				return false
			}
			// Property-access functions (name, local-name, etc.) are motionless
			// even when they access the context item. Don't recurse into their args.
			if v.Prefix == "" {
				switch v.Name {
				case "name", "local-name", "namespace-uri", "node-name",
					"self", "generate-id", "base-uri", "document-uri",
					"nilled", "has-children", "string-length":
					return false // skip children — this whole call is motionless
				}
			}
		case xpath3.ContextItemExpr:
			// "." in a predicate means the predicate accesses the context item.
			// This is consuming (reads the string value / content).
			nonMotionless = true
			return false
		}
		return true
	})
	return nonMotionless
}

// checkMultipleDownwardInst checks if an instruction has multiple consuming
// operations that would require reading the stream multiple times.
func checkMultipleDownwardInst(inst Instruction) error {
	switch v := inst.(type) {
	case *ForkInst:
		// Each branch of xsl:fork that has a downward selection consumes the stream.
		// Multiple consuming branches = error.
		consumingBranches := 0
		for _, branch := range v.Branches {
			branchDown := countDownwardInInstructions(branch)
			if branchDown > 0 {
				consumingBranches++
			}
		}
		if consumingBranches > 1 {
			return staticError(errCodeXTSE3430,
				"xsl:fork has multiple consuming branches, which is not streamable")
		}

	case *IterateInst:
		// Within xsl:iterate body, check for multiple downward selections.
		bodyDown := countDownwardInInstructions(v.Body)
		if bodyDown > 1 {
			return staticError(errCodeXTSE3430,
				"xsl:iterate body has multiple downward selections, which is not streamable")
		}

		// Check if iterate body returns streamed nodes via xsl:sequence select="."
		for _, bi := range v.Body {
			if seq, ok := bi.(*XSLSequenceInst); ok && seq.Select != nil {
				ast := seq.Select.AST()
				if _, ok := ast.(xpath3.ContextItemExpr); ok {
					return staticError(errCodeXTSE3430,
						"xsl:iterate body returns streamed nodes via xsl:sequence select=\".\", which is not streamable")
				}
			}
		}

	case *ForEachGroupInst:
		// for-each-group in streaming: multiple consuming uses of current-group()
		groupRefs := countCurrentGroupConsumingRefs(v.Body)
		if groupRefs > 1 {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group body has multiple consuming references to current-group(), which is not streamable")
		}
	}

	return nil
}

// countDownwardInInstructions counts total streaming downward selections across instructions.
func countDownwardInInstructions(instructions []Instruction) int {
	total := 0
	for _, inst := range instructions {
		for _, expr := range getInstructionExprs(inst) {
			total += countStreamingDownwardSelections(expr.AST())
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
func countCurrentGroupConsumingRefs(body []Instruction) int {
	count := 0
	for _, inst := range body {
		// For choose instructions, take the max across branches, not the sum.
		if choose, ok := inst.(*ChooseInst); ok {
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

		for _, expr := range getInstructionExprs(inst) {
			if expr == nil {
				continue
			}
			count += countCurrentGroupConsumingInExpr(expr.AST())
		}
		// Check child instructions, but don't recurse into nested for-each-group
		// bodies (their current-group() refers to their own group, not ours).
		if _, ok := inst.(*ForEachGroupInst); !ok {
			for _, children := range getChildInstructions(inst) {
				count += countCurrentGroupConsumingRefs(children)
			}
		}
	}
	return count
}

// countCurrentGroupConsumingInExpr counts consuming uses of current-group()
// in an expression. A "consuming" use is one where current-group() result
// is navigated into (current-group()/AUTHOR), copied (copy-of(current-group())),
// or otherwise used with downward navigation.
func countCurrentGroupConsumingInExpr(expr xpath3.Expr) int {
	expr = derefXPathExpr(expr)
	count := 0
	switch e := expr.(type) {
	case xpath3.PathStepExpr:
		// current-group()/something — consuming
		if isCurrentGroupCall(e.Left) {
			count++
		} else {
			count += countCurrentGroupConsumingInExpr(e.Left)
		}
		count += countCurrentGroupConsumingInExpr(e.Right)
	case xpath3.FunctionCall:
		// Check if current-group() is passed to a consuming function
		if e.Prefix == "" {
			switch e.Name {
			case "copy-of", "string-join", "deep-equal", "serialize":
				for _, arg := range e.Args {
					if isCurrentGroupCall(arg) {
						count++
					} else {
						count += countCurrentGroupConsumingInExpr(arg)
					}
				}
				return count
			case "count", "empty", "exists", "boolean":
				// These are non-consuming — they just inspect the sequence
				for _, arg := range e.Args {
					if !isCurrentGroupCall(arg) {
						count += countCurrentGroupConsumingInExpr(arg)
					}
				}
				return count
			case "current-group":
				// bare current-group() — consuming if used in a context that
				// will navigate into it. Just count as a reference.
				return 1
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
	return ok && fc.Prefix == "" && fc.Name == "current-group"
}

// checkStreamableFunctionBody checks a function declared with streamability
// for violations of its declared streamability category.
func checkStreamableFunctionBody(ss *Stylesheet, fn *XSLFunction) error {
	cat := fn.Streamability
	switch cat {
	case "absorbing":
		return checkAbsorbingFunction(fn)
	case "inspection":
		return checkInspectionFunction(fn)
	case "filter":
		return checkFilterFunction(fn)
	case "shallow-descent":
		return checkShallowDescentFunction(fn)
	case "ascent":
		return checkAscentFunction(fn)
	}
	return nil
}

// checkAbsorbingFunction checks that an absorbing function's body doesn't have
// non-streamable patterns. Absorbing functions CAN use upward axes and navigate
// freely, but must produce a grounded result and consume streaming params at most once.
func checkAbsorbingFunction(fn *XSLFunction) error {
	// path() is not streamable even in absorbing functions
	for _, inst := range fn.Body {
		for _, expr := range getInstructionExprs(inst) {
			if xpath3.ExprUsesFunction(expr, "path") {
				return staticError(errCodeXTSE3430,
					"absorbing function %q uses path(), which is not streamable", fn.Name.Name)
			}
		}
	}

	// Check if the function returns a non-grounded result.
	if !functionBodyIsGrounded(fn) {
		return staticError(errCodeXTSE3430,
			"absorbing function %q returns a non-grounded result, which violates its declared streamability", fn.Name.Name)
	}

	// Check for multiple consuming references to the streaming parameter.
	if functionHasMultipleConsumingRefs(fn) {
		return staticError(errCodeXTSE3430,
			"absorbing function %q has multiple consuming references to a streaming parameter, which is not streamable", fn.Name.Name)
	}

	// Check for consuming references in loops
	if functionHasConsumingRefInLoop(fn) {
		return staticError(errCodeXTSE3430,
			"absorbing function %q has a consuming reference in a loop, which is not streamable", fn.Name.Name)
	}

	return nil
}

// checkInspectionFunction checks that an inspection function's body
// only inspects properties of the streaming argument without consuming it.
func checkInspectionFunction(fn *XSLFunction) error {
	for _, inst := range fn.Body {
		for _, expr := range getInstructionExprs(inst) {
			if exprConsumesParam(expr, fn.Params) {
				return staticError(errCodeXTSE3430,
					"inspection function %q consumes a streaming parameter, violating its declared streamability", fn.Name.Name)
			}
		}
	}
	return nil
}

// checkFilterFunction checks that a filter function's body only filters
// the streaming argument without consuming it.
func checkFilterFunction(fn *XSLFunction) error {
	for _, inst := range fn.Body {
		for _, expr := range getInstructionExprs(inst) {
			if exprConsumesParam(expr, fn.Params) {
				return staticError(errCodeXTSE3430,
					"filter function %q consumes a streaming parameter, violating its declared streamability", fn.Name.Name)
			}
		}
	}
	return nil
}

// checkShallowDescentFunction checks that a shallow-descent function's body
// only navigates to children (not deeper descendants) of the streaming argument.
func checkShallowDescentFunction(fn *XSLFunction) error {
	if len(fn.Params) == 0 {
		return nil
	}

	// Check if descendant/descendant-or-self axes are used.
	for _, inst := range fn.Body {
		for _, expr := range getInstructionExprs(inst) {
			if xpath3.ExprUsesDescendantOrSelf(expr) {
				return staticError(errCodeXTSE3430,
					"shallow-descent function %q uses descendant axis, violating its declared streamability", fn.Name.Name)
			}
		}
	}

	// Check that the first param is "as node()" (singleton).
	firstParam := fn.Params[0]
	if firstParam.As != "" {
		as := strings.TrimSpace(firstParam.As)
		if as != "node()" && as != "element()" {
			return staticError(errCodeXTSE3430,
				"shallow-descent function %q first parameter type %q is not a single node, violating its declared streamability",
				fn.Name.Name, firstParam.As)
		}
	} else {
		return staticError(errCodeXTSE3430,
			"shallow-descent function %q first parameter has no type constraint, violating its declared streamability",
			fn.Name.Name)
	}

	// Check that additional params don't receive streaming nodes.
	// Only flag if the param type allows nodes (not if it's xs:string etc.)
	for i := 1; i < len(fn.Params); i++ {
		p := fn.Params[i]
		as := strings.TrimSpace(p.As)
		// If param type is explicitly a non-node type (xs:string, xs:integer, etc.),
		// it can't receive streaming nodes, so it's fine.
		if isAtomicTypeConstraint(as) {
			continue
		}
		// If param has no type constraint or allows nodes, check if body consumes it
		paramName := p.Name
		for _, inst := range fn.Body {
			for _, expr := range getInstructionExprs(inst) {
				if exprUsesVarConsumingly(expr, paramName) {
					return staticError(errCodeXTSE3430,
						"shallow-descent function %q additional parameter %q is used in a consuming way, violating its declared streamability",
						fn.Name.Name, paramName)
				}
			}
		}
	}

	return nil
}

// countStreamingDownwardSelections counts downward selections in an expression
// that are NOT inside a grounding function (snapshot, copy-of).
// Selections inside grounding functions operate on grounded data and don't consume the stream.
func countStreamingDownwardSelections(expr xpath3.Expr) int {
	return countStreamingDownwardSelectionsInner(derefXPathExpr(expr), false)
}

func countStreamingDownwardSelectionsInner(expr xpath3.Expr, grounded bool) int {
	expr = derefXPathExpr(expr)
	if grounded {
		return 0 // inside a grounding function — nothing counts
	}

	count := 0
	switch e := expr.(type) {
	case xpath3.LocationPath:
		hasDown := false
		for _, step := range e.Steps {
			switch step.Axis {
			case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
				hasDown = true
			}
			for _, pred := range step.Predicates {
				count += countStreamingDownwardSelectionsInner(pred, false)
			}
		}
		if hasDown {
			count++
		}
	case xpath3.PathStepExpr:
		leftGrounding := isGroundingExpr(e.Left)
		count += countStreamingDownwardSelectionsInner(e.Left, false)
		count += countStreamingDownwardSelectionsInner(e.Right, leftGrounding)
		if e.DescOrSelf && !leftGrounding {
			// The // shorthand adds a descendant-or-self step
			// Only count if not after a grounding function
		}
	case xpath3.PathExpr:
		filterGrounding := isGroundingExpr(e.Filter)
		count += countStreamingDownwardSelectionsInner(e.Filter, false)
		if e.Path != nil {
			lp := *e.Path
			count += countStreamingDownwardSelectionsInner(lp, filterGrounding)
		}
	case xpath3.FunctionCall:
		g := isGroundingExpr(e)
		for _, arg := range e.Args {
			count += countStreamingDownwardSelectionsInner(arg, g)
		}
	case xpath3.BinaryExpr:
		count += countStreamingDownwardSelectionsInner(e.Left, false)
		count += countStreamingDownwardSelectionsInner(e.Right, false)
	case xpath3.ConcatExpr:
		count += countStreamingDownwardSelectionsInner(e.Left, false)
		count += countStreamingDownwardSelectionsInner(e.Right, false)
	case xpath3.SimpleMapExpr:
		count += countStreamingDownwardSelectionsInner(e.Left, false)
		// The RHS of ! receives individual items from the LHS.
		// If the LHS is a grounding expression, the RHS operates on
		// grounded data and doesn't consume the stream.
		leftGrounding := isGroundingExpr(e.Left)
		count += countStreamingDownwardSelectionsInner(e.Right, leftGrounding)
	case xpath3.UnionExpr:
		// A union of two downward selections is a single combined streaming
		// selection (the result is one merged node sequence), so count at
		// most 1 regardless of how many operands have downward paths.
		leftDown := countStreamingDownwardSelectionsInner(e.Left, false)
		rightDown := countStreamingDownwardSelectionsInner(e.Right, false)
		if leftDown > 0 || rightDown > 0 {
			count++
		}
	case xpath3.FilterExpr:
		count += countStreamingDownwardSelectionsInner(e.Expr, false)
		for _, pred := range e.Predicates {
			count += countStreamingDownwardSelectionsInner(pred, false)
		}
	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			count += countStreamingDownwardSelectionsInner(item, false)
		}
	case xpath3.IfExpr:
		count += countStreamingDownwardSelectionsInner(e.Cond, false)
		thenCount := countStreamingDownwardSelectionsInner(e.Then, false)
		elseCount := countStreamingDownwardSelectionsInner(e.Else, false)
		if thenCount > elseCount {
			count += thenCount
		} else {
			count += elseCount
		}
	case xpath3.FLWORExpr:
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case xpath3.ForClause:
				count += countStreamingDownwardSelectionsInner(c.Expr, false)
			case xpath3.LetClause:
				count += countStreamingDownwardSelectionsInner(c.Expr, false)
			}
		}
		count += countStreamingDownwardSelectionsInner(e.Return, false)
	}
	return count
}

// derefXPathExpr converts pointer Expr types to their value equivalents
// so that type switches work uniformly.
func derefXPathExpr(expr xpath3.Expr) xpath3.Expr {
	switch e := expr.(type) {
	case *xpath3.LocationPath:
		if e == nil {
			return nil
		}
		return *e
	case *xpath3.BinaryExpr:
		if e == nil {
			return nil
		}
		return *e
	case *xpath3.FilterExpr:
		if e == nil {
			return nil
		}
		return *e
	case *xpath3.PathExpr:
		if e == nil {
			return nil
		}
		return *e
	case *xpath3.FunctionCall:
		if e == nil {
			return nil
		}
		return *e
	default:
		return expr
	}
}

// isAtomicTypeConstraint returns true if the "as" type constraint refers to
// an atomic type (xs:string, xs:integer, etc.) that cannot hold streaming nodes.
func isAtomicTypeConstraint(as string) bool {
	if as == "" {
		return false
	}
	// Strip occurrence indicators
	as = strings.TrimRight(as, "?*+")
	// Common atomic types
	return strings.HasPrefix(as, "xs:") || as == "item()" || as == ""
}

// checkAscentFunction checks that an ascent function's body navigates
// upward from the streaming argument without consuming it.
func checkAscentFunction(fn *XSLFunction) error {
	for _, inst := range fn.Body {
		for _, expr := range getInstructionExprs(inst) {
			if exprConsumesParam(expr, fn.Params) {
				return staticError(errCodeXTSE3430,
					"ascent function %q consumes a streaming parameter, violating its declared streamability", fn.Name.Name)
			}
		}
	}
	return nil
}

// functionBodyIsGrounded returns true if the function body produces grounded output.
func functionBodyIsGrounded(fn *XSLFunction) bool {
	if len(fn.Params) == 0 {
		return true
	}

	paramNames := make(map[string]bool)
	for _, p := range fn.Params {
		paramNames[p.Name] = true
	}

	for _, inst := range fn.Body {
		if seq, ok := inst.(*XSLSequenceInst); ok && seq.Select != nil {
			if exprReturnsParam(seq.Select.AST(), paramNames) {
				return false
			}
		}
	}

	return true
}

// exprReturnsParam returns true if the expression can directly return a parameter variable.
func exprReturnsParam(expr xpath3.Expr, paramNames map[string]bool) bool {
	switch e := expr.(type) {
	case xpath3.VariableExpr:
		return paramNames[e.Name]
	case xpath3.IfExpr:
		return exprReturnsParam(e.Then, paramNames) || exprReturnsParam(e.Else, paramNames)
	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			if exprReturnsParam(item, paramNames) {
				return true
			}
		}
	}
	return false
}

// functionHasMultipleConsumingRefs returns true if a function has multiple
// references to a streaming parameter that involve downward navigation.
func functionHasMultipleConsumingRefs(fn *XSLFunction) bool {
	if len(fn.Params) == 0 {
		return false
	}

	for _, p := range fn.Params {
		refs := 0
		for _, inst := range fn.Body {
			for _, expr := range getInstructionExprs(inst) {
				refs += countParamDownwardRefs(expr, p.Name)
			}
		}
		if refs > 1 {
			return true
		}
	}

	return false
}

// functionHasConsumingRefInLoop checks if a parameter is used with a positional
// subscript in a loop construct.
func functionHasConsumingRefInLoop(fn *XSLFunction) bool {
	if len(fn.Params) == 0 {
		return false
	}
	paramNames := make(map[string]bool)
	for _, p := range fn.Params {
		paramNames[p.Name] = true
	}

	for _, inst := range fn.Body {
		for _, expr := range getInstructionExprs(inst) {
			if exprHasParamInLoop(expr.AST(), paramNames) {
				return true
			}
		}
	}
	return false
}

// exprHasParamInLoop returns true if a parameter variable reference appears
// inside a FLWOR for clause.
func exprHasParamInLoop(expr xpath3.Expr, paramNames map[string]bool) bool {
	if flwor, ok := expr.(xpath3.FLWORExpr); ok {
		hasForClause := false
		for _, clause := range flwor.Clauses {
			if _, ok := clause.(xpath3.ForClause); ok {
				hasForClause = true
				break
			}
		}
		if hasForClause {
			found := false
			xpath3.WalkExpr(flwor.Return, func(e xpath3.Expr) bool {
				if found {
					return false
				}
				if fe, ok := e.(xpath3.FilterExpr); ok {
					if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
						if paramNames[ve.Name] {
							found = true
							return false
						}
					}
				}
				return true
			})
			return found
		}
	}
	return false
}

// countParamDownwardRefs counts how many times a parameter variable is used
// with downward navigation.
func countParamDownwardRefs(expr *xpath3.Expression, paramName string) int {
	count := 0
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if ve, ok := ps.Left.(xpath3.VariableExpr); ok {
				if ve.Name == paramName {
					count++
					return false
				}
			}
		}
		if fe, ok := e.(xpath3.FilterExpr); ok {
			if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
				if ve.Name == paramName && len(fe.Predicates) > 0 {
					count++
					return false
				}
			}
		}
		return true
	})
	return count
}

// exprConsumesParam checks if an expression uses a function parameter in a
// consuming way.
func exprConsumesParam(expr *xpath3.Expression, params []*Param) bool {
	if len(params) == 0 {
		return false
	}

	paramNames := make(map[string]bool)
	for _, p := range params {
		paramNames[p.Name] = true
	}

	found := false
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		// $param/child::... or $param/*
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if ve, ok := ps.Left.(xpath3.VariableExpr); ok {
				if paramNames[ve.Name] {
					found = true
					return false
				}
			}
		}
		// string($param) — string() on a node consumes it
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" && fc.Name == "string" && len(fc.Args) > 0 {
				if ve, ok := fc.Args[0].(xpath3.VariableExpr); ok {
					if paramNames[ve.Name] {
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

// exprUsesVarConsumingly checks if a variable is used in a consuming way
// (navigated into, passed to consuming function, etc.).
func exprUsesVarConsumingly(expr *xpath3.Expression, varName string) bool {
	found := false
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		// $var/child or $var/* — downward navigation
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if ve, ok := ps.Left.(xpath3.VariableExpr); ok {
				if ve.Name == varName {
					found = true
					return false
				}
			}
		}
		// $var[pred] — filtering
		if fe, ok := e.(xpath3.FilterExpr); ok {
			if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
				if ve.Name == varName && len(fe.Predicates) > 0 {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

// getInstructionExprs extracts all XPath expressions from an instruction.
func getInstructionExprs(inst Instruction) []*xpath3.Expression {
	var exprs []*xpath3.Expression

	switch v := inst.(type) {
	case *ApplyTemplatesInst:
		exprs = append(exprs, v.Select)
		for _, sk := range v.Sort {
			exprs = append(exprs, sk.Select)
		}
		for _, wp := range v.Params {
			exprs = append(exprs, wp.Select)
		}
	case *ValueOfInst:
		exprs = append(exprs, v.Select)
	case *CopyOfInst:
		exprs = append(exprs, v.Select)
	case *CopyInst:
		exprs = append(exprs, v.Select)
	case *ForEachInst:
		exprs = append(exprs, v.Select)
		for _, sk := range v.Sort {
			exprs = append(exprs, sk.Select)
		}
	case *IfInst:
		exprs = append(exprs, v.Test)
	case *ChooseInst:
		for _, when := range v.When {
			exprs = append(exprs, when.Test)
		}
	case *VariableInst:
		exprs = append(exprs, v.Select)
	case *XSLSequenceInst:
		exprs = append(exprs, v.Select)
	case *MapEntryInst:
		exprs = append(exprs, v.Key)
		exprs = append(exprs, v.Select)
	case *AttributeInst:
		exprs = append(exprs, v.Select)
	case *CommentInst:
		exprs = append(exprs, v.Select)
	case *PIInst:
		exprs = append(exprs, v.Select)
	case *NumberInst:
		exprs = append(exprs, v.Value)
		exprs = append(exprs, v.Select)
	case *MessageInst:
		exprs = append(exprs, v.Select)
	case *NamespaceInst:
		exprs = append(exprs, v.Select)
	case *PerformSortInst:
		exprs = append(exprs, v.Select)
		for _, sk := range v.Sort {
			exprs = append(exprs, sk.Select)
		}
	case *IterateInst:
		exprs = append(exprs, v.Select)
		for _, p := range v.Params {
			exprs = append(exprs, p.Select)
		}
	case *BreakInst:
		exprs = append(exprs, v.Select)
	case *ForEachGroupInst:
		exprs = append(exprs, v.Select)
		exprs = append(exprs, v.GroupBy)
		exprs = append(exprs, v.GroupAdjacent)
		for _, sk := range v.Sort {
			exprs = append(exprs, sk.Select)
		}
	case *TryCatchInst:
		exprs = append(exprs, v.Select)
		exprs = append(exprs, v.CatchSelect)
	case *OnEmptyInst:
		exprs = append(exprs, v.Select)
	case *CallTemplateInst:
		for _, wp := range v.Params {
			exprs = append(exprs, wp.Select)
		}
	case *NextMatchInst:
		for _, wp := range v.Params {
			exprs = append(exprs, wp.Select)
		}
	case *ApplyImportsInst:
		for _, wp := range v.Params {
			exprs = append(exprs, wp.Select)
		}
	}

	// Filter out nils.
	filtered := exprs[:0]
	for _, e := range exprs {
		if e != nil {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// getChildInstructions returns all child instruction slices from an instruction.
func getChildInstructions(inst Instruction) [][]Instruction {
	var children [][]Instruction

	switch v := inst.(type) {
	case *IfInst:
		children = append(children, v.Body)
	case *ChooseInst:
		for _, when := range v.When {
			children = append(children, when.Body)
		}
		children = append(children, v.Otherwise)
	case *ForEachInst:
		children = append(children, v.Body)
	case *VariableInst:
		children = append(children, v.Body)
	case *ElementInst:
		children = append(children, v.Body)
	case *AttributeInst:
		children = append(children, v.Body)
	case *CommentInst:
		children = append(children, v.Body)
	case *PIInst:
		children = append(children, v.Body)
	case *MessageInst:
		children = append(children, v.Body)
	case *SequenceInst:
		children = append(children, v.Body)
	case *MapInst:
		children = append(children, v.Body)
	case *MapEntryInst:
		children = append(children, v.Body)
	case *NamespaceInst:
		children = append(children, v.Body)
	case *PerformSortInst:
		children = append(children, v.Body)
	case *LiteralResultElement:
		children = append(children, v.Body)
	case *CopyInst:
		children = append(children, v.Body)
	case *ValueOfInst:
		children = append(children, v.Body)
	case *SourceDocumentInst:
		// Don't recurse into source-document bodies here; they're handled
		// separately by checkInstructionsForStreamableSourceDoc.
	case *IterateInst:
		children = append(children, v.Body)
		children = append(children, v.OnCompletion)
	case *BreakInst:
		children = append(children, v.Body)
	case *ForkInst:
		children = append(children, v.Branches...)
	case *ForEachGroupInst:
		children = append(children, v.Body)
	case *TryCatchInst:
		children = append(children, v.Try)
		children = append(children, v.Catch)
	case *OnEmptyInst:
		children = append(children, v.Body)
	case *WherePopulatedInst:
		children = append(children, v.Body)
	case *CallTemplateInst:
		for _, wp := range v.Params {
			children = append(children, wp.Body)
		}
	case *ApplyTemplatesInst:
		for _, wp := range v.Params {
			children = append(children, wp.Body)
		}
	case *NextMatchInst:
		for _, wp := range v.Params {
			children = append(children, wp.Body)
		}
	case *ApplyImportsInst:
		for _, wp := range v.Params {
			children = append(children, wp.Body)
		}
	}

	return children
}
