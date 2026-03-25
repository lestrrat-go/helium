package xslt3

import (
	"strings"

	"github.com/lestrrat-go/helium/xpath3"
)

func checkStreamableFunctionBody(ss *Stylesheet, fn *xslFunction) error {
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
func checkAbsorbingFunction(fn *xslFunction) error {
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

	// Check that additional params don't receive streaming nodes.
	if err := checkAdditionalParamsNotStreaming(fn, "absorbing"); err != nil {
		return err
	}

	return nil
}

// checkInspectionFunction checks that an inspection function's body
// only inspects properties of the streaming argument without consuming it.
func checkInspectionFunction(fn *xslFunction) error {
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

	// Check that additional params don't receive streaming nodes.
	if err := checkAdditionalParamsNotStreaming(fn, "inspection"); err != nil {
		return err
	}

	return nil
}

// checkFilterFunction checks that a filter function's body only filters
// the streaming argument without consuming it.
func checkFilterFunction(fn *xslFunction) error {
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

	// Check that additional params don't receive streaming nodes.
	if err := checkAdditionalParamsNotStreaming(fn, "filter"); err != nil {
		return err
	}

	return nil
}

// checkShallowDescentFunction checks that a shallow-descent function's body
// only navigates to children (not deeper descendants) of the streaming argument.
func checkShallowDescentFunction(fn *xslFunction) error {
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
	if err := checkAdditionalParamsNotStreaming(fn, "shallow-descent"); err != nil {
		return err
	}

	// Check that the parameter is not used in a simple mapping expression,
	// which would cause multiple consumption. E.g., (1 to 5) ! $n
	if functionHasParamInSimpleMap(fn) {
		return staticError(errCodeXTSE3430,
			"shallow-descent function %q uses streaming parameter in simple mapping expression, violating its declared streamability",
			fn.Name.Name)
	}

	return nil
}

// countStreamingDownwardSelections counts downward selections in an expression
// that are NOT inside a grounding function (snapshot, copy-of).
// Selections inside grounding functions operate on grounded data and don't consume the stream.
func countStreamingDownwardSelections(ss *Stylesheet, expr xpath3.Expr) int {
	return countStreamingDownwardSelectionsInner(ss, derefXPathExpr(expr), false)
}

func countStreamingDownwardSelectionsInner(ss *Stylesheet, expr xpath3.Expr, grounded bool) int {
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
				count += countStreamingDownwardSelectionsInner(ss, pred, false)
			}
		}
		if hasDown {
			count++
		}
	case xpath3.PathStepExpr:
		// A path rooted in a variable reference (e.g., $var/text()) navigates
		// the variable's value, not the streaming source. Return 0.
		if _, isVar := derefXPathExpr(e.Left).(xpath3.VariableExpr); isVar {
			return 0
		}
		leftGrounding := isGroundingExprSS(ss, e.Left)
		// A path step like A/B is a single selection path — the left side
		// provides context for the right side. Count the right side's
		// distinct downward selections; if the left side also has downward
		// steps, they combine into one path, not two separate paths.
		leftCount := countStreamingDownwardSelectionsInner(ss, e.Left, false)
		rightCount := countStreamingDownwardSelectionsInner(ss, e.Right, leftGrounding)
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
		// A path rooted in a variable reference (e.g., $var/text()) navigates
		// the variable's value, not the streaming source. Return 0.
		if _, isVar := derefXPathExpr(e.Filter).(xpath3.VariableExpr); isVar {
			return 0
		}
		filterGrounding := isGroundingExprSS(ss, e.Filter)
		filterCount := countStreamingDownwardSelectionsInner(ss, e.Filter, false)
		pathCount := 0
		if e.Path != nil {
			lp := *e.Path
			pathCount = countStreamingDownwardSelectionsInner(ss, lp, filterGrounding)
		}
		// When the filter and path both have downward selections,
		// they form a single combined path (like PathStepExpr).
		if filterCount > 0 && pathCount > 0 {
			count += pathCount
		} else {
			count += filterCount + pathCount
		}
	case xpath3.FunctionCall:
		// Check user-defined function streamability annotations.
		// Functions with filter/inspection/ascent annotations act as
		// motionless steps and do not add downward selections.
		// Absorbing functions ground their arguments.
		if e.Prefix != "" {
			cat := lookupFuncStreamability(ss, e.Name, len(e.Args))
			switch cat {
			case "filter", "shallow-descent":
				// Filter/shallow-descent: the function itself is a single
				// streaming step. Its argument provides the streaming input.
				// Count as 0 additional downward selections from the function.
				for _, arg := range e.Args {
					count += countStreamingDownwardSelectionsInner(ss, arg, false)
				}
				return count
			case "inspection", "ascent":
				// Inspection/ascent: motionless, no downward selection.
				return 0
			case "absorbing":
				// Absorbing: grounds arguments.
				for _, arg := range e.Args {
					count += countStreamingDownwardSelectionsInner(ss, arg, true)
				}
				return count
			case "unclassified":
				// Unclassified with copy-of: grounds arguments.
				for _, arg := range e.Args {
					count += countStreamingDownwardSelectionsInner(ss, arg, true)
				}
				return count
			}
		}
		// Built-in consuming functions (string, data, number, boolean,
		// normalize-space) that explicitly read the context item via "."
		// count as a consuming operation on the streaming input.
		// The zero-arg form (e.g., string() inside a path like @nr/string())
		// is NOT counted here because in that case the context is the
		// navigated-to node, not the streaming context.
		if e.Prefix == "" && isConsumingBuiltinFunc(e.Name) && len(e.Args) > 0 && isContextItemArg(e.Args[0]) {
			count++
			break
		}
		g := isGroundingExpr(e)
		for _, arg := range e.Args {
			count += countStreamingDownwardSelectionsInner(ss, arg, g)
		}
	case xpath3.BinaryExpr:
		count += countStreamingDownwardSelectionsInner(ss, e.Left, false)
		count += countStreamingDownwardSelectionsInner(ss, e.Right, false)
	case xpath3.ConcatExpr:
		count += countStreamingDownwardSelectionsInner(ss, e.Left, false)
		count += countStreamingDownwardSelectionsInner(ss, e.Right, false)
	case xpath3.SimpleMapExpr:
		// The ! operator processes items sequentially — the RHS operates
		// on each item from the LHS. Like a path expression, left and right
		// form a single combined downward selection, not two separate ones.
		leftGrounding := isGroundingExprSS(ss, e.Left)
		leftCount := countStreamingDownwardSelectionsInner(ss, e.Left, false)
		rightCount := countStreamingDownwardSelectionsInner(ss, e.Right, leftGrounding)
		if leftCount > 0 && rightCount > 0 {
			// Left and right form a single combined path.
			count += rightCount
		} else {
			count += leftCount + rightCount
		}
	case xpath3.UnionExpr:
		// A union of two downward selections is a single combined streaming
		// selection (the result is one merged node sequence), so count at
		// most 1 regardless of how many operands have downward paths.
		leftDown := countStreamingDownwardSelectionsInner(ss, e.Left, false)
		rightDown := countStreamingDownwardSelectionsInner(ss, e.Right, false)
		if leftDown > 0 || rightDown > 0 {
			count++
		}
	case xpath3.FilterExpr:
		count += countStreamingDownwardSelectionsInner(ss, e.Expr, false)
		for _, pred := range e.Predicates {
			count += countStreamingDownwardSelectionsInner(ss, pred, false)
		}
	case xpath3.SequenceExpr:
		for _, item := range e.Items {
			count += countStreamingDownwardSelectionsInner(ss, item, false)
		}
	case xpath3.IfExpr:
		count += countStreamingDownwardSelectionsInner(ss, e.Cond, false)
		thenCount := countStreamingDownwardSelectionsInner(ss, e.Then, false)
		elseCount := countStreamingDownwardSelectionsInner(ss, e.Else, false)
		if thenCount > elseCount {
			count += thenCount
		} else {
			count += elseCount
		}
	case xpath3.FLWORExpr:
		// Collect for-clause variables whose binding expressions consume the stream.
		consumingForVars := map[string]struct{}{}
		for _, clause := range e.Clauses {
			switch c := clause.(type) {
			case xpath3.ForClause:
				forCount := countStreamingDownwardSelectionsInner(ss, c.Expr, false)
				count += forCount
				// Only mark the variable as consuming if its binding expression
				// actually streams (not grounded by snapshot/copy-of).
				if forCount > 0 && !exprEndsWithGrounding(c.Expr) {
					consumingForVars[c.Var] = struct{}{}
				}
			case xpath3.LetClause:
				count += countStreamingDownwardSelectionsInner(ss, c.Expr, false)
			}
		}
		// If any for-clause consumes the stream and the return expression
		// navigates from the bound variable, that navigation also consumes
		// from the stream (the node must be buffered).
		if len(consumingForVars) > 0 && returnNavigatesFromVars(e.Return, consumingForVars) {
			count++
		}
		count += countStreamingDownwardSelectionsInner(ss, e.Return, false)
	case xpath3.InstanceOfExpr:
		count += countStreamingDownwardSelectionsInner(ss, e.Expr, false)
	case xpath3.CastExpr:
		count += countStreamingDownwardSelectionsInner(ss, e.Expr, false)
	case xpath3.CastableExpr:
		count += countStreamingDownwardSelectionsInner(ss, e.Expr, false)
	case xpath3.TreatAsExpr:
		count += countStreamingDownwardSelectionsInner(ss, e.Expr, false)
		// treat as document-node(element(X)) requires inspecting the
		// document element to verify the type, which is a consuming
		// operation on the streaming document.
		if treatAsIsConsuming(e) {
			count++
		}
	case xpath3.MapConstructorExpr:
		// Map entries are independent operands. Count crawling
		// (descendant/descendant-or-self) downward selections across all
		// entries. Striding selections (child axis) in individual entries
		// are fine — they each access a different child of the context item.
		crawlCount := 0
		for _, pair := range e.Pairs {
			keyCrawl := countCrawlingSelectionsInner(ss, pair.Key, false)
			valCrawl := countCrawlingSelectionsInner(ss, pair.Value, false)
			crawlCount += keyCrawl + valCrawl
			// Also check for a single entry where both key and value consume
			// the stream via crawling, even inside grounding functions like head().
			// E.g. map{head(//AUTHOR):data(head(//TITLE))}
			if keyCrawl == 0 && valCrawl == 0 {
				deepKey := countDeepCrawlingSelections(pair.Key)
				deepVal := countDeepCrawlingSelections(pair.Value)
				if deepKey > 0 && deepVal > 0 {
					crawlCount += deepKey + deepVal
				}
			}
		}
		count += crawlCount
	case xpath3.ArrayConstructorExpr:
		for _, item := range e.Items {
			count += countStreamingDownwardSelectionsInner(ss, item, false)
		}
	}
	return count
}

