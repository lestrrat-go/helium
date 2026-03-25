package xslt3

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

// overrideSet collects compiled override components from xsl:override.
type overrideSet struct {
	functions      map[funcKey]*xslFunction
	namedTemplates map[string]*template
	matchTemplates []*template
	variables      map[string]*variable
	params         map[string]*param
	attributeSets  map[string]*attributeSetDef
}

// processOverrides handles xsl:override children of xsl:use-package.
// It compiles the override children in the context of the package components,
// validates they match existing package components, and returns the override set.
func (c *compiler) processOverrides(usePackageElem *helium.Element, pkg *Stylesheet) (*overrideSet, error) {
	oset := &overrideSet{
		functions:      make(map[funcKey]*xslFunction),
		namedTemplates: make(map[string]*template),
		variables:      make(map[string]*variable),
		params:         make(map[string]*param),
		attributeSets:  make(map[string]*attributeSetDef),
	}

	for child := range helium.Children(usePackageElem) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT || elem.LocalName() != xslElemOverride {
			continue
		}

		if err := c.compileOverrideChildren(elem, pkg, oset); err != nil {
			return nil, err
		}
	}

	return oset, nil
}

// compileOverrideChildren compiles children of an xsl:override element.
func (c *compiler) compileOverrideChildren(overrideElem *helium.Element, pkg *Stylesheet, oset *overrideSet) error {
	// Push namespace bindings from override element
	c.collectNamespaces(overrideElem)

	// Handle default-mode on xsl:override
	savedDefaultMode := c.defaultMode
	if dm := getAttr(overrideElem, "default-mode"); dm != "" {
		c.defaultMode = c.resolveMode(dm)
	}
	defer func() { c.defaultMode = savedDefaultMode }()

	for child := range helium.Children(overrideElem) {
		elem, ok := child.(*helium.Element)
		if !ok {
			// Check for non-whitespace text nodes
			if child.Type() == helium.TextNode || child.Type() == helium.CDATASectionNode {
				if strings.TrimSpace(string(child.Content())) != "" {
					return staticError(errCodeXTSE0010, "text is not allowed as a child of xsl:override")
				}
			}
			continue
		}
		if elem.URI() != lexicon.NamespaceXSLT {
			return staticError(errCodeXTSE0010,
				"non-XSLT element %q is not allowed inside xsl:override", elem.Name())
		}

		switch elem.LocalName() {
		case xslElemFunction:
			fn, qn, err := c.compileOverrideFunction(elem, pkg)
			if err != nil {
				return err
			}
			fk := funcKey{Name: qn, Arity: len(fn.Params)}
			if _, dup := oset.functions[fk]; dup {
				return staticError(errCodeXTSE0770,
					"duplicate override of function %s#%d in xsl:override",
					fmt.Sprintf("{%s}%s", qn.URI, qn.Name), len(fn.Params))
			}
			oset.functions[fk] = fn

		case xslElemTemplate:
			tmpl, err := c.compileOverrideTemplate(elem, pkg)
			if err != nil {
				return err
			}
			if tmpl.Name != "" {
				oset.namedTemplates[tmpl.Name] = tmpl
			}
			if tmpl.Match != nil {
				oset.matchTemplates = append(oset.matchTemplates, tmpl)
			}

		case xslElemVariable:
			v, err := c.compileOverrideVariable(elem, pkg)
			if err != nil {
				return err
			}
			oset.variables[v.Name] = v

		case xslElemParam:
			p, err := c.compileOverrideParam(elem, pkg)
			if err != nil {
				return err
			}
			oset.params[p.Name] = p

		case xslElemAttributeSet:
			as, err := c.compileOverrideAttributeSet(elem, pkg)
			if err != nil {
				return err
			}
			oset.attributeSets[as.Name] = as

		default:
			return staticError(errCodeXTSE0010,
				"xsl:override may only contain template, function, variable, param, or attribute-set; got xsl:%s",
				elem.LocalName())
		}
	}

	return nil
}

