package xslt3

import (
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
				if xpath3.ExprTreeHasNonMotionlessPredicate(alt.expr) {
					return staticError(errCodeXTSE3430,
						"match pattern %q has a non-motionless predicate, which is not allowed in streaming mode %q",
						tmpl.Match.source, modeName)
				}
			}
		}
		if err := checkStreamableTemplateBody(ss, tmpl.Body); err != nil {
			return err
		}
		// In streaming templates, accumulator-after() is only valid in
		// post-descent position (after a consuming operation). If used
		// before any consuming operation, the post-descent values are
		// not yet available.
		if err := checkAccumulatorAfterPreDescent(tmpl.Body); err != nil {
			return err
		}
	}

	// Check streamable accumulators: initial-value must be motionless.
	for _, acc := range ss.accumulators {
		if !acc.Streamable {
			continue
		}
		if acc.Initial != nil {
			if err := checkStreamableExpr(ss, acc.Initial); err != nil {
				return err
			}
			// A motionless expression must not navigate the document at all.
			if xpath3.ExprHasDownwardStep(acc.Initial) {
				return staticError(errCodeXTSE3430,
					"streamable accumulator %q has non-motionless initial-value expression %q",
					acc.Name, acc.Initial.String())
			}
		}
		// Accumulator rule match patterns must be motionless in streaming.
		for _, rule := range acc.Rules {
			if rule.Match == nil {
				continue
			}
			for _, alt := range rule.Match.Alternatives {
				if xpath3.ExprTreeHasNonMotionlessPredicate(alt.expr) {
					return staticError(errCodeXTSE3430,
						"streamable accumulator %q rule match pattern %q has a non-motionless predicate",
						acc.Name, rule.Match.source)
				}
				if patternHasCurrentOnElementStep(alt.expr) {
					return staticError(errCodeXTSE3430,
						"streamable accumulator %q rule match pattern %q uses current() on an element-matching step, which is not motionless",
						acc.Name, rule.Match.source)
				}
			}
			// Accumulator rule select expressions must not navigate
			// downward (to children/descendants) in streaming mode.
			// Also, using the context item (.) on an element match is
			// consuming because the element may have unprocessed children.
			// For text/attribute matches, "." is fine (the value is atomic).
			if rule.Select != nil {
				if xpath3.ExprHasDownwardStep(rule.Select) {
					return staticError(errCodeXTSE3430,
						"streamable accumulator %q has a non-streamable select expression %q",
						acc.Name, rule.Select.String())
				}
				if xpath3.ExprUsesContextItem(rule.Select) && accRuleMatchesElement(rule) {
					return staticError(errCodeXTSE3430,
						"streamable accumulator %q rule select expression %q uses the context item on an element match, which is consuming in streaming mode",
						acc.Name, rule.Select.String())
				}
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
func streamabilityModeNameForTemplate(tmpl *template) (string, bool) {
	if tmpl == nil || tmpl.Match == nil {
		return "", false
	}
	if tmpl.Mode == "" {
		return modeDefault, true
	}
	return tmpl.Mode, true
}

// checkInstructionsForStreamableSourceDoc walks instructions looking for
// sourceDocumentInst with Streamable=true and checks their bodies.
func checkInstructionsForStreamableSourceDoc(ss *Stylesheet, instructions []instruction) error {
	for _, inst := range instructions {
		if sd, ok := inst.(*sourceDocumentInst); ok && sd.Streamable {
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
			if vo.Select != nil && xpath3.ExprUsesFunction(vo.Select, "accumulator-after") {
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
		return xpath3.ExprHasDownwardStep(v.Select) || xpath3.ExprUsesContextItem(v.Select)
	case *forEachInst:
		return v.Select != nil && xpath3.ExprHasDownwardStep(v.Select)
	case *iterateInst:
		return v.Select != nil && xpath3.ExprHasDownwardStep(v.Select)
	case *valueOfInst:
		return v.Select != nil && xpath3.ExprHasDownwardStep(v.Select)
	case *xslSequenceInst:
		return v.Select != nil && xpath3.ExprHasDownwardStep(v.Select)
	case *copyOfInst:
		return v.Select != nil && (xpath3.ExprHasDownwardStep(v.Select) || xpath3.ExprUsesContextItem(v.Select))
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
		if v.Select != nil && xpath3.ExprUsesDescendantOrSelf(v.Select) &&
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
	// last(), position(), etc. are allowed.
	if fe, ok := inst.(*forEachInst); ok {
		if fe.Select != nil && !xpath3.ExprHasDownwardStep(fe.Select) && !xpath3.ExprUsesDescendantOrSelf(fe.Select) {
			// Body of for-each over motionless/attribute axis — skip streaming checks.
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