// isConsumingBuiltinFunc returns true for built-in functions that consume
// the string/typed value of their argument. When called with an explicit
// context item reference (.), these count as consuming operations in
// streaming analysis.
func isConsumingBuiltinFunc(name string) bool {
	switch name {
	case "string", "data", "number", "boolean", "normalize-space":
		return true
	}
	return false
}

// isContextItemArg returns true if the expression is a simple context item
// reference (`.`).
func isContextItemArg(expr xpath3.Expr) bool {
	_, ok := derefXPathExpr(expr).(xpath3.ContextItemExpr)
	return ok
}

// countCrawlingSelectionsInner counts only crawling (descendant/descendant-or-self)
// selections, ignoring striding (child axis) selections. Used for map constructors
// where each entry independently strides into child elements.
// Grounding functions are skipped — their arguments produce grounded data.
func countCrawlingSelectionsInner(ss *Stylesheet, expr xpath3.Expr, _ bool) int {
	expr = derefXPathExpr(expr)
	count := 0
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
		e = derefXPathExpr(e)
		if fc, ok := e.(xpath3.FunctionCall); ok && isGroundingExprSS(ss, fc) {
			return false // grounding function — result is grounded
		}
		if lp, ok := e.(xpath3.LocationPath); ok {
			for _, step := range lp.Steps {
				switch step.Axis {
				case xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
					count++
					return false
				}
			}
		}
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if ps.DescOrSelf {
				count++
				return false
			}
		}
		return true
	})
	return count
}

