package xslt3

import "github.com/lestrrat-go/helium/xpath3"

// collectAllInstructionExprs recursively walks an instruction tree and calls
// fn for every XPath expression found, including expressions in nested bodies.
func collectAllInstructionExprs(insts []instruction, fn func(*xpath3.Expression)) {
	for _, inst := range insts {
		for _, expr := range getInstructionExprs(inst) {
			fn(expr)
		}
		// Recurse into instruction bodies
		for _, body := range getInstructionBodies(inst) {
			collectAllInstructionExprs(body, fn)
		}
	}
}

// getInstructionBodies returns all child instruction bodies from an instruction.
func getInstructionBodies(inst instruction) [][]instruction {
	switch v := inst.(type) {
	case *copyInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *ifInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *chooseInst:
		var bodies [][]instruction
		for _, w := range v.When {
			if w.Body != nil {
				bodies = append(bodies, w.Body)
			}
		}
		if v.Otherwise != nil {
			bodies = append(bodies, v.Otherwise)
		}
		return bodies
	case *forEachInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *elementInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *literalResultElement:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *variableInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *sequenceInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *valueOfInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *attributeInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *messageInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *resultDocumentInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *mapInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *mapEntryInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *documentInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *iterateInst:
		var bodies [][]instruction
		if v.Body != nil {
			bodies = append(bodies, v.Body)
		}
		if v.OnCompletion != nil {
			bodies = append(bodies, v.OnCompletion)
		}
		return bodies
	case *forkInst:
		return v.Branches
	case *forEachGroupInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *tryCatchInst:
		var bodies [][]instruction
		if v.Try != nil {
			bodies = append(bodies, v.Try)
		}
		for _, c := range v.Catches {
			if c.Body != nil {
				bodies = append(bodies, c.Body)
			}
		}
		return bodies
	case *onEmptyInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *onNonEmptyInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *breakInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *performSortInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	case *analyzeStringInst:
		var bodies [][]instruction
		if v.MatchingBody != nil {
			bodies = append(bodies, v.MatchingBody)
		}
		if v.NonMatchingBody != nil {
			bodies = append(bodies, v.NonMatchingBody)
		}
		return bodies
	case *wherePopulatedInst:
		if v.Body != nil {
			return [][]instruction{v.Body}
		}
	}
	return nil
}