// compileOverrideFunction compiles a function inside xsl:override.
func (c *compiler) compileOverrideFunction(elem *helium.Element, pkg *Stylesheet) (*xslFunction, xpath3.QualifiedName, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, xpath3.QualifiedName{}, staticError(errCodeXTSE0110, "xsl:function in xsl:override requires name attribute")
	}

	c.collectNamespaces(elem)

	var qn xpath3.QualifiedName
	if strings.HasPrefix(name, "Q{") {
		closeBrace := strings.IndexByte(name, '}')
		if closeBrace < 0 {
			return nil, xpath3.QualifiedName{}, staticError(errCodeXTSE0010, "malformed EQName in xsl:function name %q", name)
		}
		uri := name[2:closeBrace]
		local := name[closeBrace+1:]
		qn = xpath3.QualifiedName{URI: uri, Name: local}
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		uri := c.nsBindings[prefix]
		if uri == "" {
			uri = c.stylesheet.namespaces[prefix]
		}
		if uri == "" {
			return nil, xpath3.QualifiedName{}, staticError(errCodeXTSE0010, "unresolved namespace prefix %q in xsl:function name %q", prefix, name)
		}
		qn = xpath3.QualifiedName{URI: uri, Name: local}
	} else {
		return nil, xpath3.QualifiedName{}, staticError(errCodeXTSE0010, "xsl:function name %q must have a namespace prefix", name)
	}

	// Check that the function exists in the package (check by name, any arity)
	var pkgFn *xslFunction
	for fk, fn := range pkg.functions {
		if fk.Name == qn {
			pkgFn = fn
			break
		}
	}
	if pkgFn == nil {
		return nil, xpath3.QualifiedName{}, staticError(errCodeXTSE3058,
			"xsl:override function %q not found in used package", name)
	}

	// Check visibility: cannot override final
	if pkgFn != nil && pkgFn.Visibility == visFinal {
		return nil, xpath3.QualifiedName{}, staticError(errCodeXTSE3070,
			"cannot override final function %q", name)
	}

	// XTSE3070: new-each-time on the override must match the base component.
	// The base defaults to "maybe" (empty string) when not specified.
	if pkgFn != nil {
		overrideNET := getAttr(elem, "new-each-time")
		if overrideNET != "" {
			baseNET := pkgFn.NewEachTime
			if baseNET == "" {
				baseNET = "maybe" // default per spec
			}
			if overrideNET != baseNET {
				return nil, xpath3.QualifiedName{}, staticError(errCodeXTSE3070,
					"override function %q: new-each-time=%q does not match base component value %q",
					name, overrideNET, baseNET)
			}
		}
	}

	// Handle expand-text
	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		}
	}

	body, params, err := c.compileTemplateBody(elem)
	c.expandText = savedExpandText
	if err != nil {
		return nil, xpath3.QualifiedName{}, err
	}

	fn := &xslFunction{
		Name:       qn,
		Params:     params,
		Body:       body,
		As:         getAttr(elem, "as"),
		Visibility: getAttr(elem, "visibility"),
		IsOverride: true,
	}

	// Inherit the base component's visibility so that processExpose
	// in the using package does not default it to private. Abstract
	// becomes public since the override provides an implementation.
	if pkgFn != nil && pkgFn.Visibility != "" {
		if pkgFn.Visibility == visAbstract {
			fn.Visibility = visPublic
		} else {
			fn.Visibility = pkgFn.Visibility
		}
	}

	// Link the original function for xsl:original() calls.
	// Try exact arity match first, then fall back to any match by name.
	exactKey := funcKey{Name: qn, Arity: len(params)}
	if orig, ok := pkg.functions[exactKey]; ok {
		fn.OriginalFunc = orig
	} else if pkgFn != nil {
		fn.OriginalFunc = pkgFn
	}

	return fn, qn, nil
}

