package xslt3

import (
	"context"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

// applyTemplates matches and executes templates for a node.
func (ec *execContext) applyTemplates(ctx context.Context, node helium.Node, mode string, paramValues ...map[string]xpath3.Sequence) error {
	if normalizeNode(node) == nil {
		return nil
	}
	// Strip whitespace-only text nodes per xsl:strip-space
	if ec.shouldStripWhitespace(node) {
		return nil
	}

	ec.depth++
	if ec.depth > maxRecursionDepth {
		ec.depth--
		return dynamicError(errCodeXTDE0820, "recursion depth exceeded")
	}
	defer func() { ec.depth-- }()

	switch mode {
	case modeCurrent:
		mode = ec.currentMode
	case modeUnnamed:
		mode = "" // the unnamed mode, regardless of default-mode
	case modeDefault, "":
		mode = ec.stylesheet.defaultMode
	}
	// Normalize #unnamed to "" for consistent template lookup
	if mode == modeUnnamed {
		mode = ""
	}

	// Collect param values if any
	var pv map[string]xpath3.Sequence
	if len(paramValues) > 0 {
		pv = paramValues[0]
	}

	// XTTE3100: typed="yes" mode rejects untyped nodes
	// XTTE3110: typed="no" mode rejects typed nodes
	if md := ec.lookupModeDef(mode); md != nil {
		switch md.Typed {
		case lexicon.ValueYes, "true", "1", validationStrict, validationLax:
			if !ec.nodeIsTyped(node) {
				return dynamicError(errCodeXTTE3100,
					"mode %q has typed=%q but node is untyped", mode, md.Typed)
			}
		case lexicon.ValueNo, "false", "0":
			if ec.nodeIsTyped(node) {
				return dynamicError(errCodeXTTE3110,
					"mode %q has typed=%q but node is typed", mode, md.Typed)
			}
		}
	}

	// Find best matching template
	tmpl, err := ec.findBestTemplate(node, mode)
	if err != nil {
		return err
	}
	if tmpl != nil {
		return ec.executeTemplate(ctx, tmpl, node, mode, pv)
	}

	// Use built-in template rules
	return ec.applyBuiltinRules(ctx, node, mode, paramValues...)
}

// lookupModeDef returns the modeDef for the given mode name.
// Handles the mismatch where templates use "" for the unnamed mode
// but mode definitions are stored under "#default".
func (ec *execContext) lookupModeDef(mode string) *modeDef {
	modeDefs := ec.effectiveModeDefs()
	if md := modeDefs[mode]; md != nil {
		return md
	}
	if mode == "" {
		return modeDefs[modeDefault]
	}
	return nil
}

// nodeIsTyped returns true if the node has a schema type annotation,
// or if the node belongs to a document that was schema-validated (even if
// this specific node has a complex type without an individual annotation).
func (ec *execContext) nodeIsTyped(node helium.Node) bool {
	if ec.typeAnnotations == nil {
		return false
	}
	if _, ok := ec.typeAnnotations[node]; ok {
		return true
	}
	// Check if the node belongs to a validated document. This covers
	// complex-type elements that don't have individual annotations but
	// are still considered typed per the XDM data model.
	if len(ec.validatedDocs) > 0 {
		for cur := node; cur != nil; cur = cur.Parent() {
			if doc, ok := cur.(*helium.Document); ok {
				if _, validated := ec.validatedDocs[doc]; validated {
					return true
				}
				break
			}
		}
	}
	return false
}

// findBestTemplate finds the highest-priority matching template for a node.
// Returns XTRE0540 when on-multiple-match="fail" and two templates match
// with equal priority and import precedence.
func (ec *execContext) findBestTemplate(node helium.Node, mode string) (*template, error) {
	// Set currentNode to the candidate so current() works in pattern predicates
	savedCurrent := ec.currentNode
	ec.currentNode = node
	defer func() { ec.currentNode = savedCurrent }()

	best := ec.findFirstMatch(ec.stylesheet.modeTemplates[mode], node)

	// Also check #all mode templates that might not be registered in this mode
	if best == nil && mode != modeAll {
		best = ec.findFirstMatch(ec.stylesheet.modeTemplates[modeAll], node)
	}

	if best == nil {
		return nil, nil
	}

	// Check on-multiple-match="fail": look for a second matching template
	// with the same priority and import precedence.
	if ec.onMultipleMatchFail(mode) {
		if ec.hasConflictingMatch(node, mode, best) {
			return nil, dynamicError(errCodeXTDE0540,
				"ambiguous rule match for node %v in mode %q (on-multiple-match=fail)",
				node, mode)
		}
	}

	return best, nil
}

// findFirstMatch returns the first template in the list that matches the node.
func (ec *execContext) findFirstMatch(templates []*template, node helium.Node) *template {
	for _, tmpl := range templates {
		if tmpl.Match != nil && tmpl.Match.matchPattern(ec, node) {
			return tmpl
		}
	}
	return nil
}

// onMultipleMatchFail returns true when the effective on-multiple-match for
// the given mode is "fail".
func (ec *execContext) onMultipleMatchFail(mode string) bool {
	// Check mode definition in the stylesheet first
	modeDefs := ec.effectiveModeDefs()
	if md := modeDefs[mode]; md != nil && md.OnMultipleMatch != "" {
		return md.OnMultipleMatch == onNoMatchFail
	}
	// The unnamed default mode is stored as "#default" but referenced as ""
	if mode == "" {
		if md := modeDefs[modeDefault]; md != nil && md.OnMultipleMatch != "" {
			return md.OnMultipleMatch == onNoMatchFail
		}
	}
	// Fall back to transform-time override (e.g. from W3C test dependency)
	if ec.transformConfig != nil && ec.transformConfig.onMultipleMatch != "" {
		return ec.transformConfig.onMultipleMatch == onNoMatchFail
	}
	return false
}

// hasConflictingMatch checks whether there is another template (besides best)
// that matches the same node with equal import precedence and priority.
func (ec *execContext) hasConflictingMatch(node helium.Node, mode string, best *template) bool {
	check := func(templates []*template) bool {
		for _, tmpl := range templates {
			if tmpl == best {
				continue
			}
			// Templates are sorted: once we see lower precedence/priority we can stop
			if tmpl.ImportPrec < best.ImportPrec {
				return false
			}
			if tmpl.ImportPrec == best.ImportPrec && tmpl.Priority < best.Priority {
				return false
			}
			// Split union pattern branches share the same Body slice; they
			// originate from the same template rule and should not be
			// treated as conflicting (spec bug 30402).
			if len(tmpl.Body) > 0 && len(best.Body) > 0 && &tmpl.Body[0] == &best.Body[0] {
				continue
			}
			if tmpl.Match != nil && tmpl.Match.matchPattern(ec, node) {
				return true
			}
		}
		return false
	}

	if check(ec.stylesheet.modeTemplates[mode]) {
		return true
	}
	if mode != modeAll {
		return check(ec.stylesheet.modeTemplates[modeAll])
	}
	return false
}

// findAtomicTemplate finds a template matching an atomic value.
// XSLT 3.0 patterns like ".[. instance of xs:integer]" can match atomic items.
func (ec *execContext) findAtomicTemplate(item xpath3.Item, mode string) (*template, error) {
	var best *template
	templates := ec.stylesheet.modeTemplates[mode]
	for _, tmpl := range templates {
		if tmpl.Match != nil && ec.matchAtomicPattern(tmpl.Match, item) {
			best = tmpl
			break
		}
	}
	if best == nil && mode != modeAll {
		for _, tmpl := range ec.stylesheet.modeTemplates[modeAll] {
			if tmpl.Match != nil && ec.matchAtomicPattern(tmpl.Match, item) {
				best = tmpl
				break
			}
		}
	}
	if best == nil {
		return nil, nil
	}
	// Check on-multiple-match="fail": look for conflicting atomic template match.
	if ec.onMultipleMatchFail(mode) {
		if ec.hasConflictingAtomicMatch(item, mode, best) {
			return nil, dynamicError(errCodeXTDE0540,
				"ambiguous rule match for atomic item in mode %q (on-multiple-match=fail)", mode)
		}
	}
	return best, nil
}

// hasConflictingAtomicMatch checks whether there is another template (besides best)
// that matches the same atomic item with equal import precedence and priority.
func (ec *execContext) hasConflictingAtomicMatch(item xpath3.Item, mode string, best *template) bool {
	check := func(templates []*template) bool {
		for _, tmpl := range templates {
			if tmpl == best {
				continue
			}
			if tmpl.ImportPrec < best.ImportPrec {
				return false
			}
			if tmpl.ImportPrec == best.ImportPrec && tmpl.Priority < best.Priority {
				return false
			}
			if len(tmpl.Body) > 0 && len(best.Body) > 0 && &tmpl.Body[0] == &best.Body[0] {
				continue
			}
			if tmpl.Match != nil && ec.matchAtomicPattern(tmpl.Match, item) {
				return true
			}
		}
		return false
	}
	if check(ec.stylesheet.modeTemplates[mode]) {
		return true
	}
	if mode != modeAll {
		return check(ec.stylesheet.modeTemplates[modeAll])
	}
	return false
}

// matchAtomicPattern checks if an atomic item matches a pattern.
func (ec *execContext) matchAtomicPattern(p *pattern, item xpath3.Item) bool {
	for _, alt := range p.Alternatives {
		// variable reference patterns (e.g., match="$var") only match nodes,
		// never atomic items per XSLT 3.0 §5.5.3.
		if _, isVar := alt.expr.(xpath3.VariableExpr); isVar {
			continue
		}
		compiled := alt.compiled
		if compiled == nil {
			var compErr error
			compiled, compErr = xpath3.NewCompiler().CompileExpr(alt.expr)
			if compErr != nil {
				continue
			}
		}
		// Evaluate the pattern as a boolean predicate with the item as context
		result, err := ec.xpathEvaluator().ContextItem(item).Evaluate(ec.xpathContext(), compiled, nil)
		if err != nil {
			continue
		}
		// The pattern ".[. instance of xs:integer]" evaluates to the item
		// itself when matched, or empty when not. Check non-empty.
		if result.Sequence() != nil && sequence.Len(result.Sequence()) > 0 {
			return true
		}
	}
	return false
}

// executeAtomicTemplate executes a template with an atomic item as context.
func (ec *execContext) executeAtomicTemplate(ctx context.Context, tmpl *template, item xpath3.Item, mode string) error {
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	savedMode := ec.currentMode
	savedItem := ec.contextItem
	savedTemplate := ec.currentTemplate
	ec.contextItem = item
	ec.contextNode = nil
	ec.currentNode = nil
	ec.currentMode = mode
	ec.setCurrentTemplate(tmpl)
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.currentMode = savedMode
		ec.contextItem = savedItem
		ec.setCurrentTemplate(savedTemplate)
	}()

	ec.pushVarScope()
	defer ec.popVarScope()

	// Set template parameters with defaults (similar to executeTemplate).
	for _, p := range tmpl.Params {
		var val xpath3.Sequence

		if p.Tunnel {
			if ec.tunnelParams != nil {
				if v, ok := ec.tunnelParams[p.Name]; ok {
					val = v
					ec.setVar(p.Name, val)
					continue
				}
			}
		}

		if p.Required {
			return dynamicError(errCodeXTDE0700, "required parameter $%s was not supplied", p.Name)
		}

		if p.Select != nil {
			result, err := ec.xpathEvaluator().ContextItem(item).Evaluate(ec.xpathContext(), p.Select, nil)
			if err != nil {
				return err
			}
			val = result.Sequence()
		} else if len(p.Body) > 0 {
			ec.temporaryOutputDepth++
			var err error
			if p.As != "" {
				val, err = ec.evaluateBodyAsSequence(ctx, p.Body)
			} else {
				val, err = ec.evaluateBody(ctx, p.Body)
			}
			ec.temporaryOutputDepth--
			if err != nil {
				return err
			}
		} else {
			val = xpath3.EmptySequence()
		}

		if p.As != "" && val != nil && sequence.Len(val) > 0 {
			st := parseSequenceType(p.As)
			checked, err := checkSequenceType(val, st, errCodeXTTE0570, "param $"+p.Name, ec)
			if err != nil {
				return err
			}
			val = checked
		}

		ec.setVar(p.Name, val)
	}

	if tmpl.As != "" {
		return ec.executeTemplateBodyWithAs(ctx, tmpl)
	}
	return ec.executeSequenceConstructor(ctx, tmpl.Body)
}

