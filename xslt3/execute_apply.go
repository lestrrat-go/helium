package xslt3

import (
	"context"
	"fmt"

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
		if ec.contextNode == nil {
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

	// XSLT 3.0: process atomic values — try template matching first,
	// then fall back to built-in text output
	for _, item := range atomicItems {
		if tmpl := ec.findAtomicTemplate(item, mode); tmpl != nil {
			if err := ec.executeAtomicTemplate(ctx, tmpl, item, mode); err != nil {
				return err
			}
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
		checked, err := checkSequenceType(val, st, errCodeXTTE0570, "with-param $"+wp.Name)
		if err != nil {
			return nil, err
		}
		val = checked
	}

	return val, nil
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

	tmpl, ok := ec.stylesheet.namedTemplates[inst.Name]
	if !ok {
		return dynamicError(errCodeXTDE0060, "named template %q not found", inst.Name)
	}

	// Switch package function scope if the template belongs to a different package.
	savedFnsNS := ec.cachedFnsNS
	savedPackage := ec.currentPackage
	savedTemplate := ec.currentTemplate
	if tmpl.OwnerPackage != nil && tmpl.OwnerPackage != ec.currentPackage {
		ec.cachedFnsNS = nil
		ec.currentPackage = tmpl.OwnerPackage
	}
	ec.currentTemplate = tmpl
	defer func() {
		ec.cachedFnsNS = savedFnsNS
		ec.currentPackage = savedPackage
		ec.currentTemplate = savedTemplate
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
		if p.Tunnel {
			// Tunnel param: receive from tunnel context
			if ec.tunnelParams != nil {
				if val, ok := ec.tunnelParams[p.Name]; ok {
					ec.setVar(p.Name, val)
					continue
				}
			}
		} else if val, ok := paramOverrides[p.Name]; ok {
			ec.setVar(p.Name, val)
			continue
		}
		// Use default value
		if p.Select != nil {
			xpathCtx := ec.newXPathContext(ec.contextNode)
			result, err := p.Select.Evaluate(xpathCtx, ec.contextNode)
			if err != nil {
				return err
			}
			ec.setVar(p.Name, result.Sequence())
		} else if len(p.Body) > 0 {
			// Per XSLT spec: param body without select produces a
			// document node (temporary tree).
			ec.temporaryOutputDepth++
			val, err := ec.evaluateBodyAsDocument(ctx, p.Body)
			ec.temporaryOutputDepth--
			if err != nil {
				return err
			}
			ec.setVar(p.Name, val)
		} else {
			ec.setVar(p.Name, xpath3.EmptySequence())
		}
	}

	if tmpl.As != "" {
		return ec.executeTemplateBodyWithAs(ctx, tmpl)
	}

	return ec.executeSequenceConstructor(ctx, tmpl.Body)
}

func (ec *execContext) execNextMatch(ctx context.Context, inst *NextMatchInst) error {
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
	if ec.currentTemplate == nil {
		return nil
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
