package xslt3

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) execApplyTemplates(ctx context.Context, inst *ApplyTemplatesInst) error {
	var nodes []helium.Node

	var atomicItems xpath3.Sequence // XSLT 3.0: atomic values from select
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		seq := flattenArraysInSequence(result.Sequence())
		ns, ok := xpath3.NodesFrom(seq)
		if ok {
			nodes = ns
		} else {
			// XSLT 3.0: separate nodes from atomic values
			for _, item := range seq {
				if ni, ok := item.(xpath3.NodeItem); ok {
					nodes = append(nodes, ni.Node)
				} else {
					atomicItems = append(atomicItems, item)
				}
			}
		}
	} else {
		// Default select is child::node() which requires a node context item.
		// If the context item is an atomic value (contextNode is nil), raise XTTE0510.
		if normalizeNode(ec.contextNode) == nil {
			return dynamicError(errCodeXTTE0510, "apply-templates with default select requires a node context item")
		}
		nodes = selectDefaultNodes(ec.contextNode)
	}

	// Apply sort keys if present
	if len(inst.Sort) > 0 {
		var err error
		nodes, err = sortNodes(ctx, ec, nodes, inst.Sort)
		if err != nil {
			return err
		}
	}

	mode := inst.Mode
	if mode == "#current" {
		mode = ec.currentMode
	}
	// When mode is absent (empty), use the stylesheet's default-mode
	// (not the current mode — #current must be explicit)

	// Process with-param values, separating tunnel from regular params
	var paramValues map[string]xpath3.Sequence
	var newTunnelParams map[string]xpath3.Sequence
	if len(inst.Params) > 0 {
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			if wp.Tunnel {
				if newTunnelParams == nil {
					newTunnelParams = make(map[string]xpath3.Sequence)
				}
				newTunnelParams[wp.Name] = val
			} else {
				if paramValues == nil {
					paramValues = make(map[string]xpath3.Sequence)
				}
				if _, dup := paramValues[wp.Name]; dup {
					return dynamicError(errCodeXTDE0410, "duplicate parameter %q in xsl:apply-templates", wp.Name)
				}
				paramValues[wp.Name] = val
			}
		}
	}

	// Merge new tunnel params with existing tunnel params (new values override)
	savedTunnel := ec.tunnelParams
	if newTunnelParams != nil {
		merged := make(map[string]xpath3.Sequence)
		for k, v := range ec.tunnelParams {
			merged[k] = v
		}
		for k, v := range newTunnelParams {
			merged[k] = v
		}
		ec.tunnelParams = merged
	}

	// Filter whitespace-only text nodes per xsl:strip-space before
	// setting position/size, so position()/last() reflect the filtered list.
	filtered := nodes[:0]
	for _, node := range nodes {
		if !ec.shouldStripWhitespace(node) {
			filtered = append(filtered, node)
		}
	}
	nodes = filtered

	savedPos := ec.position
	savedSize := ec.size
	savedGroupKey := ec.currentGroupKey
	savedGroup := ec.currentGroup
	savedInGroupCtx := ec.inGroupContext
	savedGroupHasKey := ec.groupHasKey
	savedInMerge := ec.inMergeAction
	ec.size = len(nodes)
	// Per XSLT 3.0: current-grouping-key() and current-group() are only
	// available in the body of xsl:for-each-group itself, not in templates
	// invoked by apply-templates within that body.
	ec.currentGroupKey = nil
	ec.currentGroup = nil
	ec.inGroupContext = false
	ec.groupHasKey = false
	ec.inMergeAction = false // XTDE3480: merge context not available across template boundaries
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
		ec.tunnelParams = savedTunnel
		ec.currentGroupKey = savedGroupKey
		ec.currentGroup = savedGroup
		ec.inGroupContext = savedInGroupCtx
		ec.groupHasKey = savedGroupHasKey
		ec.inMergeAction = savedInMerge
	}()

	for i, node := range nodes {
		ec.position = i + 1

		if err := ec.applyTemplates(ctx, node, mode, paramValues); err != nil {
			return err
		}
	}

	// XSLT 3.0: sort atomic items if sort keys are present.
	if len(inst.Sort) > 0 && len(atomicItems) > 0 {
		sorted, err := sortItems(ctx, ec, xpath3.Sequence(atomicItems), inst.Sort)
		if err != nil {
			return err
		}
		atomicItems = []xpath3.Item(sorted)
	}

	// XSLT 3.0: process atomic values — try template matching first,
	// then fall back to built-in text output.
	// Set position/size so position()/last() work correctly inside templates.
	ec.size = len(atomicItems)
	for i, item := range atomicItems {
		ec.position = i + 1
		tmpl, err := ec.findAtomicTemplate(item, mode)
		if err != nil {
			return err
		}
		if tmpl != nil {
			if err := ec.executeAtomicTemplate(ctx, tmpl, item, mode); err != nil {
				return err
			}
			continue
		}
		// Check mode's on-no-match: for "deep-skip" and "shallow-skip",
		// unmatched atomic items are silently skipped.
		modeKey := mode
		if modeKey == "" || modeKey == "#unnamed" {
			modeKey = "#default"
		}
		modeDef := ec.stylesheet.modeDefs[modeKey]
		if modeDef != nil && (modeDef.OnNoMatch == "deep-skip" || modeDef.OnNoMatch == "shallow-skip") {
			continue
		}
		av, err := xpath3.AtomizeItem(item)
		if err != nil {
			continue
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			continue
		}
		text, err := ec.resultDoc.CreateText([]byte(s))
		if err != nil {
			return err
		}
		if err := ec.addNode(text); err != nil {
			return err
		}
	}

	return nil
}