// compileOverrideTemplate compiles a template inside xsl:override.
func (c *compiler) compileOverrideTemplate(elem *helium.Element, pkg *Stylesheet) (*template, error) {
	tmpl := &template{
		ImportPrec:    c.importPrec,
		MinImportPrec: c.minImportPrec,
		BaseURI:       c.baseURI,
	}

	c.collectNamespaces(elem)

	savedXPathDefaultNS := c.xpathDefaultNS
	if xdn := getAttr(elem, "xpath-default-namespace"); xdn != "" {
		c.xpathDefaultNS = xdn
	}
	tmpl.XPathDefaultNS = c.xpathDefaultNS
	defer func() { c.xpathDefaultNS = savedXPathDefaultNS }()

	matchAttr := getAttr(elem, "match")
	if matchAttr != "" {
		p, err := compilePattern(matchAttr, c.nsBindings, c.xpathDefaultNS)
		if err != nil {
			return nil, err
		}
		tmpl.Match = p
	}

	tmpl.Name = resolveQName(getAttr(elem, "name"), c.nsBindings)
	modeAttr := getAttr(elem, "mode")
	if modeAttr != "" {
		tmpl.Mode = c.resolveMode(modeAttr)
	} else if tmpl.Match != nil && c.defaultMode != "" {
		// When no explicit mode is specified and a default-mode is active,
		// match templates use the default mode.
		tmpl.Mode = c.defaultMode
	}

	if prio := getAttr(elem, "priority"); prio != "" {
		f, err := parseFloat(prio)
		if err != nil {
			return nil, staticError(errCodeXTSE0010, "invalid priority %q: %v", prio, err)
		}
		tmpl.Priority = f
	} else if tmpl.Match != nil && len(tmpl.Match.Alternatives) == 1 {
		tmpl.Priority = tmpl.Match.Alternatives[0].priority
	}

	// XTSE3440: override match template can only use modes that are public
	// or abstract in the used package. Using a mode that doesn't exist in the
	// package, or a private/final/hidden mode, is a static error.
	if tmpl.Match != nil && tmpl.Mode != modeAll {
		modes := strings.Fields(tmpl.Mode)
		if len(modes) == 0 {
			// No explicit mode: check unnamed mode
			modes = []string{""}
		}
		for _, m := range modes {
			// Resolve special names to empty string for lookup
			if m == "#unnamed" || m == "#default" {
				m = ""
			}
			if err := checkOverrideModeVisibility(m, tmpl.Mode, pkg); err != nil {
				return nil, err
			}
		}
	}

	// Validate: named template must exist in package
	if tmpl.Name != "" {
		existing, exists := pkg.namedTemplates[tmpl.Name]
		if !exists {
			return nil, staticError(errCodeXTSE3058,
				"xsl:override template %q not found in used package", tmpl.Name)
		}
		// Check visibility: cannot override final template
		if existing.Visibility == visFinal {
			return nil, staticError(errCodeXTSE3070,
				"cannot override final template %q", tmpl.Name)
		}
		// Check visibility: cannot override private/hidden template
		pkgVis := getComponentVisibility(pkg, xslElemTemplate, tmpl.Name)
		if pkgVis == visPrivate || pkgVis == visHidden {
			return nil, staticError(errCodeXTSE3050,
				"cannot override %s template %q", pkgVis, tmpl.Name)
		}
	}

	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		}
	}

	ctxDecl, body, params, err := c.compileTemplateBodyEx(elem, false)
	c.expandText = savedExpandText
	if err != nil {
		return nil, err
	}
	tmpl.Params = params
	tmpl.Body = body
	tmpl.As = getAttr(elem, "as")
	if vis := getAttr(elem, "visibility"); vis != "" {
		tmpl.Visibility = vis
	}
	if ctxDecl != nil {
		tmpl.ContextItemAs = ctxDecl.as
		tmpl.ContextItemUse = ctxDecl.use
	}

	// XTSE3070: check type compatibility of override against base template.
	if tmpl.Name != "" {
		if existing, ok := pkg.namedTemplates[tmpl.Name]; ok {
			if err := checkOverrideTemplateCompat(tmpl, existing); err != nil {
				return nil, err
			}
			// Link the original template for xsl:original calls
			tmpl.OriginalTemplate = existing
			// Inherit the base component's visibility so that processExpose
			// in the using package does not default it to private. Abstract
			// becomes public since the override provides an implementation.
			if existing.Visibility != "" {
				if existing.Visibility == visAbstract {
					tmpl.Visibility = visPublic
				} else {
					tmpl.Visibility = existing.Visibility
				}
			}
		}
	}

	return tmpl, nil
}

