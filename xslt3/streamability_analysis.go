package xslt3

import (
	"slices"

	"github.com/lestrrat-go/helium/internal/xpathstream"
	"github.com/lestrrat-go/helium/xpath3"
)

// lookupFuncStreamability returns the streamability annotation of a user-defined
// function matching the given local name and arity, or "" if not found.
// Since FunctionCall AST nodes store only prefix:localName (not namespace URI),
// we match by local name and arity. This is safe because user-defined XSLT
// functions always use a non-null namespace and duplicates by arity are rejected.
func lookupFuncStreamability(ss *Stylesheet, localName string, arity int) string {
	if ss == nil {
		return ""
	}
	for key, fn := range ss.functions {
		if key.Name.Name == localName && key.Arity == arity && fn.Streamability != "" {
			return fn.Streamability
		}
	}
	return ""
}

// analyzeStreamability performs a post-compilation pass over the stylesheet.
// It checks templates in streamable modes and bodies of xsl:source-document
// streamable="yes" for non-streamable constructs. Per the W3C spec, a Basic
// XSLT Processor that cannot stream a construct should fall back to
// non-streaming (DOM-based) execution rather than raising a fatal error.
// When an XTSE3430 violation is detected, the affected mode, source-document,
// accumulator, or function is demoted to non-streamable and compilation
// continues.
func analyzeStreamability(ss *Stylesheet) error { //nolint:unparam // always nil but callers check for future-proofing
	// Check all templates for source-document streamable="yes" in their body.
	// On XTSE3430, demote the source-document to non-streamable.
	for _, tmpl := range ss.templates {
		demoteStreamableSourceDocs(ss, tmpl.Body)
	}

	// Check templates in streamable modes for non-streamable constructs.
	// On XTSE3430, demote the entire mode to non-streamable (fallback).
	for _, tmpl := range ss.templates {
		modeName, ok := streamabilityModeNameForTemplate(tmpl)
		if !ok || modeName == modeAll {
			continue
		}
		md := ss.modeDefs[modeName]
		if md == nil || !md.Streamable {
			continue
		}
		// Match patterns in streaming modes must be motionless — positional
		// predicates like [1], [last()], or predicates that navigate downward
		// are not allowed (XTSE3430).
		if tmpl.Match != nil {
			for _, alt := range tmpl.Match.Alternatives {
				if xpathstream.ExprTreeHasNonMotionlessPredicate(alt.expr) {
					md.Streamable = false
					break
				}
			}
		}
		if !md.Streamable {
			continue
		}
		if err := checkStreamableTemplateBody(ss, tmpl.Body); err != nil {
			md.Streamable = false
			continue
		}
		// In streaming templates, accumulator-after() is only valid in
		// post-descent position (after a consuming operation). If used
		// before any consuming operation, the post-descent values are
		// not yet available.
		if err := checkAccumulatorAfterPreDescent(tmpl.Body); err != nil {
			md.Streamable = false
			continue
		}
	}

	// Check streamable accumulators: initial-value must be motionless.
	// On XTSE3430, demote the accumulator to non-streamable.
	for _, acc := range ss.accumulators {
		if !acc.Streamable {
			continue
		}
		if acc.Initial != nil {
			if err := checkStreamableExpr(ss, acc.Initial); err != nil {
				acc.Streamable = false
				continue
			}
			// A motionless expression must not navigate the document at all.
			if xpathstream.ExprHasDownwardStep(acc.Initial) {
				acc.Streamable = false
				continue
			}
		}
		// Accumulator rule match patterns must be motionless in streaming.
		demoted := false
		for _, rule := range acc.Rules {
			if rule.Match == nil {
				continue
			}
			for _, alt := range rule.Match.Alternatives {
				if xpathstream.ExprTreeHasNonMotionlessPredicate(alt.expr) {
					acc.Streamable = false
					demoted = true
					break
				}
				if patternHasCurrentOnElementStep(alt.expr) {
					acc.Streamable = false
					demoted = true
					break
				}
			}
			if demoted {
				break
			}
			// Accumulator rule select expressions must not navigate
			// downward (to children/descendants) in streaming mode.
			// Also, using the context item (.) on an element match is
			// consuming because the element may have unprocessed children.
			// For text/attribute matches, "." is fine (the value is atomic).
			if rule.Select != nil {
				if xpathstream.ExprHasDownwardStep(rule.Select) {
					acc.Streamable = false
					demoted = true
					break
				}
				if xpathstream.ExprUsesContextItem(rule.Select) && accRuleMatchesElement(rule) {
					acc.Streamable = false
					demoted = true
					break
				}
			}
		}
	}

	// Check functions with declared streamability.
	// On XTSE3430, clear the streamability annotation.
	for _, fn := range ss.functions {
		if fn.Streamability != "" {
			if err := checkStreamableFunctionBody(ss, fn); err != nil {
				fn.Streamability = ""
			}
		}
	}

	return nil
}