func (ec *execContext) evaluateWithParam(ctx context.Context, wp *WithParam) (xpath3.Sequence, error) {
	var val xpath3.Sequence
	if wp.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := wp.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return nil, err
		}
		val = result.Sequence()
	} else if len(wp.Body) > 0 {
		ec.temporaryOutputDepth++
		var err error
		if wp.As != "" {
			// With as attribute: evaluate as sequence constructor,
			// keeping each node as a separate item
			val, err = ec.evaluateBodyAsSequence(ctx, wp.Body)
		} else {
			// No as: wrap in document node (temporary tree)
			val, err = ec.evaluateBodyAsDocument(ctx, wp.Body)
		}
		ec.temporaryOutputDepth--
		if err != nil {
			return nil, err
		}
	} else {
		if wp.As != "" {
			val = xpath3.EmptySequence()
		} else {
			val = xpath3.SingleAtomic(xpath3.AtomicValue{
				TypeName: xpath3.TypeString,
				Value:    "",
			})
		}
	}

	// Type check against the declared as type
	if wp.As != "" {
		st := parseSequenceType(wp.As)
		checked, err := checkSequenceType(val, st, errCodeXTTE0570, "with-param $"+wp.Name, ec)
		if err != nil {
			return nil, err
		}
		val = checked
	}

	return val, nil
}

// resolveNamedTemplate looks up a named template by searching:
// 1. The main stylesheet's namedTemplates map
// 2. The current package context (if set) - allows private templates
//    from the package to be called by other templates in the same package
func (ec *execContext) resolveNamedTemplate(name string) (*Template, bool) {
	// Handle xsl:original — resolve to the original overridden template
	if name == "{"+NSXSLT+"}original" {
		if ec.overridingTemplate != nil && ec.overridingTemplate.OriginalTemplate != nil {
			return ec.overridingTemplate.OriginalTemplate, true
		}
		return nil, false
	}
	if tmpl, ok := ec.stylesheet.namedTemplates[name]; ok {
		return tmpl, true
	}
	if ec.currentPackage != nil {
		if tmpl, ok := ec.currentPackage.namedTemplates[name]; ok {
			return tmpl, true
		}
	}
	return nil, false
}