// compileOverrideVariable compiles a variable inside xsl:override.
func (c *compiler) compileOverrideVariable(elem *helium.Element, pkg *Stylesheet) (*variable, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:variable in xsl:override requires name attribute")
	}
	resolvedName := resolveQName(name, c.nsBindings)

	// Check that the variable or param exists in the package.
	// In XSLT 3.0, variables and parameters share the same component
	// namespace, so a variable can override a param (and vice versa).
	var pkgVar *variable
	for _, v := range pkg.globalVars {
		if v.Name == resolvedName {
			pkgVar = v
			break
		}
	}
	if pkgVar == nil {
		// Also check params — they share the same namespace.
		found := false
		for _, p := range pkg.globalParams {
			if p.Name == resolvedName {
				found = true
				break
			}
		}
		if !found {
			return nil, staticError(errCodeXTSE3058,
				"xsl:override variable %q not found in used package", name)
		}
	}

	// Check visibility: cannot override final/private/hidden variable.
	// Effective visibility comes from xsl:expose (variableVisibility map)
	// or the declared visibility attribute on the element.
	effectiveVis := ""
	if pkgVar != nil {
		effectiveVis = pkgVar.Visibility
	}
	if v, ok := pkg.variableVisibility[resolvedName]; ok {
		effectiveVis = v
	}
	if effectiveVis == "" {
		if v, ok := pkg.globalParamVisibility[resolvedName]; ok {
			effectiveVis = v
		}
	}
	switch effectiveVis {
	case visFinal:
		return nil, staticError(errCodeXTSE3060,
			"cannot override final variable %q", name)
	case visPrivate, visHidden:
		return nil, staticError(errCodeXTSE3060,
			"cannot override %s variable %q", effectiveVis, name)
	}

	v := &variable{Name: resolvedName, As: getAttr(elem, "as"), Visibility: getAttr(elem, "visibility")}

	// XTSE3070: the required type of the override must match the base.
	// Only check when both types are standard XSD types (xs: prefix or
	// built-in names). Custom schema types from xsl:import-schema cannot
	// be reliably compared by name alone.
	if pkgVar != nil && v.As != "" && pkgVar.As != "" && v.As != pkgVar.As {
		if isStandardType(v.As) && isStandardType(pkgVar.As) {
			return nil, staticError(errCodeXTSE3070,
				"override variable %q type %q does not match base type %q",
				name, v.As, pkgVar.As)
		}
	}

	// Link the original variable for $xsl:original references.
	// Inherit the base component's visibility so that processExpose
	// in the using package does not default it to private. Abstract
	// becomes public since the override provides an implementation.
	if pkgVar != nil {
		v.OriginalVar = pkgVar
		if pkgVar.Visibility != "" {
			if pkgVar.Visibility == visAbstract {
				v.Visibility = visPublic
			} else {
				v.Visibility = pkgVar.Visibility
			}
		}
	}

	// The override's own visibility attribute takes precedence over the
	// inherited base visibility. This allows e.g. visibility="private"
	// on the override to hide the variable from the using package.
	if overrideVis := getAttr(elem, "visibility"); overrideVis != "" {
		v.Visibility = overrideVis
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		v.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		v.Body = body
	}

	return v, nil
}

// compileOverrideParam compiles a param inside xsl:override.
func (c *compiler) compileOverrideParam(elem *helium.Element, pkg *Stylesheet) (*param, error) {
	p, err := c.compileParamDef(elem)
	if err != nil {
		return nil, err
	}

	// Check that the param or variable exists in the package.
	// In XSLT 3.0, variables and parameters share the same component
	// namespace, so a param can override a variable (and vice versa).
	found := false
	for _, pp := range pkg.globalParams {
		if pp.Name == p.Name {
			found = true
			break
		}
	}
	if !found {
		for _, v := range pkg.globalVars {
			if v.Name == p.Name {
				found = true
				break
			}
		}
	}
	if !found {
		return nil, staticError(errCodeXTSE3058,
			"xsl:override param %q not found in used package", p.Name)
	}

	return p, nil
}

// compileOverrideAttributeSet compiles an attribute-set inside xsl:override.
func (c *compiler) compileOverrideAttributeSet(elem *helium.Element, pkg *Stylesheet) (*attributeSetDef, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:attribute-set in xsl:override requires name attribute")
	}
	resolvedName := resolveQName(name, c.nsBindings)

	// Check that the attribute-set exists in the package
	if pkg.attributeSets == nil {
		return nil, staticError(errCodeXTSE3058,
			"xsl:override attribute-set %q not found in used package", name)
	}
	pkgAS, exists := pkg.attributeSets[resolvedName]
	if !exists {
		return nil, staticError(errCodeXTSE3058,
			"xsl:override attribute-set %q not found in used package", name)
	}

	// Check visibility: cannot override final/private/hidden attribute-set.
	// Effective visibility comes from xsl:expose (attrSetVisibility map) or
	// the declared visibility attribute on the xsl:attribute-set element.
	effectiveVis := ""
	if pkgAS != nil {
		effectiveVis = pkgAS.Visibility
	}
	if v, ok := pkg.attrSetVisibility[resolvedName]; ok {
		effectiveVis = v
	}
	switch effectiveVis {
	case visFinal:
		return nil, staticError(errCodeXTSE3060,
			"cannot override final attribute-set %q", name)
	case visPrivate, visHidden:
		return nil, staticError(errCodeXTSE3060,
			"cannot override %s attribute-set %q", effectiveVis, name)
	}

	asd := &attributeSetDef{Name: resolvedName}

	var useAttrSets []string
	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		for _, n := range strings.Fields(uas) {
			resolved := resolveQName(n, c.nsBindings)
			asd.UseAttrSets = append(asd.UseAttrSets, resolved)
			useAttrSets = append(useAttrSets, resolved)
		}
	}

	var attrs []instruction
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == lexicon.NamespaceXSLT && childElem.LocalName() == lexicon.XSLTElementAttribute {
			inst, err := c.compileAttribute(childElem)
			if err != nil {
				return nil, err
			}
			asd.Attrs = append(asd.Attrs, inst)
			attrs = append(attrs, inst)
		}
	}
	asd.Parts = []attributeSetPart{{UseAttrSets: useAttrSets, Attrs: attrs}}

	return asd, nil
}

