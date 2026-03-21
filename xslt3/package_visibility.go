package xslt3

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// componentVisibility tracks the effective visibility of a component.
// The default visibility for package components is "private" per XSLT 3.0 §3.6.1.
// The default for stylesheet (non-package) components is "public".
const (
	visPublic   = "public"
	visPrivate  = "private"
	visFinal    = "final"
	visAbstract = "abstract"
	visHidden   = "hidden"

	xslAttrComponent  = "component"
	xslAttrNames      = "names"
	xslAttrVisibility = "visibility"

	xslElemAccept       = "accept"
	xslElemExpose       = "expose"
	xslElemOverride     = "override"
	xslElemTemplate     = "template"
	xslElemFunction     = "function"
	xslElemVariable     = "variable"
	xslElemParam        = "param"
	xslElemAttributeSet = "attribute-set"
	xslElemMode         = "mode"

	xslWildcard = "*"
)

// defaultComponentVisibility returns the default visibility for a component
// in the given stylesheet context.
func defaultComponentVisibility(ss *Stylesheet) string {
	if ss.isPackage {
		return visPrivate
	}
	return visPublic
}

// visibilityLevel returns a numeric accessibility level for visibility.
// Higher level = more accessible. Used to enforce the rule that xsl:expose
// cannot make a component more accessible than its declared level.
// "final" and "public" are at the same accessibility level.
func visibilityLevel(vis string) int {
	switch vis {
	case visHidden:
		return 0
	case visPrivate:
		return 1
	case visPublic, visFinal, visAbstract:
		return 2
	default:
		return 1 // treat unknown as private
	}
}

// acceptRule describes one xsl:accept child of xsl:use-package.
type acceptRule struct {
	component  string // "template", "function", "variable", "attribute-set", "mode", "*"
	names      string // name pattern (e.g., "*", "foo", "ns:*")
	visibility string // target visibility
}

// parseAcceptRules extracts xsl:accept children from an xsl:use-package element.
func parseAcceptRules(usePackageElem *helium.Element, nsBindings map[string]string) []acceptRule {
	var rules []acceptRule
	for child := usePackageElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != NSXSLT || elem.LocalName() != xslElemAccept {
			continue
		}
		comp := getAttr(elem, xslAttrComponent)
		names := getAttr(elem, xslAttrNames)
		vis := getAttr(elem, xslAttrVisibility)
		// Resolve namespace prefixes in names
		for _, name := range strings.Fields(names) {
			resolvedName := resolveComponentName(name, nsBindings, elem)
			rules = append(rules, acceptRule{
				component:  comp,
				names:      resolvedName,
				visibility: vis,
			})
		}
	}
	return rules
}

// resolveComponentName resolves a component name (possibly prefixed) to an
// expanded Clark notation name. Handles wildcards like "*", "ns:*", "*:local".
func resolveComponentName(name string, nsBindings map[string]string, elem *helium.Element) string {
	// Handle function arity suffix (e.g., "f#2")
	arity := ""
	if idx := strings.LastIndex(name, "#"); idx >= 0 {
		arity = name[idx:]
		name = name[:idx]
	}

	if name == xslWildcard {
		return xslWildcard + arity
	}

	// Handle *:local pattern
	if strings.HasPrefix(name, "*:") {
		return name + arity
	}

	// Handle EQName notation: Q{uri}local
	if strings.HasPrefix(name, "Q{") {
		// Convert Q{uri}local to {uri}local (Clark notation)
		return name[1:] + arity
	}

	// Handle prefix:* or prefix:local
	if idx := strings.Index(name, ":"); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		// Look up namespace URI for prefix
		uri := ""
		if nsBindings != nil {
			uri = nsBindings[prefix]
		}
		// Also check element's own namespace declarations
		if uri == "" {
			for n := helium.Node(elem); n != nil; n = n.Parent() {
				if e, ok := n.(*helium.Element); ok {
					for _, ns := range e.Namespaces() {
						if ns.Prefix() == prefix {
							uri = ns.URI()
							break
						}
					}
					if uri != "" {
						break
					}
				}
			}
		}
		if uri != "" {
			if local == "*" {
				return "{" + uri + "}*" + arity
			}
			return "{" + uri + "}" + local + arity
		}
		// If no URI found, return as-is (may be an error in the stylesheet)
		return name + arity
	}

	return name + arity
}