func (ec *execContext) execCallTemplate(ctx context.Context, inst *CallTemplateInst) error {
	ec.depth++
	if ec.depth > maxRecursionDepth {
		ec.depth--
		return dynamicError(errCodeXTDE0820, "recursion depth exceeded")
	}
	defer func() { ec.depth-- }()

	// XTDE3480/XTDE3510: clear merge-action context across template boundaries.
	savedInMerge := ec.inMergeAction
	ec.inMergeAction = false
	defer func() { ec.inMergeAction = savedInMerge }()

	tmpl, ok := ec.resolveNamedTemplate(inst.Name)
	if !ok {
		return dynamicError(errCodeXTDE0060, "named template %q not found", inst.Name)
	}

	// XTTE0590: check context-item type constraint
	if err := ec.checkContextItemType(tmpl); err != nil {
		return err
	}

	// Switch package function scope if the template belongs to a different package.
	savedFnsNS := ec.cachedFnsNS
	savedPackage := ec.currentPackage
	if tmpl.OwnerPackage != nil && tmpl.OwnerPackage != ec.currentPackage {
		ec.cachedFnsNS = nil
		ec.currentPackage = tmpl.OwnerPackage
	}
	// Do NOT set ec.currentTemplate here: xsl:call-template preserves the
	// current template rule for xsl:apply-imports/xsl:next-match.

	// Track overriding template for xsl:original support.
	savedOverridingTemplate := ec.overridingTemplate
	if tmpl.OriginalTemplate != nil {
		ec.overridingTemplate = tmpl
	}

	// xsl:context-item use="absent": make the context item absent within the
	// called template's body. This means xsl:next-match will fail with
	// XTDE0560 because there is no context node to match against.
	savedContextNode := ec.contextNode
	savedContextItem := ec.contextItem
	savedCurrentNode := ec.currentNode
	savedCurrentTemplate := ec.currentTemplate
	if tmpl.ContextItemUse == "absent" {
		ec.contextNode = nil
		ec.contextItem = nil
		ec.currentNode = nil
		ec.setCurrentTemplate(nil)
	}
	defer func() {
		ec.cachedFnsNS = savedFnsNS
		ec.currentPackage = savedPackage
		ec.overridingTemplate = savedOverridingTemplate
		if tmpl.ContextItemUse == "absent" {
			ec.contextNode = savedContextNode
			ec.contextItem = savedContextItem
			ec.currentNode = savedCurrentNode
			ec.setCurrentTemplate(savedCurrentTemplate)
		}
	}()

	ec.pushVarScope()
	defer ec.popVarScope()

	// Separate tunnel from regular with-param values.
	// call-template forwards tunnel params from the caller's context.
	paramOverrides := make(map[string]xpath3.Sequence)
	savedTunnel := ec.tunnelParams
	hasTunnelOverrides := false
	for _, wp := range inst.Params {
		val, err := ec.evaluateWithParam(ctx, wp)
		if err != nil {
			return err
		}
		if wp.Tunnel {
			if !hasTunnelOverrides {
				merged := make(map[string]xpath3.Sequence)
				for k, v := range ec.tunnelParams {
					merged[k] = v
				}
				ec.tunnelParams = merged
				hasTunnelOverrides = true
			}
			ec.tunnelParams[wp.Name] = val
		} else {
			if _, dup := paramOverrides[wp.Name]; dup {
				return dynamicError(errCodeXTDE0410, "duplicate parameter %q in xsl:call-template", wp.Name)
			}
			paramOverrides[wp.Name] = val
		}
	}
	defer func() { ec.tunnelParams = savedTunnel }()

	// Set template params with defaults, overrides, or tunnel values
	for _, p := range tmpl.Params {
		var (
			val        xpath3.Sequence
			fromCaller bool
		)

		if p.Tunnel {
			// Tunnel param: receive from tunnel context
			if ec.tunnelParams != nil {
				if v, ok := ec.tunnelParams[p.Name]; ok {
					val = v
					fromCaller = true
				}
			}
		} else if v, ok := paramOverrides[p.Name]; ok {
			val = v
			fromCaller = true
		}

		if !fromCaller && p.Select != nil {
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := p.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return err
			}
			val = result.Sequence()
		} else if !fromCaller && len(p.Body) > 0 {
			// Per XSLT spec: param body without select produces a
			// document node (temporary tree).
			ec.temporaryOutputDepth++
			var err error
			if p.As != "" {
				val, err = ec.evaluateBodyAsSequence(ctx, p.Body)
			} else {
				val, err = ec.evaluateBodyAsDocument(ctx, p.Body)
			}
			ec.temporaryOutputDepth--
			if err != nil {
				return err
			}
		} else if !fromCaller {
			val = xpath3.EmptySequence()
		}

		if p.As != "" && len(val) > 0 {
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

	if tmpl.As != "" {
		return ec.executeTemplateBodyWithAs(ctx, tmpl)
	}

	return ec.executeSequenceConstructor(ctx, tmpl.Body)
}

func (ec *execContext) execNextMatch(ctx context.Context, inst *NextMatchInst) error {
	// XTDE0560: xsl:next-match when the current template rule is absent.
	if ec.currentTemplate == nil || ec.currentTemplate.Match == nil {
		return dynamicError(errCodeXTDE0560, "xsl:next-match: no current template rule")
	}
	// xsl:next-match: find the next matching template after the current one
	node := ec.currentNode
	mode := ec.currentMode

	// Process with-param (tunnel and regular).
	// Copy tunnel params to avoid mutating the caller's map.
	var pv map[string]xpath3.Sequence
	savedTunnel := ec.tunnelParams
	if len(inst.Params) > 0 {
		hasTunnel := false
		for _, wp := range inst.Params {
			if wp.Tunnel {
				hasTunnel = true
				break
			}
		}
		if hasTunnel {
			newTunnel := make(map[string]xpath3.Sequence, len(ec.tunnelParams)+len(inst.Params))
			for k, v := range ec.tunnelParams {
				newTunnel[k] = v
			}
			ec.tunnelParams = newTunnel
		}
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			if wp.Tunnel {
				ec.tunnelParams[wp.Name] = val
			} else {
				if pv == nil {
					pv = make(map[string]xpath3.Sequence)
				}
				pv[wp.Name] = val
			}
		}
	}
	defer func() { ec.tunnelParams = savedTunnel }()

	// Handle atomic context items (e.g., xsl:next-match when processing integers)
	if ec.contextItem != nil {
		templates := ec.stylesheet.modeTemplates[mode]
		foundCurrent := false
		for _, tmpl := range templates {
			if tmpl == ec.currentTemplate {
				foundCurrent = true
				continue
			}
			if foundCurrent && tmpl.Match != nil && ec.matchAtomicPattern(tmpl.Match, ec.contextItem) {
				return ec.executeAtomicTemplate(ctx, tmpl, ec.contextItem, mode)
			}
		}
		// No next match for atomic items — output string value as built-in rule
		av, ok := ec.contextItem.(xpath3.AtomicValue)
		if !ok {
			return nil
		}
		s := fmt.Sprintf("%v", av.Value)
		text, err := ec.resultDoc.CreateText([]byte(s))
		if err != nil {
			return err
		}
		return ec.addNode(text)
	}

	templates := ec.stylesheet.modeTemplates[mode]
	foundCurrent := false
	failOnMultiple := ec.onMultipleMatchFail(mode)
	for i, tmpl := range templates {
		if tmpl == ec.currentTemplate {
			foundCurrent = true
			continue
		}
		if !foundCurrent {
			continue
		}
		if tmpl.Match == nil || !tmpl.Match.matchPattern(ec, node) {
			continue
		}
		// Check for conflict at the same priority/import-precedence
		if failOnMultiple {
			for _, other := range templates[i+1:] {
				if other.ImportPrec < tmpl.ImportPrec || (other.ImportPrec == tmpl.ImportPrec && other.Priority < tmpl.Priority) {
					break
				}
				// Split union pattern branches share the same Body slice
				if len(other.Body) > 0 && len(tmpl.Body) > 0 && &other.Body[0] == &tmpl.Body[0] {
					continue
				}
				if other.Match != nil && other.Match.matchPattern(ec, node) {
					return dynamicError(errCodeXTRE0540,
						"ambiguous rule match for node %v in mode %q (on-multiple-match=fail)",
						node, mode)
				}
			}
		}
		return ec.executeTemplate(ctx, tmpl, node, mode, pv)
	}

	// No next match found — apply built-in rules
	return ec.applyBuiltinRules(ctx, node, mode, pv)
}

