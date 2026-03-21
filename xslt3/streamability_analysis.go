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

	// Check templates in streamable modes for non-streamable constructs.
	for _, tmpl := range ss.templates {
		modeName, ok := streamabilityModeNameForTemplate(tmpl)
		if !ok || modeName == "#all" {
			continue
		}
		md := ss.modeDefs[modeName]
		if md == nil || !md.Streamable {
			continue
		}
		if err := checkStreamableTemplateBody(ss, tmpl.Body); err != nil {
			return err
		}
	}

	// Check streamable accumulators: initial-value must be motionless.
	for _, acc := range ss.accumulators {
		if acc.Streamable && acc.Initial != nil {
			if err := checkStreamableExpr(acc.Initial); err != nil {
				return err
			}
			// A motionless expression must not navigate the document at all.
			if xpath3.ExprHasDownwardStep(acc.Initial) {
				return staticError(errCodeXTSE3430,
					"streamable accumulator %q has non-motionless initial-value expression %q",
					acc.Name, acc.Initial.String())
			}
		}
	}

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

// streamabilityModeNameForTemplate returns the mode name that should be used
// for streamability checks. Only match templates participate in mode-based
// streamability; named templates are handled separately where relevant.
func streamabilityModeNameForTemplate(tmpl *Template) (string, bool) {
	if tmpl == nil || tmpl.Match == nil {
		return "", false
	}
	if tmpl.Mode == "" {
		return "#default", true
	}
	return tmpl.Mode, true
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

	// Check for variables bound to the streaming context that are used
	// consumingly in subsequent loop bodies.
	if err := checkStreamingVarInLoop(body); err != nil {
		return err
	}

	return nil
}

// checkStreamableInstruction checks a single instruction for streamability violations.
func checkStreamableInstruction(ss *Stylesheet, inst Instruction) error {
	switch v := inst.(type) {
	case *ApplyTemplatesInst:
		if v.Select != nil && xpath3.ExprUsesDescendantOrSelf(v.Select) &&
			!exprEndsWithGrounding(v.Select.AST()) {
			return staticError(errCodeXTSE3430,
				"xsl:apply-templates with crawling select expression %q is not streamable", v.Select.String())
		}
	case *NumberInst:
		// xsl:number without an explicit value computes numbering from node
		// relationships, which is consuming and therefore not streamable.
		if v.Value == nil {
			return staticError(errCodeXTSE3430,
				"xsl:number without value is not streamable")
		}
	}

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

	// Check attribute sets used in streaming context.
	if err := checkUseAttributeSetsStreamable(ss, inst); err != nil {
		return err
	}

	// Check for-each/iterate with crawling select or variable-bound streaming in loop.
	if err := checkForEachStreamable(ss, inst); err != nil {
		return err
	}

	// Check for-each-group streamability constraints.
	if err := checkForEachGroupStreamable(ss, inst); err != nil {
		return err
	}

	// Check xsl:map with multiple consuming entries or non-map-entry children.
	if err := checkMapStreamable(ss, inst); err != nil {
		return err
	}

	// Check xsl:iterate for streamed node accumulation violations.
	if err := checkIterateStreamable(ss, inst); err != nil {
		return err
	}

	// Recurse into child instructions.
	// For xsl:for-each whose select does NOT consume the stream (motionless/upward),
	// skip body checks: the body operates in a new, non-streaming context where
	// last(), position(), etc. are allowed.
	if fe, ok := inst.(*ForEachInst); ok {
		if fe.Select != nil && !xpath3.ExprHasDownwardStep(fe.Select) && !xpath3.ExprUsesDescendantOrSelf(fe.Select) {
			// Body of for-each over motionless/attribute axis — skip streaming checks.
			return nil
		}
	}
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

	// 6. Grounding functions (reverse, innermost) with crawling operands are not
	// streamable because the crawling expression itself requires multi-level descent.
	if exprHasCrawlingGroundingArg(expr) {
		return staticError(errCodeXTSE3430,
			"expression %q uses a grounding function with a crawling operand, which is not streamable", expr.String())
	}

	return nil
}