// checkOverrideTemplateCompat validates that an override template is compatible
// with the base template it replaces. XTSE3070 is raised when parameter types
// differ or new required parameters are added.
func checkOverrideTemplateCompat(override, base *template) error {
	// XTSE3070: check return type compatibility.
	// If both templates have 'as' attributes, they must be compatible
	// (the override's return type must be a subtype of the base's).
	if base.As != "" && override.As != "" && base.As != override.As {
		return staticError(errCodeXTSE3070,
			"override template %q return type %q does not match base type %q",
			override.Name, override.As, base.As)
	}

	// XTSE3070: check context-item compatibility.
	// The override must have the same context-item type and use as the base.
	if override.ContextItemAs != "" && override.ContextItemAs != base.ContextItemAs {
		return staticError(errCodeXTSE3070,
			"override template %q context-item type %q does not match base type %q",
			override.Name, override.ContextItemAs, base.ContextItemAs)
	}
	if override.ContextItemUse != "" && override.ContextItemUse != base.ContextItemUse {
		return staticError(errCodeXTSE3070,
			"override template %q context-item use %q does not match base use %q",
			override.Name, override.ContextItemUse, base.ContextItemUse)
	}

	// Build map of base params by name
	baseParams := make(map[string]*param)
	for _, p := range base.Params {
		baseParams[p.Name] = p
	}

	for _, op := range override.Params {
		bp, exists := baseParams[op.Name]
		if !exists {
			// New parameter in override that doesn't exist in base
			if op.Required {
				return staticError(errCodeXTSE3070,
					"override template %q adds required parameter $%s not in base", override.Name, op.Name)
			}
			continue
		}
		// Check type compatibility: if both have 'as', they must match
		if bp.As != "" && op.As != "" && bp.As != op.As {
			return staticError(errCodeXTSE3070,
				"override template %q parameter $%s type %q does not match base type %q",
				override.Name, op.Name, op.As, bp.As)
		}
	}
	return nil
}

// checkOverrideModeVisibility validates that a mode used by an override
// template is public/abstract in the used package.
func checkOverrideModeVisibility(mode, displayMode string, pkg *Stylesheet) error {
	if pkg.modeDefs != nil {
		if md, ok := pkg.modeDefs[mode]; ok {
			if md.Visibility == visFinal {
				return staticError(errCodeXTSE3060,
					"cannot override templates in final mode %q", displayMode)
			}
			if md.Visibility == visPrivate || md.Visibility == visHidden {
				return staticError(errCodeXTSE3440,
					"cannot override templates in %s mode %q", md.Visibility, displayMode)
			}
			return nil
		}
	}
	// Mode not defined in the package — check default visibility
	defVis := defaultComponentVisibility(pkg)
	if defVis == visPrivate {
		return staticError(errCodeXTSE3440,
			"mode %q is not defined in the used package (default visibility is private)", displayMode)
	}
	return nil
}

// isStandardType returns true if the type name is a well-known XSD type
// (xs: prefix) or standard type keyword. Returns false for custom schema
// types defined via xsl:import-schema.
func isStandardType(as string) bool {
	// Strip occurrence indicator
	name := strings.TrimRight(as, "?*+")
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "xs:") {
		return true
	}
	// Standard keywords
	switch name {
	case "item()", "node()", "element()", "attribute()", "text()",
		"comment()", "processing-instruction()", "document-node()",
		"namespace-node()", "schema-element()", "schema-attribute()",
		"function(*)", "map(*)", "array(*)":
		return true
	}
	return false
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
