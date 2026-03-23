package xslt3

import (
	"strings"

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
func streamabilityModeNameForTemplate(tmpl *Template) (string, bool) {
	if tmpl == nil || tmpl.Match == nil {
		return "", false
	}
	if tmpl.Mode == "" {
		return modeDefault, true
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
// checkAccumulatorAfterPreDescent checks that accumulator-after() is not
// used in AVTs of literal result elements before a consuming operation in
// a streaming template body. AVTs are evaluated eagerly, so post-descent
// values are not available. xsl:attribute instructions with accumulator-after
// are fine because the processor can delay their evaluation.
func checkAccumulatorAfterPreDescent(body []Instruction) error {
	_, err := checkAccAfterPreDescentInner(body)
	return err
}

// checkAccAfterPreDescentInner checks for accumulator-after in pre-descent
// position. Returns (consumed, error) where consumed indicates whether a
// consuming operation was found at this level or in children.
func checkAccAfterPreDescentInner(body []Instruction) (bool, error) {
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
		if vo, ok := inst.(*ValueOfInst); ok {
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
func instrIsConsuming(inst Instruction) bool {
	switch v := inst.(type) {
	case *ApplyTemplatesInst:
		// apply-templates without select or with downward select is consuming.
		// apply-templates select="@*" is motionless, NOT consuming.
		if v.Select == nil {
			return true
		}
		return xpath3.ExprHasDownwardStep(v.Select) || xpath3.ExprUsesContextItem(v.Select)
	case *ForEachInst:
		return v.Select != nil && xpath3.ExprHasDownwardStep(v.Select)
	case *IterateInst:
		return v.Select != nil && xpath3.ExprHasDownwardStep(v.Select)
	case *ValueOfInst:
		return v.Select != nil && xpath3.ExprHasDownwardStep(v.Select)
	case *XSLSequenceInst:
		return v.Select != nil && xpath3.ExprHasDownwardStep(v.Select)
	case *CopyOfInst:
		return v.Select != nil && (xpath3.ExprHasDownwardStep(v.Select) || xpath3.ExprUsesContextItem(v.Select))
	}
	return false
}

// checkAccumulatorAfterInLRE checks literal result elements for
// accumulator-after() in their AVT attributes.
func checkAccumulatorAfterInLRE(inst Instruction) error {
	lre, ok := inst.(*LiteralResultElement)
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
func countDownwardInBody(ss *Stylesheet, body []Instruction) int {
	total := 0
	for _, inst := range body {
		if choose, ok := inst.(*ChooseInst); ok {
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
		if ifInst, ok := inst.(*IfInst); ok {
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
func checkStreamableInstruction(ss *Stylesheet, inst Instruction) error {
	return checkStreamableInstructionCtx(ss, inst, false)
}

// checkStreamableInstructionCtx checks a single instruction for streamability violations.
// When inResultDoc is true, certain checks are relaxed (e.g., xsl:sequence select="."
// inside for-each is allowed because nodes flow to a serializer rather than being
// returned as a sequence).
func checkStreamableInstructionCtx(ss *Stylesheet, inst Instruction, inResultDoc bool) error {
	switch v := inst.(type) {
	case *ApplyTemplatesInst:
		if v.Select != nil && xpath3.ExprUsesDescendantOrSelf(v.Select) &&
			!exprEndsWithGrounding(v.Select.AST()) {
			return staticError(errCodeXTSE3430,
				"xsl:apply-templates with crawling select expression %q is not streamable", v.Select.String())
		}
	case *NextMatchInst:
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
	case *NumberInst:
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
	if fe, ok := inst.(*ForEachInst); ok {
		if fe.Select != nil && !xpath3.ExprHasDownwardStep(fe.Select) && !xpath3.ExprUsesDescendantOrSelf(fe.Select) {
			// Body of for-each over motionless/attribute axis — skip streaming checks.
			return nil
		}
	}
	// When entering a result-document, propagate the inResultDoc flag so
	// that child for-each bodies allow xsl:sequence select="." (nodes flow
	// to a serializer).
	childInResultDoc := inResultDoc
	if _, ok := inst.(*ResultDocumentInst); ok {
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

// checkStreamableExpr checks a single XPath expression for non-streamable patterns.
func checkStreamableExpr(ss *Stylesheet, expr *xpath3.Expression) error {
	if expr == nil {
		return nil
	}

	// Parent/ancestor axes are motionless — ancestors of the streaming context
	// are always available. However, preceding/preceding-sibling axes require
	// backward access to already-consumed nodes, which is non-streamable.
	if xpath3.ExprUsesPrecedingAxis(expr) {
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
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		if fc, ok := e.(xpath3.FunctionCall); ok && fc.Prefix != "" && len(fc.Args) > 0 {
			cat := lookupFuncStreamability(ss, fc.Name, len(fc.Args))
			if cat == "shallow-descent" {
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
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
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
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
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
// context. Variable references, literals, and striding expressions (child-only
// paths) are not crawling.
func seqItemIsCrawling(expr xpath3.Expr) bool {
	expr = derefXPathExpr(expr)
	switch expr.(type) {
	case xpath3.VariableExpr, xpath3.LiteralExpr:
		return false
	}
	found := false
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
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
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
		if found {
			return false
		}
		fc, ok := e.(xpath3.FunctionCall)
		if !ok || fc.Prefix != "" || len(fc.Args) < 2 {
			return true
		}
		switch fc.Name {
		case "filter", "fold-right":
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
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
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
func accRuleMatchesElement(rule *AccumulatorRule) bool {
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
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
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
	xpath3.WalkExpr(expr.AST(), func(e xpath3.Expr) bool {
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
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
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
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
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
	case "snapshot", "copy-of", "copy", "current-group",
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
		// Variable references are always grounded — they hold materialized
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
			xpath3.WalkExpr(e, func(child xpath3.Expr) bool {
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

// checkMultipleDownwardInst checks if an instruction has multiple consuming
// operations that would require reading the stream multiple times.
func checkMultipleDownwardInst(ss *Stylesheet, inst Instruction) error {
	switch v := inst.(type) {
	case *ForkInst:
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
				if seq, ok := bi.(*XSLSequenceInst); ok && seq.Select != nil {
					ast := seq.Select.AST()
					// A branch returns streaming nodes if it selects
					// downward without grounding or atomizing the result.
					if xpath3.ExprHasDownwardStep(seq.Select) &&
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

	case *IterateInst:
		// Within xsl:iterate body, check for multiple downward selections.
		bodyDown := countDownwardInInstructions(ss, v.Body)
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
				if lre, ok := bi.(*LiteralResultElement); ok {
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
func checkApplyTemplatesCurrentGroup(ss *Stylesheet, body []Instruction) error {
	for _, inst := range body {
		if at, ok := inst.(*ApplyTemplatesInst); ok {
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
func bodyUsesCurrentGroupOutsideFEG(body []Instruction) bool {
	for _, bi := range body {
		// Skip for-each-group — its body establishes a new group context.
		if _, ok := bi.(*ForEachGroupInst); ok {
			continue
		}
		for _, expr := range getInstructionExprs(bi) {
			if expr == nil {
				continue
			}
			if xpath3.ExprUsesFunction(expr, "current-group") {
				return true
			}
		}
		// Check LRE attribute AVTs (not covered by getInstructionExprs).
		if lre, ok := bi.(*LiteralResultElement); ok {
			for _, attr := range lre.Attrs {
				if attr.Value.hasFunction("current-group") {
					return true
				}
			}
		}
		for _, children := range getChildInstructions(bi) {
			if bodyUsesCurrentGroupOutsideFEG(children) {
				return true
			}
		}
	}
	return false
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
func countAttributeSetDownward(ss *Stylesheet, asDef *AttributeSetDef) int {
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
func checkForEachStreamable(_ *Stylesheet, inst Instruction, inResultDoc bool) error {
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
		// Check for xsl:sequence select="." returning streamed nodes.
		// Skip this check when inside xsl:result-document — nodes are
		// written to a secondary output (serializer), not returned.
		if !inResultDoc {
			for _, bi := range v.Body {
				if seq, ok := bi.(*XSLSequenceInst); ok && seq.Select != nil {
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
//
// Grounding instructions (xsl:copy select=".", xsl:copy-of select=".",
// xsl:copy without select) are excluded because they produce deep copies
// of the current node and are allowed inside a crawling for-each.
func forEachBodyConsumesContext(body []Instruction) bool {
	for _, inst := range body {
		// xsl:copy with select="." or no select is a grounding operation.
		if ci, ok := inst.(*CopyInst); ok {
			if ci.Select == nil {
				continue // xsl:copy (shallow copy + body) — grounding
			}
			if _, isCtx := ci.Select.AST().(xpath3.ContextItemExpr); isCtx {
				continue // xsl:copy select="." — grounding
			}
		}
		// xsl:copy-of select="." is a grounding operation.
		if coi, ok := inst.(*CopyOfInst); ok && coi.Select != nil {
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
			if exprReferencesContextItem(expr.AST()) || xpath3.ExprHasDownwardStep(expr) {
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
	}

	// Crawling selects (descendant-or-self axis) with consuming use of
	// current-group() in the body are not streamable — the processor would
	// need to buffer unbounded amounts of data.  When the select is grounded,
	// items are already in memory and crawling is fine.
	if !selectGrounded && fg.Select != nil && xpath3.ExprUsesDescendantOrSelf(fg.Select) {
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
func checkSourceDocCurrentGroup(body []Instruction) error {
	for _, bi := range body {
		if sd, ok := bi.(*SourceDocumentInst); ok && sd.Streamable {
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
func sourceDocBodyUsesCurrentGroup(body []Instruction) bool {
	for _, inst := range body {
		for _, expr := range getInstructionExprs(inst) {
			if exprUsesCurrentGroup(expr) {
				return true
			}
		}
		for _, children := range getChildInstructions(inst) {
			if sourceDocBodyUsesCurrentGroup(children) {
				return true
			}
		}
	}
	return false
}

// patternHasNonMotionlessPredicate returns true if any alternative in the
// pattern has a predicate that navigates downward (child/descendant steps).
func patternHasNonMotionlessPredicate(pat *Pattern) bool {
	for _, alt := range pat.Alternatives {
		if alt.expr == nil {
			continue
		}
		nonMotionless := false
		xpath3.WalkExpr(alt.expr, func(e xpath3.Expr) bool {
			if nonMotionless {
				return false
			}
			switch v := e.(type) {
			case xpath3.LocationPath:
				for _, step := range v.Steps {
					for _, pred := range step.Predicates {
						if xpath3.PredicateIsNonMotionlessWithStep(pred, &step) {
							nonMotionless = true
							return false
						}
					}
				}
			case xpath3.FilterExpr:
				for _, pred := range v.Predicates {
					if xpath3.PredicateIsNonMotionless(pred) {
						nonMotionless = true
						return false
					}
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
func checkFocusChangingCurrentGroup(body []Instruction) error {
	for _, bi := range body {
		if err := checkFocusChangingCurrentGroupInst(bi); err != nil {
			return err
		}
	}
	return nil
}

func checkFocusChangingCurrentGroupInst(inst Instruction) error {
	switch v := inst.(type) {
	case *CopyInst:
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
func bodyUsesCurrentGroup(body []Instruction) bool {
	for _, bi := range body {
		for _, expr := range getInstructionExprs(bi) {
			if expr == nil {
				continue
			}
			if xpath3.ExprUsesFunction(expr, "current-group") {
				return true
			}
		}
		for _, children := range getChildInstructions(bi) {
			if bodyUsesCurrentGroup(children) {
				return true
			}
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
			if v.Prefix == "" && v.Name == "current-group" {
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
	return xpath3.ExprUsesFunction(expr, "current-group")
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
func checkMapStreamable(ss *Stylesheet, inst Instruction) error {
	mapInst, ok := inst.(*MapInst)
	if !ok {
		return nil
	}

	consumingEntries := 0
	hasNonEntryChildren := false
	for _, child := range mapInst.Body {
		if me, ok := child.(*MapEntryInst); ok {
			// Check if key and select/body EACH consume the stream.
			// Use ExprHasDownwardStep which counts downward steps even
			// inside grounding functions — each still consumes the stream.
			keyConsuming := me.Key != nil && xpath3.ExprHasDownwardStep(me.Key)
			selectConsuming := me.Select != nil && xpath3.ExprHasDownwardStep(me.Select)
			bodyConsuming := false
			for _, meChild := range me.Body {
				for _, expr := range getInstructionExprs(meChild) {
					if expr != nil && xpath3.ExprHasDownwardStep(expr) {
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
	// Collect variables whose select expressions reference "." (context item),
	// either directly or transitively through other variables.
	taintedVars := collectTaintedVars(body)

	return bodyAccumulatesStreamedNodesInner(body, elemParams, taintedVars)
}

// collectTaintedVars finds all variables in the instruction tree whose select
// expressions reference the context item "." either directly or transitively
// through other variables that reference ".".
func collectTaintedVars(body []Instruction) map[string]bool {
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
func collectVarExprs(body []Instruction, varExprs map[string]xpath3.Expr) {
	for _, inst := range body {
		if vi, ok := inst.(*VariableInst); ok && vi.Select != nil {
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
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
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

func bodyAccumulatesStreamedNodesInner(body []Instruction, elemParams map[string]bool, taintedVars map[string]bool) bool {
	for _, inst := range body {
		if ni, ok := inst.(*NextIterationInst); ok {
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
func countDownwardInInstructions(ss *Stylesheet, instructions []Instruction) int {
	total := 0
	for _, inst := range instructions {
		// When a for-each-group select ends with a grounding function
		// (e.g. copy-of), the group-by/group-adjacent/sort expressions
		// operate on grounded (in-memory) items, not the streaming source.
		// Only count the select expression itself as a streaming downward
		// selection; skip the grouping key and sort expressions.
		if fg, ok := inst.(*ForEachGroupInst); ok {
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

		// For fork instructions, each branch gets its own copy of the
		// stream, so take the max across branches (like choose).
		if fork, ok := inst.(*ForkInst); ok {
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
		if lre, ok := inst.(*LiteralResultElement); ok {
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
	return ok && fc.Prefix == "" && fc.Name == "current-group"
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
	xpath3.WalkExpr(expr, func(e xpath3.Expr) bool {
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
		xpath3.WalkExpr(e, func(child xpath3.Expr) bool {
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

	// Check that additional params don't receive streaming nodes.
	if err := checkAdditionalParamsNotStreaming(fn, "absorbing"); err != nil {
		return err
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

	// Check that additional params don't receive streaming nodes.
	if err := checkAdditionalParamsNotStreaming(fn, "inspection"); err != nil {
		return err
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

	// Check that additional params don't receive streaming nodes.
	if err := checkAdditionalParamsNotStreaming(fn, "filter"); err != nil {
		return err
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

// checkAdditionalParamsNotStreaming checks that additional parameters (2nd, 3rd, ...)
// of a streamable function have atomic type constraints, so they cannot receive
// streaming nodes from the caller.
func checkAdditionalParamsNotStreaming(fn *XSLFunction, category string) error {
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
		collectAllInstructionExprs(fn.Body, func(expr *xpath3.Expression) {
			refs += countParamDownwardRefs(expr, p.Name)
		})
		if refs > 1 {
			return true
		}
	}

	return false
}

// collectAllInstructionExprs recursively walks an instruction tree and calls
// fn for every XPath expression found, including expressions in nested bodies.
func collectAllInstructionExprs(insts []Instruction, fn func(*xpath3.Expression)) {
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
func getInstructionBodies(inst Instruction) [][]Instruction {
	switch v := inst.(type) {
	case *CopyInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *IfInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *ChooseInst:
		var bodies [][]Instruction
		for _, w := range v.When {
			if w.Body != nil {
				bodies = append(bodies, w.Body)
			}
		}
		if v.Otherwise != nil {
			bodies = append(bodies, v.Otherwise)
		}
		return bodies
	case *ForEachInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *ElementInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *LiteralResultElement:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *VariableInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *SequenceInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *ValueOfInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *AttributeInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *MessageInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *ResultDocumentInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *MapInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *MapEntryInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	case *DocumentInst:
		if v.Body != nil {
			return [][]Instruction{v.Body}
		}
	}
	return nil
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

// functionHasParamInSimpleMap returns true if the streaming parameter (first param)
// is referenced on the right-hand side of a simple mapping expression (! operator),
// which causes the parameter to be consumed multiple times.
// E.g., (1 to 5) ! $n — $n is accessed for each item in the range.
func functionHasParamInSimpleMap(fn *XSLFunction) bool {
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
// element nodes (NameTest or generic node() test, but not text()/comment()/etc.).
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