// demoteStreamableSourceDocs walks instructions looking for sourceDocumentInst
// with Streamable=true and checks their bodies. On XTSE3430, the
// source-document is demoted to non-streamable (fallback to DOM).
func demoteStreamableSourceDocs(ss *Stylesheet, instructions []instruction) {
	for _, inst := range instructions {
		if sd, ok := inst.(*sourceDocumentInst); ok && sd.Streamable {
			if err := checkStreamableTemplateBody(ss, sd.Body); err != nil {
				sd.Streamable = false
			}
		}
		// Recurse into any nested instruction bodies to find source-document.
		for _, child := range getChildInstructions(inst) {
			demoteStreamableSourceDocs(ss, child)
		}
	}
}

// streamabilityModeNameForTemplate returns the mode name that should be used
// for streamability checks. Only match templates participate in mode-based
// streamability; named templates are handled separately where relevant.
func streamabilityModeNameForTemplate(tmpl *template) (string, bool) {
	if tmpl == nil || tmpl.Match == nil {
		return "", false
	}
	if tmpl.Mode == "" {
		return modeDefault, true
	}
	return tmpl.Mode, true
}

// checkStreamableTemplateBody checks a template body (or source-document body)
// for non-streamable constructs.
// checkAccumulatorAfterPreDescent checks that accumulator-after() is not
// used in AVTs of literal result elements before a consuming operation in
// a streaming template body. AVTs are evaluated eagerly, so post-descent
// values are not available. xsl:attribute instructions with accumulator-after
// are fine because the processor can delay their evaluation.
func checkAccumulatorAfterPreDescent(body []instruction) error {
	_, err := checkAccAfterPreDescentInner(body)
	return err
}

// checkAccAfterPreDescentInner checks for accumulator-after in pre-descent
// position. Returns (consumed, error) where consumed indicates whether a
// consuming operation was found at this level or in children.
func checkAccAfterPreDescentInner(body []instruction) (bool, error) {
	consumed := false
	for _, inst := range body {
		if consumed {
			// After a consuming operation, accumulator-after is valid.
			return true, nil
		}
		// Check if this instruction is directly consuming.
		if instrIsConsuming(inst) {
			consumed = true
			continue
		}
		// Check AVTs in literal result elements.
		if err := checkAccumulatorAfterInLRE(inst); err != nil {
			return false, err
		}
		// Check xsl:value-of select expressions (pre-descent, so
		// accumulator-after is not available). xsl:attribute select
		// is NOT checked here because attributes can be delayed.
		if vo, ok := inst.(*valueOfInst); ok {
			if vo.Select != nil && xpathstream.ExprUsesFunction(vo.Select, "accumulator-after") {
				return false, staticError(errCodeXTSE3430,
					"accumulator-after() used in pre-descent position is not streamable")
			}
		}
		// Recurse into child instructions (e.g., LRE body, copy body).
		for _, children := range getChildInstructions(inst) {
			childConsumed, err := checkAccAfterPreDescentInner(children)
			if err != nil {
				return false, err
			}
			if childConsumed {
				consumed = true
			}
		}
	}
	return consumed, nil
}

// instrIsConsuming returns true if the instruction directly consumes the
// streaming context (navigates downward or processes children).
func instrIsConsuming(inst instruction) bool {
	switch v := inst.(type) {
	case *applyTemplatesInst:
		// apply-templates without select or with downward select is consuming.
		// apply-templates select="@*" is motionless, NOT consuming.
		if v.Select == nil {
			return true
		}
		return xpathstream.ExprHasDownwardStep(v.Select) || xpathstream.ExprUsesContextItem(v.Select)
	case *forEachInst:
		return v.Select != nil && xpathstream.ExprHasDownwardStep(v.Select)
	case *iterateInst:
		return v.Select != nil && xpathstream.ExprHasDownwardStep(v.Select)
	case *valueOfInst:
		return v.Select != nil && xpathstream.ExprHasDownwardStep(v.Select)
	case *xslSequenceInst:
		return v.Select != nil && xpathstream.ExprHasDownwardStep(v.Select)
	case *copyOfInst:
		return v.Select != nil && (xpathstream.ExprHasDownwardStep(v.Select) || xpathstream.ExprUsesContextItem(v.Select))
	}
	return false
}

