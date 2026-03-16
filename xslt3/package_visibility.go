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
)

// defaultComponentVisibility returns the default visibility for a component
// in the given stylesheet context.
func defaultComponentVisibility(ss *Stylesheet) string {
	if ss.isPackage {
		return visPrivate
	}
	return visPublic
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
		if !ok || elem.URI() != NSXSLT || elem.LocalName() != "accept" {
			continue
		}
		comp := getAttr(elem, "component")
		names := getAttr(elem, "names")
		vis := getAttr(elem, "visibility")
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

	if name == "*" {
		return "*" + arity
	}

	// Handle *:local pattern
	if strings.HasPrefix(name, "*:") {
		return name + arity
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
	if pattern == "*" {
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
		if rule.component != "*" && rule.component != compType {
			continue
		}
		if !componentNameMatches(compName, rule.names) {
			continue
		}

		// Calculate specificity: exact name > prefix:* > *
		spec := 0
		if rule.names == "*" {
			spec = 0
		} else if strings.HasSuffix(rule.names, "}*") || strings.HasPrefix(rule.names, "*:") {
			spec = 1
		} else {
			spec = 2
		}
		// Component type specificity
		if rule.component != "*" {
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
	for qn, fn := range c.stylesheet.functions {
		key := functionVisKey(qn, len(fn.Params))
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

	// Process xsl:expose children
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != NSXSLT || elem.LocalName() != "expose" {
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
	for qn, fn := range c.stylesheet.functions {
		key := functionVisKey(qn, len(fn.Params))
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
	component := getAttr(elem, "component")
	names := getAttr(elem, "names")
	visibility := getAttr(elem, "visibility")

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
		"template": {}, "function": {}, "variable": {},
		"attribute-set": {}, "mode": {}, "*": {},
	}
	if _, ok := validComponents[component]; !ok {
		return staticError(errCodeXTSE0020, "invalid component type %q on xsl:expose", component)
	}

	// Collect namespace bindings from the element's context
	nsBindings := c.collectElemNamespaces(elem)

	for _, name := range strings.Fields(names) {
		resolvedName := resolveComponentName(name, nsBindings, elem)

		switch component {
		case "template", "*":
			if component == "template" || component == "*" {
				if err := c.applyExposeToTemplates(resolvedName, visibility, component == "*"); err != nil {
					return err
				}
			}
		case "function":
			if err := c.applyExposeToFunctions(resolvedName, visibility); err != nil {
				return err
			}
		case "variable":
			if err := c.applyExposeToVariables(resolvedName, visibility); err != nil {
				return err
			}
		case "attribute-set":
			if err := c.applyExposeToAttrSets(resolvedName, visibility); err != nil {
				return err
			}
		case "mode":
			if err := c.applyExposeToModes(resolvedName, visibility); err != nil {
				return err
			}
		}

		// When component="*", apply to all component types
		if component == "*" {
			_ = c.applyExposeToFunctions(resolvedName, visibility)
			_ = c.applyExposeToVariables(resolvedName, visibility)
			_ = c.applyExposeToAttrSets(resolvedName, visibility)
			_ = c.applyExposeToModes(resolvedName, visibility)
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
	for name := range c.stylesheet.templateVisibility {
		if componentNameMatches(name, pattern) {
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
	for key := range c.stylesheet.functionVisibility {
		if componentNameMatches(key, pattern) {
			c.stylesheet.functionVisibility[key] = visibility
			matched = true
		}
	}
	// Don't error for wildcard component="*" if no functions match
	_ = matched
	return nil
}

func (c *compiler) applyExposeToVariables(pattern, visibility string) error {
	matchedVar := false
	for name := range c.stylesheet.variableVisibility {
		if componentNameMatches(name, pattern) {
			c.stylesheet.variableVisibility[name] = visibility
			matchedVar = true
		}
	}
	for name := range c.stylesheet.globalParamVisibility {
		if componentNameMatches(name, pattern) {
			c.stylesheet.globalParamVisibility[name] = visibility
			matchedVar = true
		}
	}
	_ = matchedVar
	return nil
}

func (c *compiler) applyExposeToAttrSets(pattern, visibility string) error {
	for name := range c.stylesheet.attrSetVisibility {
		if componentNameMatches(name, pattern) {
			c.stylesheet.attrSetVisibility[name] = visibility
		}
	}
	return nil
}

func (c *compiler) applyExposeToModes(pattern, visibility string) error {
	if c.stylesheet.modeDefs == nil {
		return nil
	}
	for _, md := range c.stylesheet.modeDefs {
		if componentNameMatches(md.Name, pattern) {
			md.Visibility = visibility
		}
	}
	return nil
}

func isWildcard(pattern string) bool {
	return pattern == "*" || strings.HasSuffix(pattern, "}*") || strings.HasPrefix(pattern, "*:")
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
		if !ok || elem.URI() != NSXSLT || elem.LocalName() != "override" {
			continue
		}
		for oc := elem.FirstChild(); oc != nil; oc = oc.NextSibling() {
			oe, ok := oc.(*helium.Element)
			if !ok || oe.URI() != NSXSLT {
				continue
			}
			switch oe.LocalName() {
			case "template":
				name := resolveQName(getAttr(oe, "name"), nsBindings)
				if name != "" {
					names["template:"+name] = struct{}{}
				}
			case "function":
				name := getAttr(oe, "name")
				if name != "" {
					resolved := resolveQName(name, nsBindings)
					names["function:"+resolved] = struct{}{}
				}
			case "variable":
				name := resolveQName(getAttr(oe, "name"), nsBindings)
				if name != "" {
					names["variable:"+name] = struct{}{}
				}
			case "param":
				name := resolveQName(getAttr(oe, "name"), nsBindings)
				if name != "" {
					names["variable:"+name] = struct{}{}
				}
			case "attribute-set":
				name := resolveQName(getAttr(oe, "name"), nsBindings)
				if name != "" {
					names["attribute-set:"+name] = struct{}{}
				}
			}
		}
	}
	return names
}