// functionHasConsumingRefInLoop checks if a parameter is used with a positional
// subscript in a loop construct.
func functionHasConsumingRefInLoop(fn *xslFunction) bool {
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

// functionHasParamInSimpleMap returns true if the streaming parameter (first param)
// is referenced on the right-hand side of a simple mapping expression (! operator),
// which causes the parameter to be consumed multiple times.
// E.g., (1 to 5) ! $n — $n is accessed for each item in the range.
func functionHasParamInSimpleMap(fn *xslFunction) bool {
	if len(fn.Params) == 0 {
		return false
	}
	// Only the first param is the streaming param for shallow-descent.
	streamingParamName := fn.Params[0].Name

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
				if sme, ok := e.(xpath3.SimpleMapExpr); ok {
					// Check if the RHS references the streaming parameter
					xpath3.WalkExpr(sme.Right, func(inner xpath3.Expr) bool {
						if found {
							return false
						}
						if ve, ok := inner.(xpath3.VariableExpr); ok {
							if ve.Name == streamingParamName {
								found = true
								return false
							}
						}
						return true
					})
				}
				return !found
			})
			if found {
				return true
			}
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
		if pe, ok := e.(xpath3.PathExpr); ok && pe.Path != nil {
			var matched bool
			if ve, ok := pe.Filter.(xpath3.VariableExpr); ok {
				matched = ve.Name == paramName
			} else if fe, ok := pe.Filter.(xpath3.FilterExpr); ok {
				if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
					matched = ve.Name == paramName
				}
			}
			if matched {
				count++
				return false
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
					"serialize", "deep-equal", "sort", "reverse",
					"empty", "exists", "count", "sum", "avg", "min", "max",
					"string", "data", "boolean":
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
func exprConsumesParam(expr *xpath3.Expression, params []*param) bool {
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
		if pe, ok := e.(xpath3.PathExpr); ok && pe.Path != nil {
			// Filter may be a bare VariableExpr or wrapped in FilterExpr.
			if ve, ok := pe.Filter.(xpath3.VariableExpr); ok {
				if ve.Name == varName {
					found = true
					return false
				}
			}
			if fe, ok := pe.Filter.(xpath3.FilterExpr); ok {
				if ve, ok := fe.Expr.(xpath3.VariableExpr); ok {
					if ve.Name == varName {
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
func getInstructionExprs(inst instruction) []*xpath3.Expression {
	var exprs []*xpath3.Expression

	switch v := inst.(type) {
	case *applyTemplatesInst:
		exprs = append(exprs, v.Select)
		for _, sk := range v.Sort {
			exprs = append(exprs, sk.Select)
		}
		for _, wp := range v.Params {
			exprs = append(exprs, wp.Select)
		}
	case *valueOfInst:
		exprs = append(exprs, v.Select)
	case *copyOfInst:
		exprs = append(exprs, v.Select)
	case *copyInst:
		exprs = append(exprs, v.Select)
	case *forEachInst:
		exprs = append(exprs, v.Select)
		for _, sk := range v.Sort {
			exprs = append(exprs, sk.Select)
		}
	case *ifInst:
		exprs = append(exprs, v.Test)
	case *chooseInst:
		for _, when := range v.When {
			exprs = append(exprs, when.Test)
		}
	case *variableInst:
		exprs = append(exprs, v.Select)
	case *xslSequenceInst:
		exprs = append(exprs, v.Select)
	case *mapEntryInst:
		exprs = append(exprs, v.Key)
		exprs = append(exprs, v.Select)
	case *attributeInst:
		exprs = append(exprs, v.Select)
	case *commentInst:
		exprs = append(exprs, v.Select)
	case *piInst:
		exprs = append(exprs, v.Select)
	case *numberInst:
		exprs = append(exprs, v.Value)
		exprs = append(exprs, v.Select)
	case *messageInst:
		exprs = append(exprs, v.Select)
	case *namespaceInst:
		exprs = append(exprs, v.Select)
	case *performSortInst:
		exprs = append(exprs, v.Select)
		for _, sk := range v.Sort {
			exprs = append(exprs, sk.Select)
		}
	case *iterateInst:
		exprs = append(exprs, v.Select)
		for _, p := range v.Params {
			exprs = append(exprs, p.Select)
		}
	case *breakInst:
		exprs = append(exprs, v.Select)
	case *forEachGroupInst:
		exprs = append(exprs, v.Select)
		exprs = append(exprs, v.GroupBy)
		exprs = append(exprs, v.GroupAdjacent)
		for _, sk := range v.Sort {
			exprs = append(exprs, sk.Select)
		}
	case *tryCatchInst:
		exprs = append(exprs, v.Select)
		for _, c := range v.Catches {
			exprs = append(exprs, c.Select)
		}
	case *onEmptyInst:
		exprs = append(exprs, v.Select)
	case *onNonEmptyInst:
		exprs = append(exprs, v.Select)
	case *callTemplateInst:
		for _, wp := range v.Params {
			exprs = append(exprs, wp.Select)
		}
	case *nextMatchInst:
		for _, wp := range v.Params {
			exprs = append(exprs, wp.Select)
		}
	case *applyImportsInst:
		for _, wp := range v.Params {
			exprs = append(exprs, wp.Select)
		}
	case *analyzeStringInst:
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
func getChildInstructions(inst instruction) [][]instruction {
	var children [][]instruction

	switch v := inst.(type) {
	case *ifInst:
		children = append(children, v.Body)
	case *chooseInst:
		for _, when := range v.When {
			children = append(children, when.Body)
		}
		children = append(children, v.Otherwise)
	case *forEachInst:
		children = append(children, v.Body)
	case *variableInst:
		children = append(children, v.Body)
	case *elementInst:
		children = append(children, v.Body)
	case *attributeInst:
		children = append(children, v.Body)
	case *commentInst:
		children = append(children, v.Body)
	case *piInst:
		children = append(children, v.Body)
	case *messageInst:
		children = append(children, v.Body)
	case *sequenceInst:
		children = append(children, v.Body)
	case *resultDocumentInst:
		children = append(children, v.Body)
	case *mapInst:
		children = append(children, v.Body)
	case *mapEntryInst:
		children = append(children, v.Body)
	case *namespaceInst:
		children = append(children, v.Body)
	case *performSortInst:
		children = append(children, v.Body)
	case *literalResultElement:
		children = append(children, v.Body)
	case *copyInst:
		children = append(children, v.Body)
	case *valueOfInst:
		children = append(children, v.Body)
	case *sourceDocumentInst:
		// Don't recurse into source-document bodies here; they're handled
		// separately by checkInstructionsForStreamableSourceDoc.
	case *iterateInst:
		children = append(children, v.Body)
		children = append(children, v.OnCompletion)
	case *breakInst:
		children = append(children, v.Body)
	case *forkInst:
		children = append(children, v.Branches...)
	case *forEachGroupInst:
		children = append(children, v.Body)
	case *tryCatchInst:
		children = append(children, v.Try)
		for _, c := range v.Catches {
			children = append(children, c.Body)
		}
	case *onEmptyInst:
		children = append(children, v.Body)
	case *onNonEmptyInst:
		children = append(children, v.Body)
	case *wherePopulatedInst:
		children = append(children, v.Body)
	case *callTemplateInst:
		for _, wp := range v.Params {
			children = append(children, wp.Body)
		}
	case *applyTemplatesInst:
		for _, wp := range v.Params {
			children = append(children, wp.Body)
		}
	case *nextMatchInst:
		for _, wp := range v.Params {
			children = append(children, wp.Body)
		}
	case *applyImportsInst:
		for _, wp := range v.Params {
			children = append(children, wp.Body)
		}
	case *analyzeStringInst:
		children = append(children, v.MatchingBody)
		children = append(children, v.NonMatchingBody)
	}

	return children
}

// patternHasCurrentOnElementStep checks if a pattern expression has a
// current() call in a predicate of a step that matches element nodes.
// In streaming mode, current() on an element step accesses the element's
// string value, which requires reading children — non-motionless.
// For text nodes, current() is motionless since the value is immediate.
func patternHasCurrentOnElementStep(expr xpath3.Expr) bool {
	found := false
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if lp, ok := e.(xpath3.LocationPath); ok {
			for _, step := range lp.Steps {
				if !stepMatchesElements(step) {
					continue
				}
				for _, pred := range step.Predicates {
					if exprUsesCurrent(pred) {
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

// stepMatchesElements returns true if the step's node test could match
// element nodes (nameTest or generic node() test, but not text()/comment()/etc.).
func stepMatchesElements(step xpath3.Step) bool {
	switch step.NodeTest.(type) {
	case xpath3.NameTest:
		return true // name tests match elements by default
	case xpath3.TypeTest:
		tt := step.NodeTest.(xpath3.TypeTest)
		return tt.Kind == xpath3.NodeKindNode
	}
	return false
}

// exprUsesCurrent returns true if the expression contains a call to current().
func exprUsesCurrent(expr xpath3.Expr) bool {
	found := false
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if fc, ok := e.(xpath3.FunctionCall); ok {
			if fc.Prefix == "" && fc.Name == "current" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}