// checkAccumulatorAfterInLRE checks literal result elements for
// accumulator-after() in their avt attributes.
func checkAccumulatorAfterInLRE(inst instruction) error {
	lre, ok := inst.(*literalResultElement)
	if !ok {
		return nil
	}
	for _, attr := range lre.Attrs {
		if attr.Value != nil && attr.Value.hasFunction("accumulator-after") {
			return staticError(errCodeXTSE3430,
				"accumulator-after() used in pre-descent position is not streamable")
		}
	}
	return nil
}

func checkStreamableTemplateBody(ss *Stylesheet, body []instruction) error {
	for _, inst := range body {
		if err := checkStreamableInstruction(ss, inst); err != nil {
			return err
		}
	}

	// Check for xsl:sequence at the template body level that returns
	// streaming nodes. Inside an LRE or xsl:element, nodes are consumed
	// into the output and are fine; at the top level they would be returned
	// to the caller, which is not streamable.
	for _, inst := range body {
		seq, ok := inst.(*xslSequenceInst)
		if !ok || seq.Select == nil {
			continue
		}
		if exprTopLevelReturnsStreamingNodes(seq.Select.AST()) {
			return staticError(errCodeXTSE3430,
				"xsl:sequence select %q returns streaming nodes from template body, which is not streamable",
				seq.Select.String())
		}
	}

	// Check for variables bound to the streaming context that are used
	// consumingly in subsequent loop bodies.
	if err := checkStreamingVarInLoop(body); err != nil {
		return err
	}

	// Check for multiple consuming (downward) operations across the template
	// body as a whole. Two sibling copy-of/value-of/apply-templates each
	// selecting child nodes constitute two consuming reads of the stream.
	if countDownwardInBody(ss, body) > 1 {
		return staticError(errCodeXTSE3430,
			"template body has multiple consuming operations, which is not streamable")
	}

	return nil
}

// countDownwardInBody counts consuming downward selections across a sequence
// of sibling instructions, respecting branching: only the max across
// xsl:choose branches is counted since only one branch executes.
func countDownwardInBody(ss *Stylesheet, body []instruction) int {
	total := 0
	for _, inst := range body {
		if choose, ok := inst.(*chooseInst); ok {
			maxBranch := 0
			for _, when := range choose.When {
				bc := countDownwardInBody(ss, when.Body)
				if bc > maxBranch {
					maxBranch = bc
				}
			}
			if choose.Otherwise != nil {
				bc := countDownwardInBody(ss, choose.Otherwise)
				if bc > maxBranch {
					maxBranch = bc
				}
			}
			total += maxBranch
			continue
		}
		if ifInst, ok := inst.(*ifInst); ok {
			// xsl:if is like a single branch — its body may or may
			// not execute, but from a streamability perspective the
			// consuming operations inside still count.
			total += countDownwardInBody(ss, ifInst.Body)
			continue
		}
		for _, expr := range getInstructionExprs(inst) {
			total += countStreamingDownwardSelections(ss, expr.AST())
		}
	}
	return total
}

// checkStreamableInstruction checks a single instruction for streamability violations.
func checkStreamableInstruction(ss *Stylesheet, inst instruction) error {
	return checkStreamableInstructionCtx(ss, inst, false)
}