// exprHasCrawlingGroundingArg returns true if the expression contains a call to
// reverse(), innermost(), or outermost() whose argument has a crawling expression
// on streaming (non-grounded) data.
func exprHasCrawlingGroundingArg(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" && (fc.Name == "reverse" || fc.Name == "innermost") {
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

		// Also check if a single current-group() usage has multiple downward
		// selections (e.g. current-group()/(AUTHOR||TITLE)).
		for _, bi := range v.Body {
			for _, expr := range getInstructionExprs(bi) {
				if expr == nil {
					continue
				}
				if countDownwardFromCurrentGroup(expr.AST()) > 1 {
					return staticError(errCodeXTSE3430,
						"xsl:for-each-group body has multiple downward selections from current-group(), which is not streamable")
				}
			}
			for _, children := range getChildInstructions(bi) {
				for _, ci := range children {
					for _, expr := range getInstructionExprs(ci) {
						if expr == nil {
							continue
						}
						if countDownwardFromCurrentGroup(expr.AST()) > 1 {
							return staticError(errCodeXTSE3430,
								"xsl:for-each-group body has multiple downward selections from current-group(), which is not streamable")
						}
					}
				}
			}
		}

		// Check if context item "." is used consumingly in for-each-group body.
		// In for-each-group, "." refers to the first item of the current group,
		// and using both "." (downward) and current-group() is multiple consumption.
		bodyUsesContextDown := false
		bodyUsesCurrentGroup := false
		for _, bi := range v.Body {
			for _, expr := range getInstructionExprs(bi) {
				if expr == nil {
					continue
				}
				if exprHasContextDownward(expr.AST()) {
					bodyUsesContextDown = true
				}
				if xpath3.ExprUsesFunction(expr, "current-group") {
					bodyUsesCurrentGroup = true
				}
			}
			for _, children := range getChildInstructions(bi) {
				for _, ci := range children {
					for _, expr := range getInstructionExprs(ci) {
						if expr == nil {
							continue
						}
						if exprHasContextDownward(expr.AST()) {
							bodyUsesContextDown = true
						}
						if xpath3.ExprUsesFunction(expr, "current-group") {
							bodyUsesCurrentGroup = true
						}
					}
				}
			}
		}
		if bodyUsesContextDown && bodyUsesCurrentGroup {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group body uses both context item (downward) and current-group(), which is not streamable")
		}
	}

	return nil
}

// checkUseAttributeSetsStreamable checks that attribute sets used in a streaming
// context are themselves streamable. An attribute set is non-streamable if any
// of its attribute instructions contain non-streamable expressions (downward
// navigation, last(), etc.), or if the combined downward selections from the
// instruction and attribute set exceed 1.
func checkUseAttributeSetsStreamable(ss *Stylesheet, inst Instruction) error {
	var attrSetNames []string
	switch v := inst.(type) {
	case *CopyInst:
		attrSetNames = v.UseAttrSets
	case *ElementInst:
		attrSetNames = v.UseAttrSets
	case *LiteralResultElement:
		attrSetNames = v.UseAttrSets
	}
	if len(attrSetNames) == 0 {
		return nil
	}

	// Count downward selections in the instruction's own expressions
	instDown := 0
	for _, expr := range getInstructionExprs(inst) {
		instDown += countStreamingDownwardSelections(expr.AST())
	}

	for _, name := range attrSetNames {
		asDef := ss.attributeSets[name]
		if asDef == nil {
			continue
		}

		// Check individual expressions for streamability violations
		for _, attrInst := range asDef.Attrs {
			for _, expr := range getInstructionExprs(attrInst) {
				if err := checkStreamableExpr(expr); err != nil {
					return staticError(errCodeXTSE3430,
						"attribute set %q is not streamable: %s", asDef.Name, err.Error())
				}
			}
		}

		// Any attribute set with downward navigation used in streaming context
		// is non-streamable (the attribute set would consume the stream).
		asDown := countAttributeSetDownward(ss, asDef)
		if asDown > 0 {
			return staticError(errCodeXTSE3430,
				"use-attribute-sets %q has downward navigation, which is not streamable", name)
		}

		// Check transitively-used attribute sets
		for _, usedName := range asDef.UseAttrSets {
			used := ss.attributeSets[usedName]
			if used == nil {
				continue
			}
			for _, attrInst := range used.Attrs {
				for _, expr := range getInstructionExprs(attrInst) {
					if err := checkStreamableExpr(expr); err != nil {
						return staticError(errCodeXTSE3430,
							"attribute set %q is not streamable: %s", usedName, err.Error())
					}
				}
			}
		}
	}
	return nil
}