// executeTemplate executes a template with the given node as context.
const maxRecursionDepth = 2000

func (ec *execContext) executeTemplate(ctx context.Context, tmpl *template, node helium.Node, mode string, paramOverrides ...map[string]xpath3.Sequence) error {
	// XTDE0160: backwards compatibility mode is not supported.
	// Only XSLT 1.0 triggers backwards compatible behavior. Other versions
	// (like 1.5) are forward-compatible and don't require BC support.
	if strings.TrimSpace(tmpl.Version) == "1.0" {
		return dynamicError(errCodeXTDE0160,
			"backwards compatibility mode (version 1.0) is not supported")
	}

	// If the template belongs to a package, switch function scope
	// so package-private functions are visible.
	savedFnsNS := ec.cachedFnsNS
	savedPackage := ec.currentPackage
	if tmpl.OwnerPackage != nil && tmpl.OwnerPackage != ec.currentPackage {
		ec.cachedFnsNS = nil // force rebuild with new package scope
		ec.currentPackage = tmpl.OwnerPackage
	}
	defer func() {
		ec.cachedFnsNS = savedFnsNS
		ec.currentPackage = savedPackage
	}()

	// Save and restore context
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedItem := ec.contextItem
	savedMode := ec.currentMode
	savedPos := ec.position
	savedSize := ec.size
	savedTemplate := ec.currentTemplate
	savedTunnel := ec.tunnelParams
	savedXPathDefaultNS := ec.xpathDefaultNS
	savedHasXPathDefaultNS := ec.hasXPathDefaultNS
	savedGroup := ec.currentGroup
	savedGroupKey := ec.currentGroupKey
	savedDefaultCollation := ec.defaultCollation
	ec.currentNode = node
	ec.contextNode = node
	ec.currentMode = mode
	ec.setCurrentTemplate(tmpl)
	ec.xpathDefaultNS = tmpl.XPathDefaultNS
	ec.hasXPathDefaultNS = tmpl.XPathDefaultNS != ""
	if tmpl.DefaultCollation != "" {
		ec.defaultCollation = tmpl.DefaultCollation
	}
	// When entering a template with a node context, clear the atomic context
	// item so that next-match and other instructions use the node context
	// rather than a stale atomic value (e.g., from xsl:analyze-string).
	if node != nil {
		ec.contextItem = nil
	}
	// xsl:context-item use="absent": make the context item absent within the template body
	if tmpl.ContextItemUse == ctxItemAbsent {
		ec.contextNode = nil
		ec.contextItem = nil
	}
	// XTTE0590: check context-item type constraint
	if err := ec.checkContextItemType(tmpl); err != nil {
		return err
	}
	// XSLT spec: current-group() and current-grouping-key() are only
	// available within the body of xsl:for-each-group, not in templates
	// called from it.
	ec.currentGroup = nil
	ec.currentGroupKey = nil
	// Track overriding template for xsl:original support
	savedOverridingTemplate := ec.overridingTemplate
	if tmpl.OriginalTemplate != nil {
		ec.overridingTemplate = tmpl
	}
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.contextItem = savedItem
		ec.currentMode = savedMode
		ec.setCurrentTemplate(savedTemplate)
		ec.overridingTemplate = savedOverridingTemplate
		ec.xpathDefaultNS = savedXPathDefaultNS
		ec.hasXPathDefaultNS = savedHasXPathDefaultNS
		ec.defaultCollation = savedDefaultCollation
		ec.position = savedPos
		ec.size = savedSize
		ec.tunnelParams = savedTunnel
		ec.currentGroup = savedGroup
		ec.currentGroupKey = savedGroupKey
	}()

	ec.pushVarScope()
	defer ec.popVarScope()

	// Collect param overrides
	var po map[string]xpath3.Sequence
	if len(paramOverrides) > 0 {
		po = paramOverrides[0]
	}

	// Set param values: use with-param overrides when available, else defaults.
	// Tunnel params receive from ec.tunnelParams, not from regular param overrides.
	for _, p := range tmpl.Params {
		var val xpath3.Sequence
		fromCaller := false

		if p.Tunnel {
			// Tunnel param: receive from tunnel context
			if ec.tunnelParams != nil {
				if v, ok := ec.tunnelParams[p.Name]; ok {
					val = v
					fromCaller = true
				}
			}
		} else if po != nil {
			if v, ok := po[p.Name]; ok {
				val = v
				fromCaller = true
			}
		}

		if !fromCaller {
			// XTDE0700: required parameter not supplied
			if p.Required {
				kind := ""
				if p.Tunnel {
					kind = "tunnel "
				}
				return dynamicError(errCodeXTDE0700, "required %sparameter $%s was not supplied", kind, p.Name)
			}
			// XTDE0700: optional parameter with as type that doesn't allow
			// empty sequence and no default value is effectively required.
			if p.Select == nil && len(p.Body) == 0 && p.As != "" {
				if !allowsEmptySequence(p.As) {
					return dynamicError(errCodeXTDE0700,
						"required parameter $%s (type %s does not allow empty sequence) was not supplied",
						p.Name, p.As)
				}
			}
			// Use default value
			if p.Select != nil {
				result, err := ec.evalXPath(p.Select, node)
				if err != nil {
					return err
				}
				val = result.Sequence()
			} else if len(p.Body) > 0 {
				ec.temporaryOutputDepth++
				var err error
				if p.As != "" {
					val, err = ec.evaluateBodyAsSequence(ctx, p.Body)
				} else {
					val, err = ec.evaluateBody(ctx, p.Body)
				}
				ec.temporaryOutputDepth--
				if err != nil {
					return err
				}
			} else {
				val = xpath3.EmptySequence()
			}
		}

		// Type check against the declared as type
		if p.As != "" && val != nil && sequence.Len(val) > 0 {
			st := parseSequenceType(p.As)
			errCode := errCodeXTTE0570
			if fromCaller {
				errCode = errCodeXTTE0590
			}
			checked, err := checkSequenceType(val, st, errCode, "param $"+p.Name, ec)
			if err != nil {
				return err
			}
			val = checked
		}

		ec.setVar(p.Name, val)
	}

	// Execute template body
	if tmpl.As != "" {
		// template has return type constraint — capture output and validate
		return ec.executeTemplateBodyWithAs(ctx, tmpl)
	}

	return ec.executeSequenceConstructor(ctx, tmpl.Body)
}