func (ec *execContext) execApplyImports(ctx context.Context, inst *ApplyImportsInst) error {
	// xsl:apply-imports: find a matching template with lower import precedence
	// than the currently executing template.
	// XTDE0560: xsl:apply-imports when the current template rule is absent.
	// A template rule is a template with a match pattern; named templates
	// invoked via call-template or initial-template don't count.
	if ec.currentTemplate == nil || ec.currentTemplate.Match == nil {
		return dynamicError(errCodeXTDE0560, "xsl:apply-imports: no current template rule")
	}

	node := ec.currentNode
	mode := ec.currentMode
	maxPrec := ec.currentTemplate.ImportPrec

	// Process with-param (tunnel and regular).
	// Copy tunnel params to avoid mutating the caller's map.
	var pv map[string]xpath3.Sequence
	savedTunnel := ec.tunnelParams
	if len(inst.Params) > 0 {
		hasTunnel := false
		for _, wp := range inst.Params {
			if wp.Tunnel {
				hasTunnel = true
				break
			}
		}
		if hasTunnel {
			newTunnel := make(map[string]xpath3.Sequence, len(ec.tunnelParams)+len(inst.Params))
			for k, v := range ec.tunnelParams {
				newTunnel[k] = v
			}
			ec.tunnelParams = newTunnel
		}
		for _, wp := range inst.Params {
			val, err := ec.evaluateWithParam(ctx, wp)
			if err != nil {
				return err
			}
			if wp.Tunnel {
				ec.tunnelParams[wp.Name] = val
			} else {
				if pv == nil {
					pv = make(map[string]xpath3.Sequence)
				}
				pv[wp.Name] = val
			}
		}
	}
	defer func() { ec.tunnelParams = savedTunnel }()

	// xsl:apply-imports searches only within the current module's import
	// tree. MinImportPrec marks the lowest precedence among the module's
	// (transitive) imports, so we restrict the search to [minPrec, maxPrec).
	minPrec := ec.currentTemplate.MinImportPrec

	// XSLT 3.0: when the context item is an atomic value (e.g. from
	// xsl:apply-templates select="1 to 5"), search for matching atomic
	// templates with lower import precedence, then fall back to outputting
	// the string value of the atomic item (built-in rule for atomic values).
	if ec.contextItem != nil {
		templates := ec.stylesheet.modeTemplates[mode]
		for _, tmpl := range templates {
			if tmpl.ImportPrec >= maxPrec || tmpl.ImportPrec < minPrec {
				continue
			}
			if tmpl.Match != nil && ec.matchAtomicPattern(tmpl.Match, ec.contextItem) {
				return ec.executeAtomicTemplate(ctx, tmpl, ec.contextItem, mode)
			}
		}
		// Built-in rule for atomic values: output string value
		av, err := xpath3.AtomizeItem(ec.contextItem)
		if err != nil {
			return nil
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			return nil
		}
		text, err := ec.resultDoc.CreateText([]byte(s))
		if err != nil {
			return err
		}
		return ec.addNode(text)
	}

	templates := ec.stylesheet.modeTemplates[mode]
	for _, tmpl := range templates {
		if tmpl.ImportPrec >= maxPrec || tmpl.ImportPrec < minPrec {
			continue
		}
		if tmpl.Match != nil && tmpl.Match.matchPattern(ec, node) {
			return ec.executeTemplate(ctx, tmpl, node, mode, pv)
		}
	}

	// No imported template found, use built-in rules
	return ec.applyBuiltinRules(ctx, node, mode, pv)
}