// countAttributeSetDownward counts total streaming downward selections in an
// attribute set's instructions.
func countAttributeSetDownward(ss *Stylesheet, asDef *AttributeSetDef) int {
	total := 0
	for _, attrInst := range asDef.Attrs {
		for _, expr := range getInstructionExprs(attrInst) {
			total += countStreamingDownwardSelections(expr.AST())
		}
	}
	// Include transitively used attribute sets
	for _, usedName := range asDef.UseAttrSets {
		used := ss.attributeSets[usedName]
		if used != nil {
			total += countAttributeSetDownward(ss, used)
		}
	}
	return total
}

// checkForEachStreamable checks for-each and iterate instructions for:
// - crawling select expression (//a/b = descendant-or-self then child)
// - variable bound to streamed node used consumingly in loop body
// - xsl:sequence select="." returning streamed nodes from for-each body
func checkForEachStreamable(_ *Stylesheet, inst Instruction) error {
	switch v := inst.(type) {
	case *ForEachInst:
		// Check if select expression is crawling AND body consumes context.
		// But if the select grounds its result (e.g., snapshot(//...)), the body
		// operates on grounded data and consuming is fine.
		if v.Select != nil && xpath3.ExprUsesDescendantOrSelf(v.Select) &&
			!exprEndsWithGrounding(v.Select.AST()) {
			if forEachBodyConsumesContext(v.Body) {
				return staticError(errCodeXTSE3430,
					"xsl:for-each with crawling select expression %q and consuming body is not streamable", v.Select.String())
			}
		}
		// Check for xsl:sequence select="." returning streamed nodes
		for _, bi := range v.Body {
			if seq, ok := bi.(*XSLSequenceInst); ok && seq.Select != nil {
				ast := seq.Select.AST()
				if _, ok := ast.(xpath3.ContextItemExpr); ok {
					return staticError(errCodeXTSE3430,
						"xsl:for-each body returns streamed nodes via xsl:sequence select=\".\", which is not streamable")
				}
			}
		}
		// Check for variable bound to streaming context used in loop body
		if err := checkStreamingVarInLoop(v.Body); err != nil {
			return err
		}
	case *IterateInst:
		// Check for variable bound to streaming context used in loop body
		if err := checkStreamingVarInLoop(v.Body); err != nil {
			return err
		}
	}
	return nil
}

// forEachBodyConsumesContext returns true if any instruction in the body
// consumes the context item (e.g., value-of select="."). This is checked
// when a for-each has a crawling select expression.
func forEachBodyConsumesContext(body []Instruction) bool {
	for _, inst := range body {
		for _, expr := range getInstructionExprs(inst) {
			if expr == nil {
				continue
			}
			// Check if the expression accesses "." (context item) — consuming it
			if exprReferencesContextItem(expr.AST()) {
				return true
			}
		}
		for _, children := range getChildInstructions(inst) {
			if forEachBodyConsumesContext(children) {
				return true
			}
		}
	}
	return false
}