// executeTemplateBodyWithAs runs the template body in capture mode,
// checks the result against the declared as type, then writes the
// validated items to the real output.
func (ec *execContext) executeTemplateBodyWithAs(ctx context.Context, tmpl *template) error {
	seq, err := ec.evaluateBodyAsSequence(ctx, tmpl.Body)
	if err != nil {
		return err
	}

	// Strip DOE marker PIs from the captured sequence before type checking.
	// DOE PIs are internal serialization markers that should be transparent
	// to the XSLT type system. Track which text node indices had DOE markers
	// so we can re-insert them when writing to the result tree.
	stripped, doeTextIndices := stripDOEMarkers(seq)

	st := parseSequenceType(tmpl.As)
	checked, err := checkSequenceType(stripped, st, errCodeXTTE0505, "template", ec)
	if err != nil {
		return err
	}

	// Capture the raw typed sequence when writing to the primary output
	// (outputStack depth == 1). This preserves atomic type information
	// that would otherwise be lost during serialization to text nodes.
	if len(ec.outputStack) == 1 {
		ec.rawResultSequence = checked
	}

	// Write the validated items to the real output.
	// When the output is in capture mode (e.g., inside key body or
	// variable evaluation), preserve atomic values directly so that
	// their XSD type is retained for typed comparisons.
	out := ec.currentOutput()
	for i := range sequence.Len(checked) {
		item := checked.Get(i)
		switch v := item.(type) {
		case xpath3.NodeItem:
			out.prevWasAtomic = false
			// Re-insert DOE markers when writing directly to the result tree
			// (not inside a variable/capture context). DOE is ignored when
			// writing to variables, attributes, or comments (XSLT 3.0 §20.1).
			isDOE := doeTextIndices[i]
			// Re-insert DOE markers only when writing directly to the
			// final result tree. DOE is ignored in temporary output
			// states (variables, functions) and capture contexts
			// (comments, attributes, PIs) per XSLT 3.0 §20.1.
			emitDOE := isDOE && ec.temporaryOutputDepth == 0 && !out.captureItems && !out.sequenceMode
			if emitDOE {
				pi := ec.resultDoc.CreatePI("disable-output-escaping", "")
				if err := ec.addNode(pi); err != nil {
					return err
				}
			}
			// Attribute nodes must be set as attributes on the current
			// element, not added as child nodes.
			if v.Node.Type() == helium.AttributeNode {
				attr, ok := v.Node.(*helium.Attribute)
				if ok {
					if elem, ok := out.current.(*helium.Element); ok {
						// XTRE0540: cannot add attribute after child content
						if elem.FirstChild() != nil {
							return dynamicError(errCodeXTRE0540,
								"cannot add attribute to element after children have been added")
						}
						if attr.URI() != "" {
							ns, _ := out.doc.CreateNamespace(attr.Prefix(), attr.URI())
							if err := elem.SetLiteralAttributeNS(attr.LocalName(), string(attr.Content()), ns); err != nil {
								return err
							}
						} else {
							if _, err := elem.SetAttribute(attr.LocalName(), string(attr.Content())); err != nil {
								return err
							}
						}
					}
				}
			} else if v.Node.Type() == helium.DocumentNode && !out.captureItems && !out.sequenceMode {
				// Document nodes are unwrapped when writing to a real
				// DOM output: emit children since documents can't be
				// children of elements. In capture/sequence mode, keep
				// the document node intact (pattern matching may need it).
				doc := v.Node.(*helium.Document) //nolint:forcetypeassert
				for dc := range helium.Children(doc) {
					copied, copyErr := helium.CopyNode(dc, ec.resultDoc)
					if copyErr != nil {
						return copyErr
					}
					if err := ec.addNode(copied); err != nil {
						return err
					}
				}
			} else if err := ec.addNode(v.Node); err != nil {
				return err
			}
			if emitDOE {
				piEnd := ec.resultDoc.CreatePI("enable-output-escaping", "")
				if err := ec.addNode(piEnd); err != nil {
					return err
				}
			}
		case xpath3.AtomicValue:
			if out.captureItems {
				out.pendingItems = append(out.pendingItems, v)
				out.noteOutput()
			} else {
				s, err := xpath3.AtomicToString(v)
				if err != nil {
					return err
				}
				// Insert space separator between consecutive atomic values
				// (XSLT 3.0 §11.3: adjacent atomic values are separated by spaces)
				if out.prevWasAtomic {
					sep := ec.resultDoc.CreateText([]byte(" "))
					if err := ec.addNode(sep); err != nil {
						return err
					}
				}
				text := ec.resultDoc.CreateText([]byte(s))
				if err := ec.addNode(text); err != nil {
					return err
				}
				out.prevWasAtomic = true
			}
		default:
			// Function items, maps, and arrays: capture when in
			// capture/sequence mode, otherwise error (they cannot
			// appear in the result tree).
			if out.captureItems || out.sequenceMode || ec.isItemOutputMethod() {
				out.pendingItems = append(out.pendingItems, item)
				out.noteOutput()
			} else {
				return dynamicError(errCodeXTDE0450,
					"cannot add a %T to the result tree", item)
			}
		}
	}
	return nil
}

// stripDOEMarkers removes disable-output-escaping / enable-output-escaping
// marker PIs from a sequence, returning the cleaned sequence and a map of
// indices (in the cleaned sequence) that had DOE markers around them.
func stripDOEMarkers(seq xpath3.Sequence) (xpath3.Sequence, map[int]bool) {
	doeIndices := map[int]bool{}
	result := make(xpath3.ItemSlice, 0, sequence.Len(seq))
	doeActive := false
	for item := range sequence.Items(seq) {
		if ni, ok := item.(xpath3.NodeItem); ok && ni.Node.Type() == helium.ProcessingInstructionNode {
			piName := ni.Node.Name()
			if piName == "disable-output-escaping" {
				doeActive = true
				continue
			}
			if piName == "enable-output-escaping" {
				doeActive = false
				continue
			}
		}
		if doeActive {
			doeIndices[len(result)] = true
		}
		result = append(result, item)
	}
	return result, doeIndices
}
