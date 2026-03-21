package xslt3

import (
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
)

func (c *compiler) compileValueOf(elem *helium.Element) (*ValueOfInst, error) {
	inst := &ValueOfInst{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	if sep, ok := elem.GetAttribute("separator"); ok {
		avt, err := compileAVT(sep, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Separator = avt
		inst.HasSeparator = true
	}

	// XTSE0350: xsl:value-of with select must not have non-whitespace content
	if selectAttr != "" {
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			switch child.Type() {
			case helium.TextNode, helium.CDATASectionNode:
				if strings.TrimSpace(string(child.Content())) != "" {
					return nil, staticError(errCodeXTSE0350, "xsl:value-of with select attribute must not have content")
				}
			case helium.ElementNode:
				return nil, staticError(errCodeXTSE0350, "xsl:value-of with select attribute must not have content")
			}
		}
	} else {
		// Sequence constructor body (XSLT 2.0+)
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		// XSLT 3.0: empty xsl:value-of is allowed (produces zero-length string).
		// We are an XSLT 3.0 processor, so XTSE0870 is never raised.
		inst.Body = body
	}

	inst.DisableOutputEscaping = getAttr(elem, "disable-output-escaping") == "yes"

	return inst, nil
}

func (c *compiler) compileText(elem *helium.Element) (*TextInst, error) {
	// xsl:text must contain only text and CDATA sections — no child elements (XTSE0010)
	var sb strings.Builder
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			sb.Write(child.Content())
		case helium.ElementNode:
			return nil, staticError(errCodeXTSE0010, "xsl:text must not contain child elements")
		}
	}

	text := sb.String()
	inst := &TextInst{
		Value:                 text,
		DisableOutputEscaping: getAttr(elem, "disable-output-escaping") == "yes",
	}

	if c.expandText && strings.ContainsAny(text, "{}") {
		avt, err := compileAVT(text, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.TVT = avt
	}

	return inst, nil
}

func (c *compiler) compileElement(elem *helium.Element) (*ElementInst, error) {
	// xsl:break / xsl:next-iteration not allowed inside element constructors.
	savedBreak := c.breakAllowed
	c.breakAllowed = false
	defer func() { c.breakAllowed = savedBreak }()

	// Validate attributes
	if err := validateXSLTAttrs(elem, map[string]struct{}{
		"name": {}, "namespace": {}, "inherit-namespaces": {},
		"use-attribute-sets": {}, "type": {}, "validation": {},
	}); err != nil {
		return nil, err
	}

	// Validate boolean attributes (including empty string)
	if inAttr, hasIn := elem.GetAttribute("inherit-namespaces"); hasIn {
		if err := validateBooleanAttr("xsl:element", "inherit-namespaces", inAttr); err != nil {
			return nil, err
		}
	}

	nameAttr := getAttr(elem, "name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:element requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &ElementInst{Name: nameAVT, InheritNamespaces: true}

	if inAttr := getAttr(elem, "inherit-namespaces"); inAttr == "no" {
		inst.InheritNamespaces = false
	}

	// Capture compile-time namespace bindings for runtime name resolution.
	if len(c.nsBindings) > 0 {
		inst.NSBindings = make(map[string]string, len(c.nsBindings))
		for k, v := range c.nsBindings {
			inst.NSBindings[k] = v
		}
	}

	typeAttr := getAttr(elem, "type")
	validation := getAttr(elem, "validation")

	if err := checkValidationTypeExclusive("xsl:element", validation, typeAttr); err != nil {
		return nil, err
	}
	if typeAttr != "" {
		if err := c.checkTypeAttrSchemaAware("xsl:element", typeAttr); err != nil {
			return nil, err
		}
		inst.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
	}
	if validation != "" {
		if err := validateValidationAttr("xsl:element", validation); err != nil {
			return nil, err
		}
		inst.Validation = validation
	}

	if nsAttr, hasNS := elem.GetAttribute("namespace"); hasNS {
		nsAVT, err := compileAVT(nsAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Namespace = nsAVT
	}

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		for _, name := range strings.Fields(uas) {
			inst.UseAttributeSets = append(inst.UseAttributeSets, resolveQName(name, c.nsBindings))
		}
	}

	// Capture xml:base for static base URI override during body execution.
	effectiveBase := stylesheetBaseURI(elem, c.baseURI)
	if effectiveBase != c.baseURI {
		inst.StaticBaseURI = effectiveBase
	}

	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		inst.UseAttrSets = strings.Fields(uas)
	}

	return inst, nil
}

