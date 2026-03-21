package xslt3

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
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
		if !ok || elem.URI() != NSXSLT || elem.LocalName() != xslElemOverride {
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
		if elem.URI() != NSXSLT {
			return staticError(errCodeXTSE0010,
				"non-XSLT element %q is not allowed inside xsl:override", elem.Name())
		}

		switch elem.LocalName() {
		case xslElemFunction:
			fn, qn, err := c.compileOverrideFunction(elem, pkg)
			if err != nil {
				return err
			}
			oset.functions[funcKey{Name: qn, Arity: len(fn.Params)}] = fn

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

	// Validate: named template must exist in package
	if tmpl.Name != "" {
		if _, exists := pkg.namedTemplates[tmpl.Name]; !exists {
			return nil, staticError(errCodeXTSE3058,
				"xsl:override template %q not found in used package", tmpl.Name)
		}
		// Check visibility: cannot override final template
		if existing := pkg.namedTemplates[tmpl.Name]; existing != nil && existing.Visibility == visFinal {
			return nil, staticError(errCodeXTSE3070,
				"cannot override final template %q", tmpl.Name)
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

	return tmpl, nil
}

// compileOverrideVariable compiles a variable inside xsl:override.
func (c *compiler) compileOverrideVariable(elem *helium.Element, pkg *Stylesheet) (*Variable, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:variable in xsl:override requires name attribute")
	}
	resolvedName := resolveQName(name, c.nsBindings)

	// Check that the variable exists in the package
	found := false
	for _, v := range pkg.globalVars {
		if v.Name == resolvedName {
			found = true
			break
		}
	}
	if !found {
		return nil, staticError(errCodeXTSE3058,
			"xsl:override variable %q not found in used package", name)
	}

	v := &Variable{Name: resolvedName, As: getAttr(elem, "as")}

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

	// Check that the param exists in the package
	found := false
	for _, pp := range pkg.globalParams {
		if pp.Name == p.Name {
			found = true
			break
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
	if _, exists := pkg.attributeSets[resolvedName]; !exists {
		return nil, staticError(errCodeXTSE3058,
			"xsl:override attribute-set %q not found in used package", name)
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
		if childElem.URI() == NSXSLT && childElem.LocalName() == "attribute" {
			inst, err := c.compileAttribute(childElem)
			if err != nil {
				return nil, err
			}
			asd.Attrs = append(asd.Attrs, inst)
		}
	}

	return asd, nil
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