// checkStreamableInstructionCtx checks a single instruction for streamability violations.
// When inResultDoc is true, certain checks are relaxed (e.g., xsl:sequence select="."
// inside for-each is allowed because nodes flow to a serializer rather than being
// returned as a sequence).
func checkStreamableInstructionCtx(ss *Stylesheet, inst instruction, inResultDoc bool) error {
	switch v := inst.(type) {
	case *applyTemplatesInst:
		if v.Select != nil && xpathstream.ExprUsesDescendantOrSelf(v.Select) &&
			!exprEndsWithGrounding(v.Select.AST()) {
			return staticError(errCodeXTSE3430,
				"xsl:apply-templates with crawling select expression %q is not streamable", v.Select.String())
		}
	case *nextMatchInst:
		// xsl:next-match with xsl:with-param select="." passes the streaming
		// context item to another template. The receiving template can navigate
		// into it, creating a second consumption of the streaming node. This is
		// not streamable.
		for _, wp := range v.Params {
			if wp.Select != nil {
				if _, ok := derefXPathExpr(wp.Select.AST()).(xpath3.ContextItemExpr); ok {
					return staticError(errCodeXTSE3430,
						"xsl:next-match with xsl:with-param select=\".\" passes streaming context item, which is not streamable")
				}
			}
		}
	case *numberInst:
		// xsl:number without an explicit value computes numbering from node
		// relationships, which is consuming and therefore not streamable.
		if v.Value == nil {
			return staticError(errCodeXTSE3430,
				"xsl:number without value is not streamable")
		}
	}

	for _, expr := range getInstructionExprs(inst) {
		if err := checkStreamableExpr(ss, expr); err != nil {
			return err
		}
	}

	// Check for multiple downward selections across sibling expressions in
	// fork branches and map entries (they each consume the stream).
	if err := checkMultipleDownwardInst(ss, inst); err != nil {
		return err
	}

	// Check attribute sets used in streaming context.
	if err := checkUseAttributeSetsStreamable(ss, inst); err != nil {
		return err
	}

	// Check for-each/iterate with crawling select or variable-bound streaming in loop.
	if err := checkForEachStreamable(ss, inst, inResultDoc); err != nil {
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
	// last(), position(), etc. are allowed. However, when the select navigates
	// upward (parent/ancestor), check for downward navigation in the body only
	// (up-then-down is not streamable).
	if fe, ok := inst.(*forEachInst); ok {
		if fe.Select != nil && !xpathstream.ExprHasDownwardStep(fe.Select) && !xpathstream.ExprUsesDescendantOrSelf(fe.Select) {
			// Check up-then-down: for-each navigates upward and body goes down.
			if xpathstream.ExprUsesUpwardAxis(fe.Select) && bodyHasDownwardNavigation(fe.Body) {
				return staticError(errCodeXTSE3430,
					"xsl:for-each select %q navigates upward and body navigates downward, which is not streamable (up-then-down)",
					fe.Select.String())
			}
			// Body of for-each over motionless/upward axis — skip streaming checks.
			return nil
		}
	}
	// When entering a result-document, propagate the inResultDoc flag so
	// that child for-each bodies allow xsl:sequence select="." (nodes flow
	// to a serializer).
	childInResultDoc := inResultDoc
	if _, ok := inst.(*resultDocumentInst); ok {
		childInResultDoc = true
	}
	for _, children := range getChildInstructions(inst) {
		for _, child := range children {
			if err := checkStreamableInstructionCtx(ss, child, childInResultDoc); err != nil {
				return err
			}
		}
		// Also check for streaming variable-in-loop violations within
		// nested instruction bodies (e.g., inside LREs).
		if err := checkStreamingVarInLoop(children); err != nil {
			return err
		}
	}

	return nil
}

// exprTopLevelReturnsStreamingNodes returns true if the top-level expression
// returns nodes from the streaming document (path/step expressions that select
// child/descendant elements). Does NOT flag function calls that consume nodes
// internally and return atomic values (e.g., string-join, count).
func exprTopLevelReturnsStreamingNodes(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch v := expr.(type) {
	case xpath3.LocationPath:
		for _, step := range v.Steps {
			switch step.Axis {
			case xpath3.AxisChild, xpath3.AxisDescendant, xpath3.AxisDescendantOrSelf:
				return true
			}
		}
	case xpath3.PathExpr:
		// A path whose filter is a grounding function (snapshot, copy-of)
		// returns grounded data, not streaming nodes.
		if isGroundingExpr(v.Filter) {
			return false
		}
		return true
	case xpath3.PathStepExpr:
		// Walk left to find the base of the path chain. If the leftmost
		// expression is grounding, the entire chain operates on grounded
		// data and does not return streaming nodes.
		if exprPathBaseIsGrounding(v) {
			return false
		}
		return true
	case xpath3.IfExpr:
		return exprTopLevelReturnsStreamingNodes(v.Then) || exprTopLevelReturnsStreamingNodes(v.Else)
	case xpath3.ContextItemExpr:
		return true // "." returns the streaming node itself
	}
	return false
}

// exprPathBaseIsGrounding walks a PathStepExpr chain to find the leftmost
// base expression and returns true if it is a grounding function call
// (snapshot, copy-of, etc.). When the base is grounding, subsequent path
// steps navigate grounded data, not the streaming source.
func exprPathBaseIsGrounding(expr xpath3.Expr) bool {
	for {
		expr = derefXPathExpr(expr)
		switch e := expr.(type) {
		case xpath3.PathStepExpr:
			expr = e.Left
		case xpath3.PathExpr:
			return isGroundingExpr(e.Filter)
		case xpath3.FunctionCall:
			return isGroundingExpr(e)
		case xpath3.FilterExpr:
			expr = e.Expr
		default:
			return false
		}
	}
}

// bodyHasDownwardNavigation returns true if any instruction in the body
// has an expression that navigates downward (child/descendant axis).
func bodyHasDownwardNavigation(body []instruction) bool {
	for _, inst := range body {
		for _, expr := range getInstructionExprs(inst) {
			if expr == nil {
				continue
			}
			if xpathstream.ExprHasDownwardStep(expr) {
				return true
			}
		}
		if slices.ContainsFunc(getChildInstructions(inst), bodyHasDownwardNavigation) {
			return true
		}
	}
	return false
}