// componentNameMatches checks if a component name matches a name pattern.
// Patterns: "*" matches all, "{ns}*" matches all in namespace, "*:local" matches
// any namespace with that local name, "{ns}local" matches exactly.
func componentNameMatches(compName, pattern string) bool {
	if pattern == xslWildcard {
		return true
	}

	// Handle arity in pattern (e.g., "f#2")
	patternArity := ""
	if idx := strings.LastIndex(pattern, "#"); idx >= 0 {
		patternArity = pattern[idx:]
		pattern = pattern[:idx]
	}
	compArity := ""
	compNameBase := compName
	if idx := strings.LastIndex(compName, "#"); idx >= 0 {
		compArity = compName[idx:]
		compNameBase = compName[:idx]
	}

	// If pattern has arity, it must match
	if patternArity != "" && patternArity != compArity {
		return false
	}

	// {ns}* pattern — matches any local name in that namespace
	if strings.HasSuffix(pattern, "}*") && strings.HasPrefix(pattern, "{") {
		ns := pattern[1 : len(pattern)-2]
		return strings.HasPrefix(compNameBase, "{"+ns+"}")
	}

	// *:local pattern — matches any namespace with that local name
	if strings.HasPrefix(pattern, "*:") {
		local := pattern[2:]
		compLocal := compNameBase
		if idx := strings.LastIndex(compNameBase, "}"); idx >= 0 {
			compLocal = compNameBase[idx+1:]
		}
		return compLocal == local
	}

	// Exact match
	return compNameBase == pattern
}

// applyAcceptRules determines the effective visibility of a component after
// applying xsl:accept rules. Returns the visibility and whether the component
// should be included (hidden components are excluded).
// acceptRules are processed in order; more specific patterns override wildcards.
func applyAcceptRules(compType, compName string, rules []acceptRule, defaultVis string) string {
	bestVis := defaultVis
	bestSpecificity := -1

	for _, rule := range rules {
		// Check component type matches
		if rule.component != xslWildcard && rule.component != compType {
			continue
		}
		if !componentNameMatches(compName, rule.names) {
			continue
		}

		// Calculate specificity: exact name > prefix:* > *
		spec := 0
		if rule.names == xslWildcard {
			spec = 0
		} else if strings.HasSuffix(rule.names, "}*") || strings.HasPrefix(rule.names, "*:") {
			spec = 1
		} else {
			spec = 2
		}
		// Component type specificity
		if rule.component != xslWildcard {
			spec += 3
		}

		if spec > bestSpecificity {
			bestSpecificity = spec
			bestVis = rule.visibility
		}
	}

	return bestVis
}