func (c *compiler) compileAttribute(elem *helium.Element) (*AttributeInst, error) {
	nameAttr := getAttr(elem, "name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:attribute requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &AttributeInst{Name: nameAVT}

	if typeAttr := getAttr(elem, "type"); typeAttr != "" {
		if err := c.checkTypeAttrSchemaAware("xsl:attribute", typeAttr); err != nil {
			return nil, err
		}
		inst.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
	}

	if valAttr := getAttr(elem, "validation"); valAttr != "" {
		if err := validateValidationAttr("xsl:attribute", valAttr); err != nil {
			return nil, err
		}
		if !c.stylesheet.schemaAware && (valAttr == "strict" || valAttr == "lax") {
			return nil, staticError(errCodeXTSE0220, "validation attribute requires schema-aware processing")
		}
		inst.Validation = valAttr
	}

	if nsAttr, hasNS := elem.GetAttribute("namespace"); hasNS {
		nsAVT, err := compileAVT(nsAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Namespace = nsAVT
	}

	if sep, hasSep := elem.GetAttribute("separator"); hasSep {
		sepAVT, err := compileAVT(sep, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Separator = sepAVT
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileComment(elem *helium.Element) (*CommentInst, error) {
	inst := &CommentInst{}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		// XTSE0940: select and non-empty content are mutually exclusive.
		if err := c.validateEmptyElement(elem, "xsl:comment"); err != nil {
			return nil, staticError(errCodeXTSE0940, "xsl:comment has both @select and content")
		}
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compilePI(elem *helium.Element) (*PIInst, error) {
	nameAttr := getAttr(elem, "name")
	if nameAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:processing-instruction requires name attribute")
	}

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &PIInst{Name: nameAVT}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		// XTSE0940: select and non-empty content are mutually exclusive.
		if err := c.validateEmptyElement(elem, "xsl:processing-instruction"); err != nil {
			return nil, staticError(errCodeXTSE0940, "xsl:processing-instruction has both @select and content")
		}
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileCopy(elem *helium.Element) (*CopyInst, error) {
	// Validate boolean attributes (including empty string)
	if inAttr, hasIn := elem.GetAttribute("inherit-namespaces"); hasIn {
		if err := validateBooleanAttr("xsl:copy", "inherit-namespaces", inAttr); err != nil {
			return nil, err
		}
	}
	if cn, hasCN := elem.GetAttribute("copy-namespaces"); hasCN {
		if err := validateBooleanAttr("xsl:copy", "copy-namespaces", cn); err != nil {
			return nil, err
		}
	}

	inst := &CopyInst{
		CopyNamespaces:    true,
		InheritNamespaces: true,
	}

	if cn := getAttr(elem, "copy-namespaces"); cn != "" {
		if v, ok := parseXSDBool(cn); ok && !v {
			inst.CopyNamespaces = false
		}
	}
	if in := getAttr(elem, "inherit-namespaces"); in != "" {
		if v, ok := parseXSDBool(in); ok && !v {
			inst.InheritNamespaces = false
		}
	}

	// Shadow attributes (_attr overrides attr via AVT at runtime)
	if scn := getAttr(elem, "_copy-namespaces"); scn != "" {
		avt, err := compileAVT(scn, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.CopyNamespacesAVT = avt
	}
	if sin := getAttr(elem, "_inherit-namespaces"); sin != "" {
		avt, err := compileAVT(sin, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.InheritNamespacesAVT = avt
	}

	if v := getAttr(elem, "validation"); v != "" {
		if err := validateValidationAttr("xsl:copy", v); err != nil {
			return nil, err
		}
		inst.Validation = v
	}
	if typeAttr := getAttr(elem, "type"); typeAttr != "" {
		inst.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
	}

	if selectAttr := getAttr(elem, "select"); selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		for _, name := range strings.Fields(uas) {
			inst.UseAttributeSets = append(inst.UseAttributeSets, resolveQName(name, c.nsBindings))
		}
	}

	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body

	if uas := getAttr(elem, "use-attribute-sets"); uas != "" {
		inst.UseAttrSets = strings.Fields(uas)
	}

	return inst, nil
}

func (c *compiler) compileCopyOf(elem *helium.Element) (*CopyOfInst, error) {
	// XTSE0260: xsl:copy-of must have no significant content
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.ElementNode:
			return nil, staticError(errCodeXTSE0260,
				"xsl:copy-of must be empty, but contains child element %s", child.Name())
		case helium.TextNode, helium.CDATASectionNode:
			if strings.TrimSpace(string(child.Content())) != "" {
				return nil, staticError(errCodeXTSE0260,
					"xsl:copy-of must be empty, but contains text content")
			}
		}
	}

	// XTSE0090: reject specific attributes that are not allowed on xsl:copy-of
	for _, attr := range elem.Attributes() {
		if attr.URI() != "" {
			continue
		}
		if attr.LocalName() == "match" || attr.LocalName() == "count" || attr.LocalName() == "from" {
			return nil, staticError(errCodeXTSE0090,
				"attribute %q is not allowed on xsl:copy-of", attr.LocalName())
		}
	}

	// Validate boolean attribute: copy-namespaces
	if cn, hasCN := elem.GetAttribute("copy-namespaces"); hasCN {
		if err := validateBooleanAttr("xsl:copy-of", "copy-namespaces", cn); err != nil {
			return nil, err
		}
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:copy-of requires select attribute")
	}

	expr, err := compileXPath(selectAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &CopyOfInst{
		Select:         expr,
		CopyNamespaces: true,
	}
	if cn := getAttr(elem, "copy-namespaces"); cn != "" {
		if v, ok := parseXSDBool(cn); ok && !v {
			inst.CopyNamespaces = false
		}
	}
	// Shadow attribute _copy-namespaces overrides at runtime
	if scn := getAttr(elem, "_copy-namespaces"); scn != "" {
		avt, err := compileAVT(scn, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.CopyNamespacesAVT = avt
	}
	if v := getAttr(elem, "validation"); v != "" {
		if err := validateValidationAttr("xsl:copy-of", v); err != nil {
			return nil, err
		}
		inst.Validation = v
	}
	if typeAttr := getAttr(elem, "type"); typeAttr != "" {
		inst.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
	}
	return inst, nil
}

func (c *compiler) compileNumber(elem *helium.Element) (*NumberInst, error) {
	inst := &NumberInst{
		Level: getAttr(elem, "level"),
	}
	if inst.Level == "" {
		inst.Level = "single"
	}

	if countAttr := getAttr(elem, "count"); countAttr != "" {
		p, err := compilePattern(countAttr, c.nsBindings, c.xpathDefaultNS)
		if err != nil {
			return nil, err
		}
		inst.Count = p
	}

	if fromAttr := getAttr(elem, "from"); fromAttr != "" {
		p, err := compilePattern(fromAttr, c.nsBindings, c.xpathDefaultNS)
		if err != nil {
			return nil, err
		}
		inst.From = p
	}

	if valueAttr := getAttr(elem, "value"); valueAttr != "" {
		// XTSE0975: when value is present, select/level/count/from must be absent.
		_, hasLevel := elem.GetAttribute("level")
		if getAttr(elem, "select") != "" || getAttr(elem, "count") != "" ||
			getAttr(elem, "from") != "" || hasLevel {
			return nil, staticError(errCodeXTSE0975, "xsl:number: when @value is present, @select, @level, @count, and @from must be absent")
		}
		expr, err := compileXPath(valueAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Value = expr
	}

	if fmtAttr := getAttr(elem, "format"); fmtAttr != "" {
		avt, err := compileAVT(fmtAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Format = avt
	}

	if gs := getAttr(elem, "grouping-separator"); gs != "" {
		avt, err := compileAVT(gs, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.GroupingSeparator = avt
	}

	if gsz := getAttr(elem, "grouping-size"); gsz != "" {
		avt, err := compileAVT(gsz, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.GroupingSize = avt
	}

	if ord := getAttr(elem, "ordinal"); ord != "" {
		avt, err := compileAVT(ord, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Ordinal = avt
	}

	if sa := getAttr(elem, "start-at"); sa != "" {
		avt, err := compileAVT(sa, c.nsBindings)
		if err != nil {
			return nil, err
		}
		// Validate static start-at: must be a space-separated list of integers
		if !strings.Contains(sa, "{") {
			for _, part := range strings.Fields(sa) {
				if _, err := strconv.Atoi(part); err != nil {
					return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:number/@start-at", sa)
				}
			}
		}
		inst.StartAt = avt
	}

	if sel := getAttr(elem, "select"); sel != "" {
		expr, err := compileXPath(sel, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	}

	if lang := getAttr(elem, "lang"); lang != "" {
		avt, err := compileAVT(lang, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Lang = avt
	}

	if lv := getAttr(elem, "letter-value"); lv != "" {
		avt, err := compileAVT(lv, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.LetterValue = avt
	}

	return inst, nil
}

func (c *compiler) compileNamespace(elem *helium.Element) (*NamespaceInst, error) {
	nameAttr, hasName := elem.GetAttribute("name")
	if !hasName {
		return nil, staticError(errCodeXTSE0110, "xsl:namespace requires name attribute")
	}
	// name="" is valid: it sets the default namespace

	nameAVT, err := compileAVT(nameAttr, c.nsBindings)
	if err != nil {
		return nil, err
	}

	inst := &NamespaceInst{Name: nameAVT}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		inst.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		inst.Body = body
	}

	return inst, nil
}

func (c *compiler) compileDocument(elem *helium.Element) (*DocumentInst, error) {
	inst := &DocumentInst{}
	if v := getAttr(elem, "validation"); v != "" {
		if err := validateValidationAttr("xsl:document", v); err != nil {
			return nil, err
		}
		inst.Validation = v
	}
	if typeAttr := getAttr(elem, "type"); typeAttr != "" {
		inst.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
	}
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	inst.Body = body
	return inst, nil
}

func (c *compiler) compileSequence(elem *helium.Element) (Instruction, error) {
	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		return &XSLSequenceInst{Select: expr}, nil
	}
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	return &SequenceInst{Body: body}, nil
}

func (c *compiler) compileLiteralResultElement(elem *helium.Element) (*LiteralResultElement, error) {
	// xsl:break / xsl:next-iteration not allowed inside element constructors.
	savedBreak := c.breakAllowed
	c.breakAllowed = false
	defer func() { c.breakAllowed = savedBreak }()

	lre := &LiteralResultElement{
		Name:       elem.Name(),
		Namespace:  elem.URI(),
		Prefix:     elem.Prefix(),
		LocalName:  elem.LocalName(),
		Namespaces: make(map[string]string),
	}

	// Collect element-level xsl:exclude-result-prefixes and
	// xsl:extension-element-prefixes (cumulative with parent)
	savedExcludes := c.localExcludes
	needNewExcludes := false
	if _, ok := elem.GetAttributeNS("exclude-result-prefixes", NSXSLT); ok {
		needNewExcludes = true
	}
	if _, ok := elem.GetAttributeNS("extension-element-prefixes", NSXSLT); ok {
		needNewExcludes = true
	}
	if needNewExcludes {
		newExcludes := make(map[string]struct{})
		for k, v := range c.localExcludes {
			newExcludes[k] = v
		}
		if erp, ok := elem.GetAttributeNS("exclude-result-prefixes", NSXSLT); ok {
			if erp == "#all" {
				for prefix := range c.stylesheet.namespaces {
					newExcludes[prefix] = struct{}{}
				}
			} else {
				for _, prefix := range strings.Fields(erp) {
					newExcludes[prefix] = struct{}{}
				}
			}
		}
		if eep, ok := elem.GetAttributeNS("extension-element-prefixes", NSXSLT); ok {
			for _, prefix := range strings.Fields(eep) {
				newExcludes[prefix] = struct{}{}
			}
		}
		c.localExcludes = newExcludes
	}
	// Also register extension-element-prefixes URIs so that child elements
	// in those namespaces are recognised as extension elements (and their
	// xsl:fallback children are compiled).
	savedExtURIs := c.extensionURIs
	if eep, ok := elem.GetAttributeNS("extension-element-prefixes", NSXSLT); ok {
		newExtURIs := make(map[string]struct{})
		for k, v := range c.extensionURIs {
			newExtURIs[k] = v
		}
		for _, prefix := range strings.Fields(eep) {
			if uri, uriOK := c.nsBindings[prefix]; uriOK && uri != "" {
				newExtURIs[uri] = struct{}{}
			}
		}
		c.extensionURIs = newExtURIs
	}
	defer func() {
		c.localExcludes = savedExcludes
		c.extensionURIs = savedExtURIs
	}()

	// Build set of excluded namespace URIs. Stylesheet-level URIs were
	// resolved at compile init (before template processing mutates namespaces).
	// Local (element-level) URIs are resolved from c.nsBindings which has the
	// correct in-scope bindings at this point.
	excludedURIs := make(map[string]struct{}, len(c.stylesheet.excludeURIs))
	for uri := range c.stylesheet.excludeURIs {
		excludedURIs[uri] = struct{}{}
	}
	for prefix := range c.localExcludes {
		if uri, ok := c.nsBindings[prefix]; ok {
			excludedURIs[uri] = struct{}{}
		}
	}

	isExcluded := func(_, uri string) bool {
		_, ok := excludedURIs[uri]
		return ok
	}

	// Copy in-scope namespace declarations that are not excluded.
	// Use c.nsBindings (scoped to this element's position in the stylesheet)
	// rather than c.stylesheet.namespaces (which accumulates globally and
	// leaks namespaces from sibling variable trees).
	for prefix, uri := range c.nsBindings {
		if uri == NSXSLT || prefix == "" {
			continue
		}
		if isExcluded(prefix, uri) {
			continue
		}
		lre.Namespaces[prefix] = uri
	}
	// Override/add from directly declared namespaces on this element
	for _, ns := range elem.Namespaces() {
		uri := ns.URI()
		prefix := ns.Prefix()
		if uri == NSXSLT || prefix == "" {
			continue
		}
		if isExcluded(prefix, uri) {
			delete(lre.Namespaces, prefix)
			continue
		}
		lre.Namespaces[prefix] = uri
	}

	// Validate and compile attributes
	for _, attr := range elem.Attributes() {
		if attr.URI() == NSXSLT {
			// XTSE0805: only certain XSLT attributes are allowed on LREs
			if _, ok := lreAllowedXSLTAttrs[attr.LocalName()]; !ok {
				// In forwards-compatible mode (version > 3.0), unknown
				// XSLT attributes on LREs are silently ignored.
				if c.effectiveVersion > "3.0" {
					continue
				}
				return nil, staticError(errCodeXTSE0805,
					"attribute xsl:%s is not allowed on a literal result element", attr.LocalName())
			}
			continue
		}
		avt, err := compileAVT(attr.Value(), c.nsBindings)
		if err != nil {
			return nil, err
		}
		lre.Attrs = append(lre.Attrs, &LiteralAttribute{
			Name:      attr.Name(),
			Namespace: attr.URI(),
			Prefix:    attr.Prefix(),
			LocalName: attr.LocalName(),
			Value:     avt,
		})
	}

	// Apply namespace aliases (xsl:namespace-alias).
	// This must happen after namespace declarations and attributes are collected,
	// but before children are compiled.
	if len(c.stylesheet.namespaceAliases) > 0 {
		// Alias the element itself
		if resultURI, resultPfx, ok := c.resolveNamespaceAlias(lre.Namespace); ok {
			lre.Namespace = resultURI
			if resultPfx != "" {
				lre.Prefix = resultPfx
				lre.Name = resultPfx + ":" + lre.LocalName
			} else {
				lre.Prefix = ""
				lre.Name = lre.LocalName
			}
		}

		// Alias attributes
		for _, attr := range lre.Attrs {
			if attr.Namespace == "" {
				continue
			}
			if resultURI, resultPfx, ok := c.resolveNamespaceAlias(attr.Namespace); ok {
				attr.Namespace = resultURI
				if resultPfx != "" {
					attr.Prefix = resultPfx
					attr.Name = resultPfx + ":" + attr.LocalName
				} else {
					attr.Prefix = ""
					attr.Name = attr.LocalName
				}
			}
		}

		// Alias namespace declarations
		aliasedNS := make(map[string]string)
		for prefix, uri := range lre.Namespaces {
			if resultURI, resultPfx, ok := c.resolveNamespaceAlias(uri); ok {
				if resultPfx != "" {
					aliasedNS[resultPfx] = resultURI
				} else if resultURI != "" {
					// Only keep aliased prefix when the result URI is non-empty.
					// Mapping a prefixed namespace to "" (no namespace) means
					// the prefix is no longer needed — dropping it avoids
					// emitting the illegal xmlns:p="" undeclaration.
					aliasedNS[prefix] = resultURI
				}
				// else: prefixed namespace aliased to #default with no default
				// namespace → drop entirely (the element/attr already got the
				// correct no-namespace treatment above).
			} else {
				aliasedNS[prefix] = uri
			}
		}
		lre.Namespaces = aliasedNS
	}

	// Handle xsl:use-attribute-sets
	if uas, ok := elem.GetAttributeNS("use-attribute-sets", NSXSLT); ok {
		for _, name := range strings.Fields(uas) {
			lre.UseAttributeSets = append(lre.UseAttributeSets, resolveQName(name, c.nsBindings))
		}
		lre.UseAttrSets = strings.Fields(uas)
	}

	// Handle xsl:validation
	if valAttr, ok := elem.GetAttributeNS("validation", NSXSLT); ok {
		if err := validateValidationAttr("LRE (xsl:validation)", valAttr); err != nil {
			return nil, err
		}
		if !c.stylesheet.schemaAware && (valAttr == "strict" || valAttr == "lax") {
			return nil, staticError(errCodeXTSE0220, "xsl:validation requires schema-aware processing")
		}
		lre.Validation = valAttr
	}

	// Handle xsl:type
	if typeAttr, ok := elem.GetAttributeNS("type", NSXSLT); ok {
		if err := c.checkTypeAttrSchemaAware("LRE (xsl:type)", typeAttr); err != nil {
			return nil, err
		}
		lre.TypeName = resolveXSDTypeName(typeAttr, c.nsBindings)
	}

	// Compute effective static base URI: if any ancestor stylesheet element
	// (including this one) has xml:base, the static base URI for expressions
	// inside this LRE differs from the template/stylesheet base URI.
	effectiveBase := stylesheetBaseURI(elem, c.baseURI)
	if effectiveBase != c.baseURI {
		lre.StaticBaseURI = effectiveBase
	}

	// Compile children
	body, err := c.compileChildren(elem)
	if err != nil {
		return nil, err
	}
	lre.Body = body

	return lre, nil
}