// countDeepCrawlingSelections counts crawling (descendant/descendant-or-self)
// selections in expr, including inside grounding functions. Used to detect
// multiple independent crawling paths in map constructor entries even when
// each is wrapped in a grounding function like head().
func countDeepCrawlingSelections(expr xpath3.Expr) int {
	expr = derefXPathExpr(expr)
	count := 0
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
		e = derefXPathExpr(e)
		if lp, ok := e.(xpath3.LocationPath); ok {
			for _, step := range lp.Steps {
				switch step.Axis {
				case xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
					count++
					return false
				}
			}
		}
		if ps, ok := e.(xpath3.PathStepExpr); ok {
			if ps.DescOrSelf {
				count++
				return false
			}
		}
		return true
	})
	return count
}

// treatAsIsConsuming returns true if a treat-as expression has a type test
// that requires consuming the streaming document. Specifically,
// document-node(element(X)) requires inspecting the document element.
func treatAsIsConsuming(e xpath3.TreatAsExpr) bool {
	if dt, ok := e.Type.ItemTest.(xpath3.DocumentTest); ok && dt.Inner != nil {
		return true
	}
	return false
}

// exprHasConsumingTreatAsInPath returns true if the expression contains a
// TreatAsExpr with a consuming type test (e.g., document-node(element(X)))
// used as the LHS/filter of a path expression that navigates downward.
// The type check consumes the document, and the path also consumes, giving
// two consuming operations which is not streamable.
func exprHasConsumingTreatAsInPath(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	found := false
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		e = derefXPathExpr(e)
		switch v := e.(type) {
		case xpath3.PathExpr:
			if v.Path != nil {
				if ta, ok := derefXPathExpr(v.Filter).(xpath3.TreatAsExpr); ok {
					if treatAsIsConsuming(ta) {
						found = true
						return false
					}
				}
			}
		case xpath3.PathStepExpr:
			if ta, ok := derefXPathExpr(v.Left).(xpath3.TreatAsExpr); ok {
				if treatAsIsConsuming(ta) {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
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
// a type that cannot hold streaming nodes: atomic types (xs:string, etc.),
// map types (map(*)), array types (array(*)), and function types (function(*)).
func isAtomicTypeConstraint(as string) bool {
	if as == "" {
		return false
	}
	// Strip occurrence indicators
	as = strings.TrimRight(as, "?*+")
	// Common atomic types
	if strings.HasPrefix(as, "xs:") {
		return true
	}
	// Map, array, and function types cannot hold streaming nodes
	if strings.HasPrefix(as, "map(") || strings.HasPrefix(as, "array(") || strings.HasPrefix(as, "function(") {
		return true
	}
	return false
}

// checkAscentFunction checks that an ascent function's body navigates
// upward from the streaming argument without consuming it.
func checkAscentFunction(fn *xslFunction) error {
	// First param must be a singleton node (node()?), not a sequence (node()*).
	if err := checkStreamingParamIsSingleton(fn, "ascent"); err != nil {
		return err
	}

	// Ascent functions may return the streaming parameter only if
	// the function body navigates upward from it (ancestor/parent).
	// If the body just returns the param without upward navigation,
	// it returns a raw streaming node, which is invalid.
	if functionReturnsStreamingParam(fn) && !functionUsesUpwardFromParam(fn) {
		return staticError(errCodeXTSE3430,
			"ascent function %q returns a streaming parameter without upward navigation, violating its declared streamability", fn.Name.Name)
	}

	for _, inst := range fn.Body {
		for _, expr := range getInstructionExprs(inst) {
			if exprConsumesParam(expr, fn.Params) {
				return staticError(errCodeXTSE3430,
					"ascent function %q consumes a streaming parameter, violating its declared streamability", fn.Name.Name)
			}
		}
	}

	// Check that additional params don't receive streaming nodes.
	if err := checkAdditionalParamsNotStreaming(fn, "ascent"); err != nil {
		return err
	}

	return nil
}

// checkStreamingParamIsSingleton checks that the first parameter of a streamable
// function accepts a singleton node (node()?), not a sequence (node()*).
// Functions declared with streamability categories like filter, inspection, ascent,
// and absorbing must receive a single streaming node, not a sequence.
func checkStreamingParamIsSingleton(fn *xslFunction, category string) error {
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

// checkAdditionalParamsNotStreaming checks that additional parameters (2nd, 3rd, ...)
// of a streamable function have atomic type constraints, so they cannot receive
// streaming nodes from the caller.
func checkAdditionalParamsNotStreaming(fn *xslFunction, category string) error {
	for i := 1; i < len(fn.Params); i++ {
		p := fn.Params[i]
		as := strings.TrimSpace(p.As)
		if isAtomicTypeConstraint(as) {
			continue
		}
		return staticError(errCodeXTSE3430,
			"%s function %q additional parameter %q could receive streaming nodes, violating its declared streamability",
			category, fn.Name.Name, p.Name)
	}
	return nil
}

// functionReturnsStreamingParam returns true if the function body can return
// the streaming parameter directly (not wrapped in a grounding function).
func functionReturnsStreamingParam(fn *xslFunction) bool {
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
		if seq, ok := inst.(*xslSequenceInst); ok && seq.Select != nil {
			if exprReturnsParam(seq.Select.AST(), paramNames) {
				return true
			}
		}
	}
	return false
}

// functionUsesUpwardFromParam returns true if the function navigates upward
// (parent/ancestor) from a streaming parameter.
func functionUsesUpwardFromParam(fn *xslFunction) bool {
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
				// $param/.. (PathExpr form)
				if pe, ok := e.(xpath3.PathExpr); ok && pe.Path != nil {
					var varName string
					// Filter may be a bare VariableExpr or wrapped in FilterExpr.
					if ve, ok := pe.Filter.(xpath3.VariableExpr); ok {
						varName = ve.Name
					} else if fe, ok := pe.Filter.(xpath3.FilterExpr); ok {
						if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
							varName = ve.Name
						}
					}
					if varName != "" && paramNames[varName] {
						lp := *pe.Path
						if exprHasUpwardAxis(lp) {
							found = true
							return false
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
func functionBodyIsGrounded(fn *xslFunction) bool {
	if len(fn.Params) == 0 {
		return true
	}

	paramNames := make(map[string]bool)
	for _, p := range fn.Params {
		paramNames[p.Name] = true
	}

	for _, inst := range fn.Body {
		if seq, ok := inst.(*xslSequenceInst); ok && seq.Select != nil {
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
func functionHasMultipleConsumingRefs(fn *xslFunction) bool {
	if len(fn.Params) == 0 {
		return false
	}

	for _, p := range fn.Params {
		refs := 0
		collectAllInstructionExprs(fn.Body, func(expr *xpath3.Expression) {
			refs += countParamDownwardRefs(expr, p.Name)
		})
		if refs > 1 {
			return true
		}
	}

	return false
}