// processExpose processes xsl:expose children of xsl:package to set component
// visibility. This is called during package compilation.
func (c *compiler) processExpose(root *helium.Element) error {
	if !c.stylesheet.isPackage {
		return nil
	}

	// Initialize visibility maps
	c.stylesheet.templateVisibility = make(map[string]string)
	c.stylesheet.functionVisibility = make(map[string]string)
	c.stylesheet.variableVisibility = make(map[string]string)
	c.stylesheet.attrSetVisibility = make(map[string]string)
	c.stylesheet.globalParamVisibility = make(map[string]string)

	// Set defaults: all package components default to private
	for _, tmpl := range c.stylesheet.templates {
		if tmpl.Name != "" {
			if tmpl.Visibility != "" {
				c.stylesheet.templateVisibility[tmpl.Name] = tmpl.Visibility
			} else {
				c.stylesheet.templateVisibility[tmpl.Name] = visPrivate
			}
		}
	}
	for fk, fn := range c.stylesheet.functions {
		key := functionVisKey(fk.Name, len(fn.Params))
		if fn.Visibility != "" {
			c.stylesheet.functionVisibility[key] = fn.Visibility
		} else {
			c.stylesheet.functionVisibility[key] = visPrivate
		}
	}
	for _, v := range c.stylesheet.globalVars {
		if v.Visibility != "" {
			c.stylesheet.variableVisibility[v.Name] = v.Visibility
		} else {
			c.stylesheet.variableVisibility[v.Name] = visPrivate
		}
	}
	for _, p := range c.stylesheet.globalParams {
		if p.Visibility != "" {
			c.stylesheet.globalParamVisibility[p.Name] = p.Visibility
		} else {
			c.stylesheet.globalParamVisibility[p.Name] = visPrivate
		}
	}
	for name := range c.stylesheet.attributeSets {
		c.stylesheet.attrSetVisibility[name] = visPrivate
	}
	if c.stylesheet.modeDefs != nil {
		for _, md := range c.stylesheet.modeDefs {
			if md.Visibility == "" {
				md.Visibility = visPrivate
			}
		}
	}

	// Save explicitly declared visibility before expose rules modify them.
	// Only store components that had an explicit visibility attribute — default
	// (private) does not restrict expose. XTSE3010 only applies when the
	// component was explicitly declared with a visibility.
	c.declaredTemplateVis = make(map[string]string)
	for _, tmpl := range c.stylesheet.templates {
		if tmpl.Name != "" && tmpl.Visibility != "" {
			c.declaredTemplateVis[tmpl.Name] = tmpl.Visibility
		}
	}
	c.declaredFunctionVis = make(map[string]string)
	for fk, fn := range c.stylesheet.functions {
		if fn.Visibility != "" {
			key := functionVisKey(fk.Name, len(fn.Params))
			c.declaredFunctionVis[key] = fn.Visibility
		}
	}
	c.declaredVariableVis = make(map[string]string)
	for _, v := range c.stylesheet.globalVars {
		if v.Visibility != "" {
			c.declaredVariableVis[v.Name] = v.Visibility
		}
	}
	c.declaredAttrSetVis = make(map[string]string)
	// Attribute sets have no explicit visibility attribute in XSLT;
	// they always default to private, so no restriction needed.
	c.declaredParamVis = make(map[string]string)
	for _, p := range c.stylesheet.globalParams {
		if p.Visibility != "" {
			c.declaredParamVis[p.Name] = p.Visibility
		}
	}

	// Process xsl:expose children
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != NSXSLT || elem.LocalName() != xslElemExpose {
			continue
		}
		if err := c.compileExpose(elem); err != nil {
			return err
		}
	}

	// Now update the actual component visibility fields
	for _, tmpl := range c.stylesheet.templates {
		if tmpl.Name != "" {
			if vis, ok := c.stylesheet.templateVisibility[tmpl.Name]; ok {
				tmpl.Visibility = vis
			}
		}
	}
	for fk, fn := range c.stylesheet.functions {
		key := functionVisKey(fk.Name, len(fn.Params))
		if vis, ok := c.stylesheet.functionVisibility[key]; ok {
			fn.Visibility = vis
		}
	}
	for _, v := range c.stylesheet.globalVars {
		if vis, ok := c.stylesheet.variableVisibility[v.Name]; ok {
			v.Visibility = vis
		}
	}
	for _, p := range c.stylesheet.globalParams {
		if vis, ok := c.stylesheet.globalParamVisibility[p.Name]; ok {
			p.Visibility = vis
		}
	}

	return nil
}