// checkStreamingVarInLoop checks if a variable is bound to the streaming context
// node (select=".") and then used consumingly (downward navigation) in a loop body.
// This is non-streamable because the loop iterates over a non-streaming range but
// accesses the streamed variable repeatedly.
func checkStreamingVarInLoop(body []Instruction) error {
	// Collect variables bound to streaming context (select=".")
	streamingVars := make(map[string]bool)
	for _, inst := range body {
		if vi, ok := inst.(*VariableInst); ok && vi.Select != nil {
			ast := vi.Select.AST()
			if _, ok := ast.(xpath3.ContextItemExpr); ok {
				streamingVars[vi.Name] = true
			}
		}
	}
	if len(streamingVars) == 0 {
		return nil
	}
	// Check if any subsequent for-each/iterate loop uses these vars consumingly
	for _, inst := range body {
		switch v := inst.(type) {
		case *ForEachInst:
			if err := checkVarConsumingInBody(streamingVars, v.Body); err != nil {
				return err
			}
		case *IterateInst:
			if err := checkVarConsumingInBody(streamingVars, v.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkVarConsumingInBody checks if any streaming variable is used consumingly
// in a set of instructions.
func checkVarConsumingInBody(streamingVars map[string]bool, body []Instruction) error {
	for _, inst := range body {
		for _, expr := range getInstructionExprs(inst) {
			if expr == nil {
				continue
			}
			for varName := range streamingVars {
				if exprUsesVarConsumingly(expr, varName) {
					return staticError(errCodeXTSE3430,
						"variable $%s bound to streaming context is used consumingly in a loop, which is not streamable", varName)
				}
			}
		}
		for _, children := range getChildInstructions(inst) {
			if err := checkVarConsumingInBody(streamingVars, children); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkForEachGroupStreamable checks for-each-group specific streamability constraints:
// - group-by/group-adjacent with consuming grouping expression
// - group-starting-with/group-ending-with with consuming pattern
// - sorted for-each-group in streaming context
// - nested for-each-group consuming current-group()
// - for-each-group inside fork with crawling select
// - for-each-group with non-motionless grouping key (e.g., PRICE/text())
func checkForEachGroupStreamable(_ *Stylesheet, inst Instruction) error {
	fg, ok := inst.(*ForEachGroupInst)
	if !ok {
		return nil
	}

	// Check if group-adjacent with position() in grouping key
	if fg.GroupAdjacent != nil {
		if xpath3.ExprUsesFunction(fg.GroupAdjacent, "position") {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group group-adjacent uses position(), which is not streamable")
		}
	}

	// Check if sort is used with for-each-group in streaming (non-streamable)
	if len(fg.Sort) > 0 {
		return staticError(errCodeXTSE3430,
			"xsl:for-each-group with sort is not streamable")
	}

	// Check if grouping key navigates downward (e.g., PRICE/text())
	if fg.GroupBy != nil && xpath3.ExprHasDownwardStep(fg.GroupBy) {
		return staticError(errCodeXTSE3430,
			"xsl:for-each-group group-by expression %q navigates downward, which is not streamable",
			fg.GroupBy.String())
	}
	if fg.GroupAdjacent != nil && xpath3.ExprHasDownwardStep(fg.GroupAdjacent) {
		return staticError(errCodeXTSE3430,
			"xsl:for-each-group group-adjacent expression %q navigates downward, which is not streamable",
			fg.GroupAdjacent.String())
	}

	// Check if select uses descendant-or-self (crawling) for grouped streaming,
	// but only when the select doesn't ground its result via copy-of/snapshot.
	if fg.Select != nil && xpath3.ExprUsesDescendantOrSelf(fg.Select) && !exprEndsWithGrounding(fg.Select.AST()) {
		return staticError(errCodeXTSE3430,
			"xsl:for-each-group select expression %q is crawling, which is not streamable", fg.Select.String())
	}

	// Check for nested for-each-group that consumes current-group(),
	// but only when the outer for-each-group select doesn't ground its data.
	if fg.Select == nil || !exprEndsWithGrounding(fg.Select.AST()) {
		for _, bi := range fg.Body {
			if innerFg, ok := bi.(*ForEachGroupInst); ok {
				if innerFg.Select != nil {
					if exprUsesCurrentGroup(innerFg.Select) {
						// Nested for-each-group selecting from current-group() is consuming
						cgRefs := countCurrentGroupConsumingRefs(innerFg.Body)
						if cgRefs > 0 {
							return staticError(errCodeXTSE3430,
								"nested xsl:for-each-group consuming current-group() is not streamable")
						}
					}
				}
			}
		}
	}

	// Check if for-each-group body has source-document with streaming that uses current-group()
	for _, bi := range fg.Body {
		if sd, ok := bi.(*SourceDocumentInst); ok && sd.Streamable {
			for _, sdi := range sd.Body {
				for _, expr := range getInstructionExprs(sdi) {
					if exprUsesCurrentGroup(expr) {
						return staticError(errCodeXTSE3430,
							"xsl:source-document streamable body uses current-group() from outer for-each-group, which is not streamable")
					}
				}
			}
		}
	}

	// Check for-each-group with group-starting-with: copy-of(current-group()) is not streamable
	// because group-starting-with needs to buffer and the pattern-matching consumes nodes
	if fg.GroupStartingWith != nil {
		for _, bi := range fg.Body {
			if err := checkGroupStartingWithBody(bi); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkGroupStartingWithBody checks if a for-each-group group-starting-with body
// uses current-group() in a consuming way that requires buffering.
func checkGroupStartingWithBody(inst Instruction) error {
	for _, expr := range getInstructionExprs(inst) {
		if expr == nil {
			continue
		}
		if exprUsesCurrentGroupConsumingly(expr) {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group group-starting-with body consumes current-group(), which is not streamable")
		}
	}
	for _, children := range getChildInstructions(inst) {
		for _, child := range children {
			if err := checkGroupStartingWithBody(child); err != nil {
				return err
			}
		}
	}
	return nil
}

// exprUsesCurrentGroupConsumingly checks if an expression uses current-group()
// in a consuming way (copy-of, downward navigation, apply-templates, etc.).
func exprUsesCurrentGroupConsumingly(expr *xpath3.Expression) bool {
	found := false
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		// copy-of(current-group()) is consuming
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" && (fc.Name == "copy-of" || fc.Name == "snapshot") {
				for _, arg := range fc.Args {
					if isCurrentGroupCall(arg) {
						found = true
						return false
					}
				}
			}
		}
		// current-group()/something is consuming
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if isCurrentGroupCall(ps.Left) {
				found = true
				return false
			}
		}
		// apply-templates select="current-group()" is consuming (can't check from XPath,
		// but current-group() passed to apply-templates will appear in instruction exprs)
		return true
	})
	return found
}

// exprUsesCurrentGroup returns true if the expression calls current-group() anywhere.
func exprUsesCurrentGroup(expr *xpath3.Expression) bool {
	return xpath3.ExprUsesFunction(expr, "current-group")
}

// checkMapStreamable checks xsl:map instructions for streamability violations:
// - Multiple map-entry children with consuming expressions
// - Non-map-entry children (like xsl:if wrapping map-entry)
func checkMapStreamable(_ *Stylesheet, inst Instruction) error {
	mapInst, ok := inst.(*MapInst)
	if !ok {
		return nil
	}

	consumingEntries := 0
	hasNonEntryChildren := false
	for _, child := range mapInst.Body {
		if me, ok := child.(*MapEntryInst); ok {
			consuming := false
			if me.Select != nil {
				if countStreamingDownwardSelections(me.Select.AST()) > 0 {
					consuming = true
				}
			}
			if me.Key != nil {
				if countStreamingDownwardSelections(me.Key.AST()) > 0 {
					consuming = true
				}
			}
			// Also check child instructions of map-entry for consuming patterns
			if !consuming {
				for _, meChild := range me.Body {
					for _, expr := range getInstructionExprs(meChild) {
						if countStreamingDownwardSelections(expr.AST()) > 0 {
							consuming = true
							break
						}
					}
					if consuming {
						break
					}
				}
			}
			if consuming {
				consumingEntries++
			}
		} else {
			hasNonEntryChildren = true
		}
	}

	if consumingEntries > 1 {
		return staticError(errCodeXTSE3430,
			"xsl:map has multiple map-entry elements with consuming expressions, which is not streamable")
	}

	if hasNonEntryChildren && consumingEntries > 0 {
		return staticError(errCodeXTSE3430,
			"xsl:map has non-map-entry children with consuming map entries, which is not streamable")
	}

	return nil
}

// checkIterateStreamable checks xsl:iterate for violations where streamed nodes
// are accumulated via xsl:with-param, creating non-streamable patterns.
func checkIterateStreamable(_ *Stylesheet, inst Instruction) error {
	iter, ok := inst.(*IterateInst)
	if !ok {
		return nil
	}

	// Check if any iterate param is typed as element()* and the body uses
	// the context item (.) in a with-param, which means streamed nodes accumulate.
	// This pattern: <xsl:param name="x" as="element()*"/>
	//               <xsl:with-param name="x" select="($x, .)"/>
	// is non-streamable because it accumulates streamed nodes across iterations.
	elemParams := make(map[string]bool)
	for _, p := range iter.Params {
		as := strings.TrimSpace(p.As)
		if strings.Contains(as, "element") && (strings.Contains(as, "*") || strings.Contains(as, "+")) {
			elemParams[p.Name] = true
		}
	}
	if len(elemParams) == 0 {
		return nil
	}

	// Look for next-iteration with-param that includes "." (context item)
	// This is the pattern where streamed nodes accumulate
	if bodyAccumulatesStreamedNodes(iter.Body, elemParams) {
		return staticError(errCodeXTSE3430,
			"xsl:iterate accumulates streamed nodes in element()* parameter, which is not streamable")
	}

	return nil
}

// bodyAccumulatesStreamedNodes checks if an iterate body accumulates streamed
// nodes by passing context items to element()* parameters via next-iteration.
func bodyAccumulatesStreamedNodes(body []Instruction, elemParams map[string]bool) bool {
	for _, inst := range body {
		if ni, ok := inst.(*NextIterationInst); ok {
			for _, wp := range ni.Params {
				if !elemParams[wp.Name] {
					continue
				}
				if wp.Select != nil && exprReferencesContextItem(wp.Select.AST()) {
					return true
				}
			}
		}
		// Recurse into child instructions (e.g., inside choose/if)
		for _, children := range getChildInstructions(inst) {
			if bodyAccumulatesStreamedNodes(children, elemParams) {
				return true
			}
		}
	}
	return false
}

// exprReferencesContextItem returns true if the expression references "." (context item).
func exprReferencesContextItem(expr xpath3.Expr) bool {
	found := false
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if _, ok := e.(xpath3.ContextItemExpr); ok {
			found = true
			return false
		}
		return true
	})
	return found
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
// in an expression. In streaming, any use of current-group() that requires
// enumerating the group members is consuming. This includes count(),
// current-group()/AUTHOR, copy-of(current-group()), etc.
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
		if e.Prefix == "" {
			switch e.Name {
			case "current-group":
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
	case xpath3.FunctionCall:
		return isGroundingExpr(e)
	case xpath3.LocationPath:
		// A LocationPath ending with a grounding step? Not typical.
		// Check last step for function-like patterns.
		return false
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

// exprHasContextDownward returns true if the expression accesses child/descendant
// steps from the context item (not from a variable or function call).
func exprHasContextDownward(expr xpath3.Expr) bool {
	found := false
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if lp, ok := e.(xpath3.LocationPath); ok {
			for _, step := range lp.Steps {
				switch step.Axis {
				case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
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
	// First param must be a singleton node (node()?), not a sequence (node()*).
	if err := checkStreamingParamIsSingleton(fn, "inspection"); err != nil {
		return err
	}

	// Inspection must not return the streaming parameter itself.
	if functionReturnsStreamingParam(fn) {
		return staticError(errCodeXTSE3430,
			"inspection function %q returns a streaming parameter, violating its declared streamability", fn.Name.Name)
	}

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
	// First param must be a singleton node (node()?), not a sequence (node()*).
	if err := checkStreamingParamIsSingleton(fn, "filter"); err != nil {
		return err
	}

	// Filter function must not navigate upward from param (return climbing nodes).
	if functionUsesUpwardFromParam(fn) {
		return staticError(errCodeXTSE3430,
			"filter function %q navigates upward from streaming parameter, violating its declared streamability", fn.Name.Name)
	}

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
	// For shallow-descent, additional parameters that could receive nodes
	// are not allowed because the caller might pass streaming nodes.
	for i := 1; i < len(fn.Params); i++ {
		p := fn.Params[i]
		as := strings.TrimSpace(p.As)
		// If param type is explicitly a non-node type (xs:string, xs:integer, etc.),
		// it can't receive streaming nodes, so it's fine.
		if isAtomicTypeConstraint(as) {
			continue
		}
		// Additional parameter with no type or node type is not allowed
		// for shallow-descent because the caller could pass streaming nodes.
		return staticError(errCodeXTSE3430,
			"shallow-descent function %q additional parameter %q could receive streaming nodes, violating its declared streamability",
			fn.Name.Name, p.Name)
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
		// A path step like A/B is a single selection path — the left side
		// provides context for the right side. Count the right side's
		// distinct downward selections; if the left side also has downward
		// steps, they combine into one path, not two separate paths.
		leftCount := countStreamingDownwardSelectionsInner(e.Left, false)
		rightCount := countStreamingDownwardSelectionsInner(e.Right, leftGrounding)
		if leftCount > 0 && rightCount > 0 {
			// Left and right form a single combined path.
			// The right side may have multiple branches (e.g., sequence expressions).
			// Count as: the max of (left alone as 1 path) + (right branches - 1).
			// For chapter/(chtitle, @nr): left=1 (chapter), right=1 (chtitle only).
			// Result: 1 path, not 2.
			count += rightCount
		} else {
			count += leftCount + rightCount
		}
		if e.DescOrSelf && !leftGrounding {
			// The // shorthand adds a descendant-or-self step
			// Only count if not after a grounding function
			count++
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
	// First param must be a singleton node (node()?), not a sequence (node()*).
	if err := checkStreamingParamIsSingleton(fn, "ascent"); err != nil {
		return err
	}

	// Ascent must not return the streaming parameter itself.
	if functionReturnsStreamingParam(fn) {
		return staticError(errCodeXTSE3430,
			"ascent function %q returns a streaming parameter, violating its declared streamability", fn.Name.Name)
	}

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

// checkStreamingParamIsSingleton checks that the first parameter of a streamable
// function accepts a singleton node (node()?), not a sequence (node()*).
// Functions declared with streamability categories like filter, inspection, ascent,
// and absorbing must receive a single streaming node, not a sequence.
func checkStreamingParamIsSingleton(fn *XSLFunction, category string) error {
	if len(fn.Params) == 0 {
		return nil
	}
	firstParam := fn.Params[0]
	as := strings.TrimSpace(firstParam.As)
	// node()* or node()+ — sequence of nodes — not allowed for streaming
	if as != "" && (strings.HasSuffix(as, "*") || strings.HasSuffix(as, "+")) {
		base := strings.TrimRight(as, "*+")
		// Only flag if it's a node type (not xs:string*, etc.)
		if !strings.HasPrefix(base, "xs:") {
			return staticError(errCodeXTSE3430,
				"%s function %q first parameter type %q accepts a sequence, violating its declared streamability",
				category, fn.Name.Name, firstParam.As)
		}
	}
	return nil
}

// functionReturnsStreamingParam returns true if the function body can return
// the streaming parameter directly (not wrapped in a grounding function).
func functionReturnsStreamingParam(fn *XSLFunction) bool {
	if len(fn.Params) == 0 {
		return false
	}
	// If the function has a return type that is an atomic type, it can't return nodes.
	if fn.As != "" {
		as := strings.TrimRight(strings.TrimSpace(fn.As), "?*+")
		if strings.HasPrefix(as, "xs:") {
			return false
		}
	}

	paramNames := make(map[string]bool)
	for _, p := range fn.Params {
		paramNames[p.Name] = true
	}

	for _, inst := range fn.Body {
		if seq, ok := inst.(*XSLSequenceInst); ok && seq.Select != nil {
			if exprReturnsParam(seq.Select.AST(), paramNames) {
				return true
			}
		}
	}
	return false
}

// functionUsesUpwardFromParam returns true if the function navigates upward
// (parent/ancestor) from a streaming parameter.
func functionUsesUpwardFromParam(fn *XSLFunction) bool {
	if len(fn.Params) == 0 {
		return false
	}
	paramNames := make(map[string]bool)
	for _, p := range fn.Params {
		paramNames[p.Name] = true
	}

	for _, inst := range fn.Body {
		for _, expr := range getInstructionExprs(inst) {
			if expr == nil {
				continue
			}
			found := false
			xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
				if found {
					return false
				}
				// $param/.. or $param/parent::* (PathStepExpr form)
				if ps, ok := e.(xpath3.PathStepExpr); ok {
					if ve, ok := ps.Left.(xpath3.VariableExpr); ok {
						if paramNames[ve.Name] {
							if exprHasUpwardAxis(ps.Right) {
								found = true
								return false
							}
						}
					}
				}
				// $param/.. (PathExpr form: FilterExpr($param) / LocationPath(..))
				if pe, ok := e.(xpath3.PathExpr); ok {
					if fe, ok := pe.Filter.(xpath3.FilterExpr); ok {
						if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
							if paramNames[ve.Name] && pe.Path != nil {
								lp := *pe.Path
								if exprHasUpwardAxis(lp) {
									found = true
									return false
								}
							}
						}
					}
				}
				return true
			})
			if found {
				return true
			}
		}
	}
	return false
}

// exprHasUpwardAxis returns true if the expression contains any upward axis step.
func exprHasUpwardAxis(expr xpath3.Expr) bool {
	found := false
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if lp, ok := e.(xpath3.LocationPath); ok {
			for _, step := range lp.Steps {
				switch step.Axis {
				case xpath3.AxisParent, xpath3.AxisAncestor, xpath3.AxisAncestorOrSelf:
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
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
// in a consuming way (downward navigation, function arguments, etc.).
func countParamDownwardRefs(expr *xpath3.Expression, paramName string) int {
	count := 0
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		// $param/child or $param/* (PathStepExpr)
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if ve, ok := ps.Left.(xpath3.VariableExpr); ok {
				if ve.Name == paramName {
					count++
					return false
				}
			}
		}
		// $param/child or $param/* (PathExpr)
		if pe, ok := e.(xpath3.PathExpr); ok {
			if fe, ok := pe.Filter.(xpath3.FilterExpr); ok {
				if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
					if ve.Name == paramName && pe.Path != nil {
						count++
						return false
					}
				}
			}
		}
		// $param[pred] — filtering
		if fe, ok := e.(xpath3.FilterExpr); ok {
			if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
				if ve.Name == paramName && len(fe.Predicates) > 0 {
					count++
					return false
				}
			}
		}
		// head($param), tail($param), etc. — consuming functions
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" {
				switch fc.Name {
				case "head", "tail", "copy-of", "snapshot", "string-join",
					"serialize", "deep-equal", "sort", "reverse":
					for _, arg := range fc.Args {
						if ve, ok := arg.(xpath3.VariableExpr); ok {
							if ve.Name == paramName {
								count++
								return false
							}
						}
					}
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
		// $param[predicate] — predicate accessing context consumes it
		if fe, ok := e.(xpath3.FilterExpr); ok {
			if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
				if paramNames[ve.Name] && len(fe.Predicates) > 0 {
					// Check if any predicate is non-motionless (accesses "." etc.)
					for _, pred := range fe.Predicates {
						if predicateIsNonMotionless(pred) {
							found = true
							return false
						}
					}
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
		// $var/child or $var/* — downward navigation (PathStepExpr form)
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if ve, ok := ps.Left.(xpath3.VariableExpr); ok {
				if ve.Name == varName {
					found = true
					return false
				}
			}
		}
		// $var/child or $var/* — downward navigation (PathExpr form)
		if pe, ok := e.(xpath3.PathExpr); ok {
			if fe, ok := pe.Filter.(xpath3.FilterExpr); ok {
				if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
					if ve.Name == varName && pe.Path != nil {
						found = true
						return false
					}
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
		for _, c := range v.Catches {
			exprs = append(exprs, c.Select)
		}
	case *OnEmptyInst:
		exprs = append(exprs, v.Select)
	case *OnNonEmptyInst:
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
	case *AnalyzeStringInst:
		exprs = append(exprs, v.Select)
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
	case *ResultDocumentInst:
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
		for _, c := range v.Catches {
			children = append(children, c.Body)
		}
	case *OnEmptyInst:
		children = append(children, v.Body)
	case *OnNonEmptyInst:
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
	case *AnalyzeStringInst:
		children = append(children, v.MatchingBody)
		children = append(children, v.NonMatchingBody)
	}

	return children
}