// checkContextItemType validates the context item against the template's
// xsl:context-item declaration. Returns XTTE0590 on type mismatch or
// XPDY0002 when the context item is absent but required.
func (ec *execContext) checkContextItemType(tmpl *Template) error {
	if tmpl.ContextItemAs == "" && tmpl.ContextItemUse == "" {
		return nil
	}

	// Determine the current context item
	var contextSeq xpath3.Sequence
	if ec.contextItem != nil {
		contextSeq = xpath3.Sequence{ec.contextItem}
	} else if normalizeNode(ec.contextNode) != nil {
		contextSeq = xpath3.Sequence{xpath3.NodeItem{Node: ec.contextNode}}
	}

	use := tmpl.ContextItemUse
	if use == "" {
		// Default use depends on template type:
		// - match templates default to "required" (context always present)
		// - named templates default to "optional" (may be called without context)
		if tmpl.Match != nil {
			use = "required"
		} else {
			use = "optional"
		}
	}

	// XPDY0002: context item absent when use="required"
	if len(contextSeq) == 0 && use == "required" {
		return dynamicError("XPDY0002", "context item is absent but xsl:context-item use=\"required\"")
	}

	// Type check: if as is specified and context item exists, check the type.
	// Context-item type checking uses "instance of" semantics (no atomization/casting).
	if tmpl.ContextItemAs != "" && len(contextSeq) > 0 {
		asExpr := stripXPathComments(tmpl.ContextItemAs)
		if !instanceOfItemType(contextSeq[0], asExpr, ec) {
			return dynamicError(errCodeXTTE0590,
				"context item does not match required type %s", asExpr)
		}
	}

	return nil
}