// compileExpose processes a single xsl:expose element.
func (c *compiler) compileExpose(elem *helium.Element) error {
	component := getAttr(elem, xslAttrComponent)
	names := getAttr(elem, xslAttrNames)
	visibility := getAttr(elem, xslAttrVisibility)

	if component == "" || names == "" || visibility == "" {
		return staticError(errCodeXTSE0010, "xsl:expose requires component, names, and visibility attributes")
	}

	// Validate visibility value
	switch visibility {
	case visPublic, visPrivate, visFinal, visAbstract, visHidden:
		// ok
	default:
		return staticError(errCodeXTSE0020, "invalid visibility value %q on xsl:expose", visibility)
	}

	// Validate component type
	validComponents := map[string]struct{}{
		xslElemTemplate: {}, xslElemFunction: {}, xslElemVariable: {},
		xslElemAttributeSet: {}, xslElemMode: {}, xslWildcard: {},
	}
	if _, ok := validComponents[component]; !ok {
		return staticError(errCodeXTSE0020, "invalid component type %q on xsl:expose", component)
	}

	// Collect namespace bindings from the element's context
	nsBindings := c.collectElemNamespaces(elem)

	for _, name := range strings.Fields(names) {
		resolvedName := resolveComponentName(name, nsBindings, elem)

		switch component {
		case xslElemTemplate:
			if err := c.applyExposeToTemplates(resolvedName, visibility, false); err != nil {
				return err
			}
		case xslElemFunction:
			if err := c.applyExposeToFunctionsStrict(resolvedName, visibility); err != nil {
				return err
			}
		case xslElemVariable:
			if err := c.applyExposeToVariablesStrict(resolvedName, visibility); err != nil {
				return err
			}
		case xslElemAttributeSet:
			if err := c.applyExposeToAttrSets(resolvedName, visibility, true); err != nil {
				return err
			}
		case xslElemMode:
			if err := c.applyExposeToModes(resolvedName, visibility, true); err != nil {
				return err
			}
		case xslWildcard:
			// When component="*", apply to all component types.
			// Use non-strict mode so we don't error when a name matches
			// in one component type but not others.
			_ = c.applyExposeToTemplates(resolvedName, visibility, true)
			_ = c.applyExposeToFunctions(resolvedName, visibility)
			_ = c.applyExposeToVariables(resolvedName, visibility)
			_ = c.applyExposeToAttrSets(resolvedName, visibility, false)
			_ = c.applyExposeToModes(resolvedName, visibility, false)
		}
	}

	return nil
}

// collectElemNamespaces gathers namespace bindings from an element and its ancestors.
func (c *compiler) collectElemNamespaces(elem *helium.Element) map[string]string {
	bindings := make(map[string]string)
	for k, v := range c.nsBindings {
		bindings[k] = v
	}
	for n := helium.Node(elem); n != nil; n = n.Parent() {
		if e, ok := n.(*helium.Element); ok {
			for _, ns := range e.Namespaces() {
				if _, exists := bindings[ns.Prefix()]; !exists {
					bindings[ns.Prefix()] = ns.URI()
				}
			}
		}
	}
	return bindings
}

func (c *compiler) applyExposeToTemplates(pattern, visibility string, isWildcardComponent bool) error {
	matched := false
	isWild := isWildcard(pattern)
	for name := range c.stylesheet.templateVisibility {
		if componentNameMatches(name, pattern) {
			// XTSE3010: cannot increase visibility beyond declared level
			// (only checked for non-wildcard patterns; wildcards silently skip)
			if declared, ok := c.declaredTemplateVis[name]; ok {
				if visibilityLevel(visibility) > visibilityLevel(declared) {
					if isWild {
						continue // skip this component for wildcard patterns
					}
					return staticError(errCodeXTSE3010,
						"xsl:expose: cannot increase visibility of template %q from %s to %s", name, declared, visibility)
				}
			}
			c.stylesheet.templateVisibility[name] = visibility
			matched = true
		}
	}
	if !matched && !isWildcard(pattern) && !isWildcardComponent {
		return staticError(errCodeXTSE3010, "xsl:expose: no template %q found in package", pattern)
	}
	return nil
}

