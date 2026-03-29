package xslt3

import (
	"slices"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
)

// checkMultipleDownwardInst checks if an instruction has multiple consuming
// operations that would require reading the stream multiple times.
func checkMultipleDownwardInst(ss *Stylesheet, inst instruction) error {
	switch v := inst.(type) {
	case *forkInst:
		// xsl:fork is specifically designed to allow multiple consuming branches.
		// Each branch processes the input stream independently, so multiple
		// downward selections across branches are permitted.
		// However, within a single branch, multiple downward selections are still an error.
		for _, branch := range v.Branches {
			branchDown := countDownwardInInstructions(ss, branch)
			if branchDown > 1 {
				return staticError(errCodeXTSE3430,
					"xsl:fork branch has multiple consuming operations, which is not streamable")
			}
		}

		// When a fork has multiple consuming branches and any branch
		// returns raw streaming nodes via xsl:sequence select=<downward>,
		// the fork would need to return streaming nodes from multiple
		// stream copies, which is not streamable.
		consumingBranches := 0
		hasStreamingReturn := false
		for _, branch := range v.Branches {
			branchDown := countDownwardInInstructions(ss, branch)
			if branchDown > 0 {
				consumingBranches++
			}
			for _, bi := range branch {
				if seq, ok := bi.(*xslSequenceInst); ok && seq.Select != nil {
					ast := seq.Select.AST()
					// A branch returns streaming nodes if it selects
					// downward without grounding or atomizing the result.
					if xpathstream.ExprHasDownwardStep(seq.Select) &&
						!exprEndsWithGrounding(ast) &&
						!exprProducesAtomicResult(ast) {
						hasStreamingReturn = true
					}
				}
			}
		}
		if consumingBranches > 1 && hasStreamingReturn {
			return staticError(errCodeXTSE3430,
				"xsl:fork with multiple consuming branches returns streamed nodes, which is not streamable")
		}

	case *iterateInst:
		// Within xsl:iterate body, check for multiple downward selections.
		bodyDown := countDownwardInInstructions(ss, v.Body)
		if bodyDown > 1 {
			return staticError(errCodeXTSE3430,
				"xsl:iterate body has multiple downward selections, which is not streamable")
		}

		// Check if iterate body returns streamed nodes via xsl:sequence select="."
		for _, bi := range v.Body {
			if seq, ok := bi.(*xslSequenceInst); ok && seq.Select != nil {
				ast := seq.Select.AST()
				if _, ok := ast.(xpath3.ContextItemExpr); ok {
					return staticError(errCodeXTSE3430,
						"xsl:iterate body returns streamed nodes via xsl:sequence select=\".\", which is not streamable")
				}
			}
		}

	case *forEachGroupInst:
		// When the select is grounded (copy-of/snapshot), the group items are
		// in-memory copies.  current-group() and context item both refer to
		// grounded data, so multiple-consumption checks don't apply.
		fegSelectGrounded := v.Select != nil && exprEndsWithGrounding(v.Select.AST())

		if !fegSelectGrounded {
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
			// Only flag when current-group() is used consumingly AND the context
			// item is used with downward navigation from a DIFFERENT expression.
			// Downward steps reached via current-group()/... are navigation into
			// group items, not from the context item.
			bodyUsesContextDown := false
			bodyUsesCurrentGroupConsuming := false
			for _, bi := range v.Body {
				for _, expr := range getInstructionExprs(bi) {
					if expr == nil {
						continue
					}
					if exprHasContextOnlyDownward(expr.AST()) {
						bodyUsesContextDown = true
					}
					if exprUsesCurrentGroupConsumingly(expr) {
						bodyUsesCurrentGroupConsuming = true
					}
				}
				// Check LRE attribute AVTs.
				if lre, ok := bi.(*literalResultElement); ok {
					for _, attr := range lre.Attrs {
						if attr.Value != nil {
							for _, part := range attr.Value.parts {
								if part.expr != nil {
									if exprHasContextOnlyDownward(part.expr.AST()) {
										bodyUsesContextDown = true
									}
									if exprUsesCurrentGroupConsumingly(part.expr) {
										bodyUsesCurrentGroupConsuming = true
									}
								}
							}
						}
					}
				}
				for _, children := range getChildInstructions(bi) {
					for _, ci := range children {
						for _, expr := range getInstructionExprs(ci) {
							if expr == nil {
								continue
							}
							if exprHasContextOnlyDownward(expr.AST()) {
								bodyUsesContextDown = true
							}
							if exprUsesCurrentGroupConsumingly(expr) {
								bodyUsesCurrentGroupConsuming = true
							}
						}
					}
				}
			}
			if bodyUsesContextDown && bodyUsesCurrentGroupConsuming {
				return staticError(errCodeXTSE3430,
					"xsl:for-each-group body uses both context item (downward) and current-group(), which is not streamable")
			}

			// Check for focus-changing instructions (xsl:copy select=...,
			// xsl:for-each) whose body uses current-group().  Per bug 29482,
			// current-group() inside a focus-changing instruction is a
			// higher-order consumption that is not streamable.
			if err := checkFocusChangingCurrentGroup(v.Body); err != nil {
				return err
			}

			// Check if apply-templates selects current-group() and the target
			// streaming mode templates use current-group().  current-group() is
			// not available in applied templates, so this is a static error.
			if err := checkApplyTemplatesCurrentGroup(ss, v.Body); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkApplyTemplatesCurrentGroup checks if any apply-templates instruction
// in a for-each-group body selects current-group() and applies to a streaming
// mode whose templates use current-group() outside of their own for-each-group.
// current-group() is not available in applied templates, so using it there is
// a static streamability error.
func checkApplyTemplatesCurrentGroup(ss *Stylesheet, body []instruction) error {
	for _, inst := range body {
		if at, ok := inst.(*applyTemplatesInst); ok {
			if at.Select != nil && exprUsesCurrentGroup(at.Select) {
				modeName := at.Mode
				if modeName == "" || modeName == "#current" {
					modeName = "#default"
				}
				md := ss.modeDefs[modeName]
				if md != nil && md.Streamable {
					// Check if any template in this mode uses current-group()
					// at the top level (not inside a nested for-each-group).
					for _, tmpl := range ss.templates {
						tmplMode, ok := streamabilityModeNameForTemplate(tmpl)
						if !ok || tmplMode != modeName {
							continue
						}
						if bodyUsesCurrentGroupOutsideFEG(tmpl.Body) {
							return staticError(errCodeXTSE3430,
								"xsl:apply-templates select=\"current-group()\" targets streaming mode %q whose templates use current-group(), which is not streamable",
								modeName)
						}
					}
				}
			}
		}
		// Recurse into child instructions (e.g., LREs wrapping apply-templates).
		for _, children := range getChildInstructions(inst) {
			if err := checkApplyTemplatesCurrentGroup(ss, children); err != nil {
				return err
			}
		}
	}
	return nil
}

// bodyUsesCurrentGroupOutsideFEG checks if any instruction in the body uses
// current-group() outside of a for-each-group instruction.  Usage inside a
// nested for-each-group refers to that inner group, not the outer one.
func bodyUsesCurrentGroupOutsideFEG(body []instruction) bool {
	for _, bi := range body {
		// Skip for-each-group — its body establishes a new group context.
		if _, ok := bi.(*forEachGroupInst); ok {
			continue
		}
		for _, expr := range getInstructionExprs(bi) {
			if expr == nil {
				continue
			}
			if xpathstream.ExprUsesFunction(expr, lexicon.FnCurrentGroup) {
				return true
			}
		}
		// Check LRE attribute AVTs (not covered by getInstructionExprs).
		if lre, ok := bi.(*literalResultElement); ok {
			for _, attr := range lre.Attrs {
				if attr.Value.hasFunction(lexicon.FnCurrentGroup) {
					return true
				}
			}
		}
		if slices.ContainsFunc(getChildInstructions(bi), bodyUsesCurrentGroupOutsideFEG) {
			return true
		}
	}
	return false
}

// checkUseAttributeSetsStreamable checks that attribute sets used in a streaming
// context are themselves streamable. An attribute set is non-streamable if any
// of its attribute instructions contain non-streamable expressions (downward
// navigation, last(), etc.), or if the combined downward selections from the
// instruction and attribute set exceed 1.
func checkUseAttributeSetsStreamable(ss *Stylesheet, inst instruction) error {
	var attrSetNames []string
	switch v := inst.(type) {
	case *copyInst:
		attrSetNames = v.UseAttrSets
	case *elementInst:
		attrSetNames = v.UseAttrSets
	case *literalResultElement:
		attrSetNames = v.UseAttrSets
	}
	if len(attrSetNames) == 0 {
		return nil
	}

	// Count downward selections in the instruction's own expressions
	instDown := 0
	for _, expr := range getInstructionExprs(inst) {
		instDown += countStreamingDownwardSelections(ss, expr.AST())
	}

	for _, name := range attrSetNames {
		asDef := ss.attributeSets[name]
		if asDef == nil {
			continue
		}

		// Check individual expressions for streamability violations
		for _, attrInst := range asDef.Attrs {
			for _, expr := range getInstructionExprs(attrInst) {
				if err := checkStreamableExpr(ss, expr); err != nil {
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
					if err := checkStreamableExpr(ss, expr); err != nil {
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
func countAttributeSetDownward(ss *Stylesheet, asDef *attributeSetDef) int {
	total := 0
	for _, attrInst := range asDef.Attrs {
		for _, expr := range getInstructionExprs(attrInst) {
			total += countStreamingDownwardSelections(ss, expr.AST())
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
//
// When inResultDoc is true, the xsl:sequence select="." check is skipped
// because nodes flow to a serializer (xsl:result-document) rather than
// being returned as a sequence that requires materialization.
func checkForEachStreamable(_ *Stylesheet, inst instruction, inResultDoc bool) error {
	switch v := inst.(type) {
	case *forEachInst:
		// Check if select expression is crawling AND body consumes context.
		// But if the select grounds its result (e.g., snapshot(//...)), the body
		// operates on grounded data and consuming is fine.
		if v.Select != nil && xpathstream.ExprUsesDescendantOrSelf(v.Select) &&
			!exprEndsWithGrounding(v.Select.AST()) {
			if forEachBodyConsumesContext(v.Body) {
				return staticError(errCodeXTSE3430,
					"xsl:for-each with crawling select expression %q and consuming body is not streamable", v.Select.String())
			}
		}
		// Check for xsl:sequence select="." returning streamed nodes.
		// Skip this check when inside xsl:result-document — nodes are
		// written to a secondary output (serializer), not returned.
		if !inResultDoc {
			for _, bi := range v.Body {
				if seq, ok := bi.(*xslSequenceInst); ok && seq.Select != nil {
					ast := seq.Select.AST()
					if _, ok := ast.(xpath3.ContextItemExpr); ok {
						return staticError(errCodeXTSE3430,
							"xsl:for-each body returns streamed nodes via xsl:sequence select=\".\", which is not streamable")
					}
				}
			}
		}
		// Check for variable bound to streaming context used in loop body
		if err := checkStreamingVarInLoop(v.Body); err != nil {
			return err
		}
	case *iterateInst:
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
//
// Grounding instructions (xsl:copy select=".", xsl:copy-of select=".",
// xsl:copy without select) are excluded because they produce deep copies
// of the current node and are allowed inside a crawling for-each.
func forEachBodyConsumesContext(body []instruction) bool {
	for _, inst := range body {
		// xsl:copy with select="." or no select is a grounding operation.
		if ci, ok := inst.(*copyInst); ok {
			if ci.Select == nil {
				continue // xsl:copy (shallow copy + body) — grounding
			}
			if _, isCtx := ci.Select.AST().(xpath3.ContextItemExpr); isCtx {
				continue // xsl:copy select="." — grounding
			}
		}
		// xsl:copy-of select="." is a grounding operation.
		if coi, ok := inst.(*copyOfInst); ok && coi.Select != nil {
			if _, isCtx := coi.Select.AST().(xpath3.ContextItemExpr); isCtx {
				continue // copy-of select="." — grounding
			}
		}
		for _, expr := range getInstructionExprs(inst) {
			if expr == nil {
				continue
			}
			// Check if the expression accesses "." (context item) — consuming it.
			// Also check for implicit context usage via downward steps (child/
			// descendant axis) which navigate into children of the streaming
			// context item.
			if exprReferencesContextItem(expr.AST()) || xpathstream.ExprHasDownwardStep(expr) {
				return true
			}
		}
		if slices.ContainsFunc(getChildInstructions(inst), forEachBodyConsumesContext) {
			return true
		}
	}
	return false
}

// checkStreamingVarInLoop checks if a variable is bound to the streaming context
// node (select=".") and then used consumingly (downward navigation) in a loop body.
// This is non-streamable because the loop iterates over a non-streaming range but
// accesses the streamed variable repeatedly.
func checkStreamingVarInLoop(body []instruction) error {
	// Collect variables bound to streaming context (select=".")
	streamingVars := make(map[string]bool)
	for _, inst := range body {
		if vi, ok := inst.(*variableInst); ok && vi.Select != nil {
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
		case *forEachInst:
			if err := checkVarConsumingInBody(streamingVars, v.Body); err != nil {
				return err
			}
		case *iterateInst:
			if err := checkVarConsumingInBody(streamingVars, v.Body); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkVarConsumingInBody checks if any streaming variable is used consumingly
// in a set of instructions.
func checkVarConsumingInBody(streamingVars map[string]bool, body []instruction) error {
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
func checkForEachGroupStreamable(_ *Stylesheet, inst instruction) error {
	fg, ok := inst.(*forEachGroupInst)
	if !ok {
		return nil
	}

	// When select ends with a grounding function (copy-of, snapshot), the
	// selected items are in-memory copies.  Grouping keys can navigate freely
	// into them, current-group() is grounded, and the body does not consume the
	// stream.  Most of the checks below only apply to non-grounded selects.
	selectGrounded := fg.Select != nil && exprEndsWithGrounding(fg.Select.AST())

	// Check if sort is used with for-each-group in streaming (non-streamable).
	// When the select is grounded (copy-of/snapshot), items are in memory and
	// sorting is fine.
	if len(fg.Sort) > 0 && !selectGrounded {
		return staticError(errCodeXTSE3430,
			"xsl:for-each-group with sort is not streamable")
	}

	// Check if group-starting-with / group-ending-with pattern has
	// non-motionless predicates (e.g., record[foo = 'a'] where foo is
	// a child element).  The pattern must be motionless for streaming
	// because the processor tests the pattern before consuming the element.
	// Skip when select is grounded (items are in memory) or when select
	// uses current-group() (items are already materialized from an outer
	// for-each-group).
	selectMaterialized := selectGrounded || (fg.Select != nil && exprUsesCurrentGroup(fg.Select))
	if !selectMaterialized {
		if fg.GroupStartingWith != nil && patternHasNonMotionlessPredicate(fg.GroupStartingWith) {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group group-starting-with pattern has a non-motionless predicate, which is not streamable")
		}
		if fg.GroupEndingWith != nil && patternHasNonMotionlessPredicate(fg.GroupEndingWith) {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group group-ending-with pattern has a non-motionless predicate, which is not streamable")
		}
	}

	// Check if grouping key navigates downward (e.g., PRICE/text()) — only
	// when the select is NOT grounded, because grounded items can be navigated.
	if !selectGrounded {
		if fg.GroupBy != nil && xpathstream.ExprHasDownwardStep(fg.GroupBy) {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group group-by expression %q navigates downward, which is not streamable",
				fg.GroupBy.String())
		}
		if fg.GroupAdjacent != nil && xpathstream.ExprHasDownwardStep(fg.GroupAdjacent) {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group group-adjacent expression %q navigates downward, which is not streamable",
				fg.GroupAdjacent.String())
		}
	}

	// Crawling selects (descendant-or-self axis) with consuming use of
	// current-group() in the body are not streamable — the processor would
	// need to buffer unbounded amounts of data.  When the select is grounded,
	// items are already in memory and crawling is fine.
	if !selectGrounded && fg.Select != nil && xpathstream.ExprUsesDescendantOrSelf(fg.Select) {
		groupRefs := countCurrentGroupConsumingRefs(fg.Body)
		if groupRefs > 0 {
			return staticError(errCodeXTSE3430,
				"xsl:for-each-group with crawling select and consuming use of current-group() is not streamable")
		}
	}

	// Check for nested for-each-group that consumes current-group(),
	// but only when the outer for-each-group select doesn't ground its data.
	if !selectGrounded {
		for _, bi := range fg.Body {
			if innerFg, ok := bi.(*forEachGroupInst); ok {
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

	// Check if for-each-group body has source-document with streaming that uses
	// current-group() consumingly.  When the outer select is grounded,
	// current-group() returns in-memory nodes that can be navigated freely, so
	// only flag uses that truly consume (copy-of, apply-templates with
	// downward select, etc.) — not simple attribute/property access.
	// Recurse into nested instructions (LREs, etc.) to find source-documents.
	if !selectGrounded {
		if err := checkSourceDocCurrentGroup(fg.Body); err != nil {
			return err
		}
	}

	// group-starting-with naturally buffers the entire current group so that
	// it can detect where a new group starts.  Because the group is already
	// buffered, current-group() in the body is always available and does not
	// represent an extra stream consumption — no additional check is needed.

	return nil
}

// checkSourceDocCurrentGroup recursively checks if any source-document
// instruction with streaming uses current-group() in its body.
func checkSourceDocCurrentGroup(body []instruction) error {
	for _, bi := range body {
		if sd, ok := bi.(*sourceDocumentInst); ok && sd.Streamable {
			if sourceDocBodyUsesCurrentGroup(sd.Body) {
				return staticError(errCodeXTSE3430,
					"xsl:source-document streamable body uses current-group() from outer for-each-group, which is not streamable")
			}
		}
		// Recurse into child instructions to find nested source-documents.
		for _, children := range getChildInstructions(bi) {
			if err := checkSourceDocCurrentGroup(children); err != nil {
				return err
			}
		}
	}
	return nil
}

// sourceDocBodyUsesCurrentGroup checks if any expression in the source-document
// body uses current-group().
func sourceDocBodyUsesCurrentGroup(body []instruction) bool {
	for _, inst := range body {
		if slices.ContainsFunc(getInstructionExprs(inst), exprUsesCurrentGroup) {
			return true
		}
		if slices.ContainsFunc(getChildInstructions(inst), sourceDocBodyUsesCurrentGroup) {
			return true
		}
	}
	return false
}

// patternHasNonMotionlessPredicate returns true if any alternative in the
// pattern has a predicate that navigates downward (child/descendant steps).
func patternHasNonMotionlessPredicate(pat *pattern) bool {
	for _, alt := range pat.Alternatives {
		if alt.expr == nil {
			continue
		}
		nonMotionless := false
		xpathstream.WalkExpr(alt.expr, func(e xpath3.Expr) bool {
			if nonMotionless {
				return false
			}
			switch v := e.(type) {
			case xpath3.LocationPath:
				for _, step := range v.Steps {
					for _, pred := range step.Predicates {
						if xpathstream.PredicateIsNonMotionlessWithStep(pred, &step) {
							nonMotionless = true
							return false
						}
					}
				}
			case xpath3.FilterExpr:
				if slices.ContainsFunc(v.Predicates, xpathstream.PredicateIsNonMotionless) {
					nonMotionless = true
					return false
				}
			}
			return true
		})
		if nonMotionless {
			return true
		}
	}
	return false
}

// checkFocusChangingCurrentGroup checks if a focus-changing instruction
// (xsl:copy with select, xsl:for-each) in the for-each-group body uses
// current-group() in its own body.  Per W3C bug 29482, this is a
// higher-order consumption that is not streamable.
func checkFocusChangingCurrentGroup(body []instruction) error {
	for _, bi := range body {
		if err := checkFocusChangingCurrentGroupInst(bi); err != nil {
			return err
		}
	}
	return nil
}

func checkFocusChangingCurrentGroupInst(inst instruction) error {
	switch v := inst.(type) {
	case *copyInst:
		// xsl:copy with an explicit select changes focus. If its body
		// uses current-group(), that's a higher-order consumption.
		// xsl:copy without select (nil) or with select="." copies the
		// context item, which is the normal streaming case.
		if v.Select != nil && !exprIsContextItem(v.Select) {
			if bodyUsesCurrentGroup(v.Body) {
				return staticError(errCodeXTSE3430,
					"xsl:for-each-group body has current-group() inside focus-changing xsl:copy, which is not streamable")
			}
		}
	}
	// Recurse into child instructions (e.g., result-document wrapping copy).
	for _, children := range getChildInstructions(inst) {
		for _, child := range children {
			if err := checkFocusChangingCurrentGroupInst(child); err != nil {
				return err
			}
		}
	}
	return nil
}

// exprIsContextItem returns true if the expression is just "." (context item).
func exprIsContextItem(expr *xpath3.Expression) bool {
	if expr == nil {
		return false
	}
	ast := expr.AST()
	if ast == nil {
		return false
	}
	_, ok := derefXPathExpr(ast).(xpath3.ContextItemExpr)
	return ok
}

// bodyUsesCurrentGroup returns true if any instruction in the body
// uses current-group() in any expression.
func bodyUsesCurrentGroup(body []instruction) bool {
	for _, bi := range body {
		for _, expr := range getInstructionExprs(bi) {
			if expr == nil {
				continue
			}
			if xpathstream.ExprUsesFunction(expr, lexicon.FnCurrentGroup) {
				return true
			}
		}
		if slices.ContainsFunc(getChildInstructions(bi), bodyUsesCurrentGroup) {
			return true
		}
	}
	return false
}

// exprUsesCurrentGroupConsumingly checks if an expression uses current-group()
// in a consuming way.  The only non-consuming use is when current-group() is
// wrapped in snapshot() which produces grounded copies.  All other uses —
// bare current-group(), copy-of(current-group()), current-group()/path — are
// consuming because they iterate over the streamed group items.
func exprUsesCurrentGroupConsumingly(expr *xpath3.Expression) bool {
	found := false
	var walk func(e xpath3.Expr, insideSnapshot bool)
	walk = func(e xpath3.Expr, insideSnapshot bool) {
		if found || e == nil {
			return
		}
		e = derefXPathExpr(e)
		switch v := e.(type) {
		case xpath3.FunctionCall:
			if v.Prefix == "" && v.Name == lexicon.FnCurrentGroup {
				if !insideSnapshot {
					found = true
				}
				return
			}
			// snapshot() grounds its arguments — current-group() inside is not consuming.
			if v.Prefix == "" && v.Name == "snapshot" {
				for _, arg := range v.Args {
					walk(arg, true)
				}
				return
			}
			for _, arg := range v.Args {
				walk(arg, insideSnapshot)
			}
		case xpath3.PathStepExpr:
			walk(v.Left, insideSnapshot)
			walk(v.Right, insideSnapshot)
		case xpath3.PathExpr:
			walk(v.Filter, insideSnapshot)
			if v.Path != nil {
				lp := *v.Path
				walk(lp, insideSnapshot)
			}
		case xpath3.FilterExpr:
			walk(v.Expr, insideSnapshot)
			for _, p := range v.Predicates {
				walk(p, insideSnapshot)
			}
		case xpath3.BinaryExpr:
			walk(v.Left, insideSnapshot)
			walk(v.Right, insideSnapshot)
		case xpath3.UnaryExpr:
			walk(v.Operand, insideSnapshot)
		case xpath3.ConcatExpr:
			walk(v.Left, insideSnapshot)
			walk(v.Right, insideSnapshot)
		case xpath3.UnionExpr:
			walk(v.Left, insideSnapshot)
			walk(v.Right, insideSnapshot)
		case xpath3.IfExpr:
			walk(v.Cond, insideSnapshot)
			walk(v.Then, insideSnapshot)
			walk(v.Else, insideSnapshot)
		case xpath3.SequenceExpr:
			for _, item := range v.Items {
				walk(item, insideSnapshot)
			}
		case xpath3.LocationPath:
			for _, step := range v.Steps {
				for _, pred := range step.Predicates {
					walk(pred, insideSnapshot)
				}
			}
		case xpath3.SimpleMapExpr:
			walk(v.Left, insideSnapshot)
			walk(v.Right, insideSnapshot)
		case xpath3.LiteralExpr, xpath3.VariableExpr, xpath3.RootExpr,
			xpath3.ContextItemExpr, xpath3.PlaceholderExpr:
			// leaf nodes — nothing to walk
		}
	}
	walk(expr.AST(), false)
	return found
}

// exprUsesCurrentGroup returns true if the expression calls current-group() anywhere.
func exprUsesCurrentGroup(expr *xpath3.Expression) bool {
	return xpathstream.ExprUsesFunction(expr, lexicon.FnCurrentGroup)
}

// checkMapStreamable checks xsl:map instructions for streamability violations.
//
// Per the XSLT 3.0 spec, xsl:map with all xsl:map-entry children acts as an
// implicit fork: each map-entry is processed independently, so multiple
// consuming entries are allowed. However:
//   - A single map-entry whose key AND select/body are both consuming is invalid
//     (multiple consuming references within a single fork branch).
//   - A map-entry whose select returns ungrounded streaming nodes is invalid
//     because node references become dangling after the fork branch completes.
//   - Non-map-entry children (like xsl:if wrapping map-entry) break the implicit
//     fork guarantee and are invalid when consuming entries are present.
func checkMapStreamable(_ *Stylesheet, inst instruction) error {
	mapInst, ok := inst.(*mapInst)
	if !ok {
		return nil
	}

	consumingEntries := 0
	hasNonEntryChildren := false
	for _, child := range mapInst.Body {
		if me, ok := child.(*mapEntryInst); ok {
			// Check if key and select/body EACH consume the stream.
			// Use ExprHasDownwardStep which counts downward steps even
			// inside grounding functions — each still consumes the stream.
			keyConsuming := me.Key != nil && xpathstream.ExprHasDownwardStep(me.Key)
			selectConsuming := me.Select != nil && xpathstream.ExprHasDownwardStep(me.Select)
			bodyConsuming := false
			for _, meChild := range me.Body {
				for _, expr := range getInstructionExprs(meChild) {
					if expr != nil && xpathstream.ExprHasDownwardStep(expr) {
						bodyConsuming = true
						break
					}
				}
				if bodyConsuming {
					break
				}
			}

			// Within a single map-entry (one fork branch), key and
			// select/body must not both consume the stream.
			consumingParts := 0
			if keyConsuming {
				consumingParts++
			}
			if selectConsuming || bodyConsuming {
				consumingParts++
			}
			if consumingParts > 1 {
				return staticError(errCodeXTSE3430,
					"xsl:map-entry has multiple consuming expressions (key and value), which is not streamable")
			}

			// A map-entry select that navigates downward into the streaming
			// source but does not ground or atomize the result produces
			// streaming nodes that become invalid after the branch completes.
			if me.Select != nil && argHasStreamingDownwardUngrounded(me.Select.AST()) {
				return staticError(errCodeXTSE3430,
					"xsl:map-entry select returns ungrounded streaming nodes, which is not streamable")
			}

			if keyConsuming || selectConsuming || bodyConsuming {
				consumingEntries++
			}
		} else {
			hasNonEntryChildren = true
		}
	}

	if hasNonEntryChildren && consumingEntries > 0 {
		return staticError(errCodeXTSE3430,
			"xsl:map has non-map-entry children with consuming map entries, which is not streamable")
	}

	return nil
}

// checkIterateStreamable checks xsl:iterate for violations where streamed nodes
// are accumulated via xsl:with-param, creating non-streamable patterns.
func checkIterateStreamable(_ *Stylesheet, inst instruction) error {
	iter, ok := inst.(*iterateInst)
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
func bodyAccumulatesStreamedNodes(body []instruction, elemParams map[string]bool) bool {
	// Collect variables whose select expressions reference "." (context item),
	// either directly or transitively through other variables.
	taintedVars := collectTaintedVars(body)

	return bodyAccumulatesStreamedNodesInner(body, elemParams, taintedVars)
}

// collectTaintedVars finds all variables in the instruction tree whose select
// expressions reference the context item "." either directly or transitively
// through other variables that reference ".".
func collectTaintedVars(body []instruction) map[string]bool {
	// First pass: collect all variable select expressions.
	varExprs := make(map[string]xpath3.Expr)
	collectVarExprs(body, varExprs)

	// Build direct taint set: variables whose select directly contains ".".
	tainted := make(map[string]bool)
	for name, expr := range varExprs {
		if exprReferencesContextItem(expr) {
			tainted[name] = true
		}
	}

	// Propagate taint: if a variable references a tainted variable, it's tainted too.
	// Fixed-point iteration.
	changed := true
	for changed {
		changed = false
		for name, expr := range varExprs {
			if tainted[name] {
				continue
			}
			if exprReferencesAnyVar(expr, tainted) {
				tainted[name] = true
				changed = true
			}
		}
	}

	return tainted
}

// collectVarExprs collects variable name→expression mappings from instructions.
func collectVarExprs(body []instruction, varExprs map[string]xpath3.Expr) {
	for _, inst := range body {
		if vi, ok := inst.(*variableInst); ok && vi.Select != nil {
			varExprs[vi.Name] = vi.Select.AST()
		}
		for _, children := range getChildInstructions(inst) {
			collectVarExprs(children, varExprs)
		}
	}
}

// exprReferencesAnyVar returns true if the expression references any variable
// in the given set.
func exprReferencesAnyVar(expr xpath3.Expr, vars map[string]bool) bool {
	found := false
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if ve, ok := e.(xpath3.VariableExpr); ok {
			if vars[ve.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func bodyAccumulatesStreamedNodesInner(body []instruction, elemParams map[string]bool, taintedVars map[string]bool) bool {
	for _, inst := range body {
		if ni, ok := inst.(*nextIterationInst); ok {
			for _, wp := range ni.Params {
				if !elemParams[wp.Name] {
					continue
				}
				if wp.Select != nil {
					ast := wp.Select.AST()
					if exprReferencesContextItem(ast) {
						return true
					}
					if exprReferencesAnyVar(ast, taintedVars) {
						return true
					}
				}
			}
		}
		// Recurse into child instructions (e.g., inside choose/if)
		for _, children := range getChildInstructions(inst) {
			if bodyAccumulatesStreamedNodesInner(children, elemParams, taintedVars) {
				return true
			}
		}
	}
	return false
}

// exprReferencesContextItem returns true if the expression references "." (context item).
func exprReferencesContextItem(expr xpath3.Expr) bool {
	found := false
	xpathstream.WalkExpr(expr, func(e xpath3.Expr) bool {
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