// instanceOfItemType checks if an item is an instance of the given item type
// WITHOUT coercion or atomization. This is used for xsl:context-item/@as which
// requires "instance of" semantics per XSLT 3.0 §9.8.
func instanceOfItemType(item xpath3.Item, itemType string, ec *execContext) bool {
	switch itemType {
	case "item()":
		return true
	case "node()":
		_, ok := item.(xpath3.NodeItem)
		return ok
	case "element()":
		ni, ok := item.(xpath3.NodeItem)
		return ok && ni.Node.Type() == helium.ElementNode
	case "attribute()":
		ni, ok := item.(xpath3.NodeItem)
		return ok && ni.Node.Type() == helium.AttributeNode
	case "text()":
		ni, ok := item.(xpath3.NodeItem)
		return ok && (ni.Node.Type() == helium.TextNode || ni.Node.Type() == helium.CDATASectionNode)
	case "comment()":
		ni, ok := item.(xpath3.NodeItem)
		return ok && ni.Node.Type() == helium.CommentNode
	case "processing-instruction()":
		ni, ok := item.(xpath3.NodeItem)
		return ok && ni.Node.Type() == helium.ProcessingInstructionNode
	case "document-node()":
		ni, ok := item.(xpath3.NodeItem)
		return ok && ni.Node.Type() == helium.DocumentNode
	}

	// Handle element(name) / element(name, type)
	if strings.HasPrefix(itemType, "element(") {
		ni, ok := item.(xpath3.NodeItem)
		if !ok || ni.Node.Type() != helium.ElementNode {
			return false
		}
		inner := strings.TrimSpace(itemType[len("element(") : len(itemType)-1])
		if inner == "" || inner == "*" {
			return true
		}
		parts := splitTopLevelTypeArgs(inner)
		reqName := strings.TrimSpace(parts[0])
		if reqName != "*" {
			reqLocal, reqNS := resolveSchemaQName(reqName, ec)
			elem, isElem := ni.Node.(*helium.Element)
			if !isElem || elem.LocalName() != reqLocal || elem.URI() != reqNS {
				return false
			}
		}
		return true
	}

	// Handle document-node(element(...))
	if strings.HasPrefix(itemType, "document-node(") {
		ni, ok := item.(xpath3.NodeItem)
		if !ok || ni.Node.Type() != helium.DocumentNode {
			return false
		}
		inner := strings.TrimSpace(itemType[len("document-node(") : len(itemType)-1])
		if inner == "" {
			return true
		}
		doc, isDoc := ni.Node.(*helium.Document)
		if !isDoc {
			return false
		}
		rootElem := findDocumentElement(doc)
		if rootElem == nil {
			return false
		}
		return instanceOfItemType(xpath3.NodeItem{Node: rootElem}, inner, ec)
	}

	// Atomic types: item must be an atomic value of the specified type
	av, ok := item.(xpath3.AtomicValue)
	if !ok {
		// Node items are not instances of atomic types
		return false
	}
	target := normalizeTypeName(itemType, ec)
	if target == "xs:anyAtomicType" {
		return true
	}
	if av.TypeName == target {
		return true
	}
	// Check built-in subtype relationships (e.g., xs:integer is instance of xs:decimal)
	if isBuiltinSubtypeOf(av.TypeName, target) {
		return true
	}
	// Check schema-derived subtype relationships
	if ec != nil && ec.schemaRegistry != nil {
		if ec.schemaRegistry.IsSubtypeOf(av.TypeName, target) {
			return true
		}
	}
	return false
}

// stripXPathComments removes XPath 2.0+ comments (: ... :) from a string
// and normalizes whitespace (collapses runs of whitespace into single spaces,
// removes spaces adjacent to parentheses).
func stripXPathComments(s string) string {
	// First pass: remove comments
	var buf strings.Builder
	depth := 0
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '(' && s[i+1] == ':' {
			depth++
			i++ // skip ':'
			continue
		}
		if i+1 < len(s) && s[i] == ':' && s[i+1] == ')' {
			if depth > 0 {
				depth--
			}
			i++ // skip ')'
			continue
		}
		if depth == 0 {
			buf.WriteByte(s[i])
		}
	}
	// Normalize whitespace: collapse runs into single space, trim
	result := strings.Join(strings.Fields(buf.String()), " ")
	// Remove spaces around parentheses and commas for canonical form:
	// "element ( doc )" → "element(doc)"
	result = strings.ReplaceAll(result, " (", "(")
	result = strings.ReplaceAll(result, "( ", "(")
	result = strings.ReplaceAll(result, " )", ")")
	result = strings.ReplaceAll(result, ") ", ")")
	result = strings.ReplaceAll(result, " ,", ",")
	result = strings.ReplaceAll(result, ", ", ",")
	return result
}