func (c *compiler) applyExposeToFunctions(pattern, visibility string) error {
	matched := false
	isWild := isWildcard(pattern)
	for key := range c.stylesheet.functionVisibility {
		if componentNameMatches(key, pattern) {
			if declared, ok := c.declaredFunctionVis[key]; ok {
				if visibilityLevel(visibility) > visibilityLevel(declared) {
					if isWild {
						continue
					}
					return staticError(errCodeXTSE3010,
						"xsl:expose: cannot increase visibility of function %q from %s to %s", key, declared, visibility)
				}
			}
			c.stylesheet.functionVisibility[key] = visibility
			matched = true
		}
	}
	_ = matched
	return nil
}

// applyExposeToFunctionsStrict is like applyExposeToFunctions but reports
// XTSE3010 when a non-wildcard pattern has no match. Used when the expose
// element has component="function" (not component="*").
func (c *compiler) applyExposeToFunctionsStrict(pattern, visibility string) error {
	matched := false
	isWild := isWildcard(pattern)
	for key := range c.stylesheet.functionVisibility {
		if componentNameMatches(key, pattern) {
			if declared, ok := c.declaredFunctionVis[key]; ok {
				if visibilityLevel(visibility) > visibilityLevel(declared) {
					if isWild {
						continue
					}
					return staticError(errCodeXTSE3010,
						"xsl:expose: cannot increase visibility of function %q from %s to %s", key, declared, visibility)
				}
			}
			c.stylesheet.functionVisibility[key] = visibility
			matched = true
		}
	}
	if !matched && !isWildcard(pattern) {
		return staticError(errCodeXTSE3010, "xsl:expose: no function %q found in package", pattern)
	}
	return nil
}

func (c *compiler) applyExposeToVariables(pattern, visibility string) error {
	matched := false
	isWild := isWildcard(pattern)
	for name := range c.stylesheet.variableVisibility {
		if componentNameMatches(name, pattern) {
			if declared, ok := c.declaredVariableVis[name]; ok {
				if visibilityLevel(visibility) > visibilityLevel(declared) {
					if isWild {
						continue
					}
					return staticError(errCodeXTSE3010,
						"xsl:expose: cannot increase visibility of variable %q from %s to %s", name, declared, visibility)
				}
			}
			c.stylesheet.variableVisibility[name] = visibility
			matched = true
		}
	}
	for name := range c.stylesheet.globalParamVisibility {
		if componentNameMatches(name, pattern) {
			if declared, ok := c.declaredParamVis[name]; ok {
				if visibilityLevel(visibility) > visibilityLevel(declared) {
					if isWild {
						continue
					}
					return staticError(errCodeXTSE3010,
						"xsl:expose: cannot increase visibility of parameter %q from %s to %s", name, declared, visibility)
				}
			}
			c.stylesheet.globalParamVisibility[name] = visibility
			matched = true
		}
	}
	_ = matched
	return nil
}

// applyExposeToVariablesStrict is like applyExposeToVariables but reports
// XTSE3010 when a non-wildcard pattern has no match.
func (c *compiler) applyExposeToVariablesStrict(pattern, visibility string) error {
	matched := false
	isWild := isWildcard(pattern)
	for name := range c.stylesheet.variableVisibility {
		if componentNameMatches(name, pattern) {
			if declared, ok := c.declaredVariableVis[name]; ok {
				if visibilityLevel(visibility) > visibilityLevel(declared) {
					if isWild {
						continue
					}
					return staticError(errCodeXTSE3010,
						"xsl:expose: cannot increase visibility of variable %q from %s to %s", name, declared, visibility)
				}
			}
			c.stylesheet.variableVisibility[name] = visibility
			matched = true
		}
	}
	for name := range c.stylesheet.globalParamVisibility {
		if componentNameMatches(name, pattern) {
			if declared, ok := c.declaredParamVis[name]; ok {
				if visibilityLevel(visibility) > visibilityLevel(declared) {
					if isWild {
						continue
					}
					return staticError(errCodeXTSE3010,
						"xsl:expose: cannot increase visibility of parameter %q from %s to %s", name, declared, visibility)
				}
			}
			c.stylesheet.globalParamVisibility[name] = visibility
			matched = true
		}
	}
	if !matched && !isWildcard(pattern) {
		return staticError(errCodeXTSE3010, "xsl:expose: no variable or parameter %q found in package", pattern)
	}
	return nil
}

