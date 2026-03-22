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
	functions      map[funcKey]*XSLFunction
	namedTemplates map[string]*Template
	matchTemplates []*Template
	variables      map[string]*Variable
	params         map[string]*Param
	attributeSets  map[string]*AttributeSetDef
}

// processOverrides handles xsl:override children of xsl:use-package.
// It compiles the override children in the context of the package components,
// validates they match existing package components, and returns the override set.
func (c *compiler) processOverrides(usePackageElem *helium.Element, pkg *Stylesheet) (*overrideSet, error) {
	oset := &overrideSet{
		functions:      make(map[funcKey]*XSLFunction),
		namedTemplates: make(map[string]*Template),
		variables:      make(map[string]*Variable),
		params:         make(map[string]*Param),
		attributeSets:  make(map[string]*AttributeSetDef),
	}

	for child := usePackageElem.FirstChild(); child != nil; child = child.NextSibling() {
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

	for child := overrideElem.FirstChild(); child != nil; child = child.NextSibling() {
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
func (c *compiler) compileOverrideFunction(elem *helium.Element, pkg *Stylesheet) (*XSLFunction, xpath3.QualifiedName, error) {
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
	var pkgFn *XSLFunction
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

	fn := &XSLFunction{
		Name:   qn,
		Params: params,
		Body:   body,
		As:     getAttr(elem, "as"),
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
func (c *compiler) compileOverrideTemplate(elem *helium.Element, pkg *Stylesheet) (*Template, error) {
	tmpl := &Template{
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

	// Validate: override match template can only use public/abstract modes
	if tmpl.Mode != "" && pkg.modeDefs != nil {
		if md, ok := pkg.modeDefs[tmpl.Mode]; ok {
			if md.Visibility == visFinal {
				return nil, staticError(errCodeXTSE3060,
					"cannot override templates in final mode %q", tmpl.Mode)
			}
			if md.Visibility == visPrivate || md.Visibility == visHidden {
				return nil, staticError(errCodeXTSE3060,
					"cannot override templates in %s mode %q", md.Visibility, tmpl.Mode)
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

	body, params, err := c.compileTemplateBody(elem)
	c.expandText = savedExpandText
	if err != nil {
		return nil, err
	}
	tmpl.Params = params
	tmpl.Body = body
	tmpl.As = getAttr(elem, "as")

	// XTSE3070: check type compatibility of override against base template.
	if tmpl.Name != "" {
		if existing, ok := pkg.namedTemplates[tmpl.Name]; ok {
			if err := checkOverrideTemplateCompat(tmpl, existing); err != nil {
				return nil, err
			}
			// Link the original template for xsl:original calls
			tmpl.OriginalTemplate = existing
		}
	}

	return tmpl, nil
}

// compileOverrideVariable compiles a variable inside xsl:override.
func (c *compiler) compileOverrideVariable(elem *helium.Element, pkg *Stylesheet) (*Variable, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:variable in xsl:override requires name attribute")
	}
	resolvedName := resolveQName(name, c.nsBindings)

	// Check that the variable or param exists in the package.
	// In XSLT 3.0, variables and parameters share the same component
	// namespace, so a variable can override a param (and vice versa).
	var pkgVar *Variable
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

	v := &Variable{Name: resolvedName, As: getAttr(elem, "as")}

	// Link the original variable for $xsl:original references
	if pkgVar != nil {
		v.OriginalVar = pkgVar
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
func (c *compiler) compileOverrideParam(elem *helium.Element, pkg *Stylesheet) (*Param, error) {
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
func (c *compiler) compileOverrideAttributeSet(elem *helium.Element, pkg *Stylesheet) (*AttributeSetDef, error) {
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

	asd := &AttributeSetDef{Name: resolvedName}

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		for _, n := range strings.Fields(uas) {
			asd.UseAttrSets = append(asd.UseAttrSets, resolveQName(n, c.nsBindings))
		}
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.URI() == lexicon.NamespaceXSLT && childElem.LocalName() == "attribute" {
			inst, err := c.compileAttribute(childElem)
			if err != nil {
				return nil, err
			}
			asd.Attrs = append(asd.Attrs, inst)
		}
	}

	return asd, nil
}

// checkOverrideTemplateCompat validates that an override template is compatible
// with the base template it replaces. XTSE3070 is raised when parameter types
// differ or new required parameters are added.
func checkOverrideTemplateCompat(override, base *Template) error {
	// XTSE3070: check return type compatibility.
	// If both templates have 'as' attributes, they must be compatible
	// (the override's return type must be a subtype of the base's).
	if base.As != "" && override.As != "" && base.As != override.As {
		return staticError(errCodeXTSE3070,
			"override template %q return type %q does not match base type %q",
			override.Name, override.As, base.As)
	}

	// Build map of base params by name
	baseParams := make(map[string]*Param)
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

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