func (c *compiler) applyExposeToAttrSets(pattern, visibility string, strict bool) error {
	matched := false
	isWild := isWildcard(pattern)
	for name := range c.stylesheet.attrSetVisibility {
		if componentNameMatches(name, pattern) {
			if declared, ok := c.declaredAttrSetVis[name]; ok {
				if visibilityLevel(visibility) > visibilityLevel(declared) {
					if isWild {
						continue
					}
					return staticError(errCodeXTSE3010,
						"xsl:expose: cannot increase visibility of attribute-set %q from %s to %s", name, declared, visibility)
				}
			}
			c.stylesheet.attrSetVisibility[name] = visibility
			matched = true
		}
	}
	if strict && !matched && !isWildcard(pattern) {
		return staticError(errCodeXTSE3010, "xsl:expose: no attribute-set %q found in package", pattern)
	}
	return nil
}

func (c *compiler) applyExposeToModes(pattern, visibility string, strict bool) error {
	if c.stylesheet.modeDefs == nil {
		if strict && !isWildcard(pattern) {
			return staticError(errCodeXTSE3010, "xsl:expose: no mode %q found in package", pattern)
		}
		return nil
	}
	matched := false
	for _, md := range c.stylesheet.modeDefs {
		if componentNameMatches(md.Name, pattern) {
			md.Visibility = visibility
			matched = true
		}
	}
	if strict && !matched && !isWildcard(pattern) {
		return staticError(errCodeXTSE3010, "xsl:expose: no mode %q found in package", pattern)
	}
	return nil
}

func isWildcard(pattern string) bool {
	return pattern == xslWildcard || strings.HasSuffix(pattern, "}*") || strings.HasPrefix(pattern, "*:")
}

// functionVisKey creates a visibility map key for a function (name#arity).
func functionVisKey(qn xpath3.QualifiedName, arity int) string {
	return fmt.Sprintf("{%s}%s#%d", qn.URI, qn.Name, arity)
}

// collectOverrideNames scans xsl:override children of xsl:use-package and returns
// a set of "type:name" keys for components being overridden.
func (c *compiler) collectOverrideNames(usePackageElem *helium.Element, nsBindings map[string]string) map[string]struct{} {
	names := make(map[string]struct{})
	for child := usePackageElem.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != NSXSLT || elem.LocalName() != xslElemOverride {
			continue
		}
		for oc := elem.FirstChild(); oc != nil; oc = oc.NextSibling() {
			oe, ok := oc.(*helium.Element)
			if !ok || oe.URI() != NSXSLT {
				continue
			}
			switch oe.LocalName() {
			case xslElemTemplate:
				name := resolveQName(getAttr(oe, "name"), nsBindings)
				if name != "" {
					names[xslElemTemplate+":"+name] = struct{}{}
				}
			case xslElemFunction:
				name := getAttr(oe, "name")
				if name != "" {
					resolved := resolveQName(name, nsBindings)
					names[xslElemFunction+":"+resolved] = struct{}{}
				}
			case xslElemVariable:
				name := resolveQName(getAttr(oe, "name"), nsBindings)
				if name != "" {
					names[xslElemVariable+":"+name] = struct{}{}
				}
			case xslElemParam:
				name := resolveQName(getAttr(oe, "name"), nsBindings)
				if name != "" {
					names[xslElemVariable+":"+name] = struct{}{}
				}
			case xslElemAttributeSet:
				name := resolveQName(getAttr(oe, "name"), nsBindings)
				if name != "" {
					names[xslElemAttributeSet+":"+name] = struct{}{}
				}
			}
		}
	}
	return names
}
