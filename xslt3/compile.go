package xslt3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// compiler holds state during stylesheet compilation.
type compiler struct {
	stylesheet     *Stylesheet
	nsBindings     map[string]string
	xpathDefaultNS string // current xpath-default-namespace
	preserveSpace  bool   // xml:space="preserve" in effect
	expandText     bool   // expand-text="yes" — text value templates enabled
	importPrec     int
	importStack    map[string]struct{} // circular import detection
	baseURI        string
	resolver       URIResolver
	localExcludes  map[string]struct{} // accumulated LRE-level exclude-result-prefixes
}

// shouldStripText returns true if a whitespace-only text node should be stripped
// during compilation (i.e., xml:space is not "preserve").
func (c *compiler) shouldStripText(text string) bool {
	if c.preserveSpace {
		return false
	}
	return strings.TrimSpace(text) == ""
}

// getAttr returns the value of an attribute or "" if not found.
func getAttr(elem *helium.Element, name string) string {
	v, _ := elem.GetAttribute(name)
	return v
}

// parseXSDBool parses an xs:boolean value ("yes"/"no", "true"/"false", "1"/"0")
// with whitespace trimming per the XSD specification.
func parseXSDBool(s string) (bool, bool) {
	switch strings.TrimSpace(s) {
	case "yes", "true", "1":
		return true, true
	case "no", "false", "0":
		return false, true
	default:
		return false, false
	}
}

// resolveQName resolves a QName (prefix:local or just local) to an expanded name.
// If the QName has a prefix, it is resolved using the given namespace bindings
// and the result is returned in Clark notation: {uri}local.
// If no prefix, the name is returned as-is.
func resolveQName(qname string, nsBindings map[string]string) string {
	qname = strings.TrimSpace(qname)
	// Handle EQName syntax: Q{uri}local
	if braceIdx := strings.IndexByte(qname, '{'); braceIdx >= 0 {
		closeIdx := strings.IndexByte(qname, '}')
		if closeIdx > braceIdx {
			uri := qname[braceIdx+1 : closeIdx]
			local := qname[closeIdx+1:]
			if uri == "" {
				return local // Q{}local → just local (no namespace)
			}
			return "{" + uri + "}" + local
		}
	}
	idx := strings.IndexByte(qname, ':')
	if idx < 0 {
		return qname
	}
	prefix := qname[:idx]
	local := qname[idx+1:]
	if uri, ok := nsBindings[prefix]; ok {
		return "{" + uri + "}" + local
	}
	// prefix not found; return as-is
	return qname
}

// compileXPath compiles an XPath expression with the given namespace bindings.
func compileXPath(expr string, nsBindings map[string]string) (*xpath3.Expression, error) {
	compiled, err := xpath3.Compile(expr)
	if err != nil {
		return nil, staticError(errCodeXTSE0165, "invalid XPath %q: %v", expr, err)
	}
	return compiled, nil
}

// compile compiles a stylesheet document into a Stylesheet.
func compile(doc *helium.Document, cfg *compileConfig) (*Stylesheet, error) {
	root := doc.DocumentElement()
	if root == nil {
		return nil, staticError(errCodeXTSE0010, "empty stylesheet document")
	}

	// Check if this is an XSLT stylesheet
	if root.URI() != NSXSLT {
		// Could be a simplified stylesheet (literal result element)
		return compileSimplified(doc, root, cfg)
	}

	localName := root.LocalName()
	if localName != "stylesheet" && localName != "transform" {
		return nil, staticError(errCodeXTSE0010, "root element must be xsl:stylesheet or xsl:transform, got %s", root.Name())
	}

	c := &compiler{
		stylesheet: &Stylesheet{
			namedTemplates: make(map[string]*Template),
			modeTemplates:  make(map[string][]*Template),
			keys:           make(map[string]*KeyDef),
			outputs:        make(map[string]*OutputDef),
			functions:      make(map[xpath3.QualifiedName]*XSLFunction),
			namespaces:     make(map[string]string),
			accumulators:   make(map[string]*AccumulatorDef),
		},
		nsBindings:    make(map[string]string),
		importStack:   make(map[string]struct{}),
		localExcludes: make(map[string]struct{}),
	}

	if cfg != nil {
		c.baseURI = cfg.baseURI
		c.resolver = cfg.resolver
	}

	// Collect namespace declarations from root
	c.collectNamespaces(root)

	// Read default-validation from stylesheet root (XSLT 3.0)
	if dv := getAttr(root, "default-validation"); dv != "" {
		c.stylesheet.defaultValidation = dv
	}

	// Read xpath-default-namespace from stylesheet root
	if xdn := getAttr(root, "xpath-default-namespace"); xdn != "" {
		c.xpathDefaultNS = xdn
	}

	// Read expand-text from stylesheet root (XSLT 3.0, using GetAttribute to catch empty values)
	if et, hasET := root.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else {
			return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:stylesheet/@expand-text", et)
		}
	}

	// Read default-validation from stylesheet root (XSLT 3.0)
	if dv := getAttr(root, "default-validation"); dv != "" {
		c.stylesheet.defaultValidation = dv
	}

	// Read default-mode from stylesheet root (XSLT 3.0)
	if dm := getAttr(root, "default-mode"); dm != "" {
		c.stylesheet.defaultMode = dm
	}

	// Read version
	c.stylesheet.version = getAttr(root, "version")
	if c.stylesheet.version == "" {
		c.stylesheet.version = "3.0"
	}

	// Parse exclude-result-prefixes
	c.stylesheet.excludePrefixes = make(map[string]struct{})
	if erp := getAttr(root, "exclude-result-prefixes"); erp != "" {
		if erp == "#all" {
			for prefix := range c.stylesheet.namespaces {
				c.stylesheet.excludePrefixes[prefix] = struct{}{}
			}
		} else {
			for _, prefix := range strings.Fields(erp) {
				c.stylesheet.excludePrefixes[prefix] = struct{}{}
			}
		}
	}
	// extension-element-prefixes are also excluded from output
	if eep := getAttr(root, "extension-element-prefixes"); eep != "" {
		for _, prefix := range strings.Fields(eep) {
			c.stylesheet.excludePrefixes[prefix] = struct{}{}
		}
	}

	// Process top-level elements
	if err := c.compileTopLevel(root); err != nil {
		return nil, err
	}

	// Sort templates by import precedence (desc) then priority (desc)
	c.sortTemplates()

	// Store the stylesheet source document and base URI
	c.stylesheet.sourceDoc = doc
	c.stylesheet.baseURI = c.baseURI

	// Post-compilation streamability analysis: check for XTSE3430 errors.
	if err := analyzeStreamability(c.stylesheet); err != nil {
		return nil, err
	}

	return c.stylesheet, nil
}

// collectNamespaces gathers namespace declarations from an element.
func (c *compiler) collectNamespaces(elem *helium.Element) {
	for _, ns := range elem.Namespaces() {
		prefix := ns.Prefix()
		uri := ns.URI()
		// All namespace bindings (including XSLT) are needed for XPath
		// resolution (e.g., document('')/*/xsl:template).
		c.nsBindings[prefix] = uri
		c.stylesheet.namespaces[prefix] = uri
	}
}

// compileTopLevel processes all top-level elements in the stylesheet.
func (c *compiler) compileTopLevel(root *helium.Element) error {
	// First pass: imports and includes (must come before templates so that
	// import precedence is properly assigned before any template registration).
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != NSXSLT {
			continue
		}
		switch elem.LocalName() {
		case "import":
			if err := c.compileImport(elem); err != nil {
				return err
			}
		case "include":
			if err := c.compileInclude(elem); err != nil {
				return err
			}
		}
	}

	// Second pass: everything else
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if elem.URI() != NSXSLT {
			continue
		}

		switch elem.LocalName() {
		case "import", "include":
			// Already processed in pass 1
		case "template":
			if err := c.compileTemplate(elem); err != nil {
				return err
			}
		case "variable":
			if err := c.compileGlobalVariable(elem); err != nil {
				return err
			}
		case "param":
			if err := c.compileGlobalParam(elem); err != nil {
				return err
			}
		case "key":
			if err := c.compileKey(elem); err != nil {
				return err
			}
		case "output":
			if err := c.compileOutput(elem); err != nil {
				return err
			}
		case "strip-space":
			c.compileSpaceHandling(elem, true)
		case "preserve-space":
			c.compileSpaceHandling(elem, false)
		case "function":
			if err := c.compileFunction(elem); err != nil {
				return err
			}
		case "decimal-format":
			c.compileDecimalFormat(elem)
		case "mode":
			if err := c.compileMode(elem); err != nil {
				return err
			}
		case "import-schema":
			if err := c.compileImportSchema(elem); err != nil {
				return err
			}
		case "accumulator":
			if err := c.compileAccumulator(elem); err != nil {
				return err
			}
		case "attribute-set":
			if err := c.compileAttributeSet(elem); err != nil {
				return err
			}
		case "namespace-alias":
			// TODO: implement namespace-alias
		default:
			// Unknown top-level element - ignore for forward compatibility
		}
	}

	return nil
}

func (c *compiler) compileTemplate(elem *helium.Element) error {
	tmpl := &Template{
		ImportPrec: c.importPrec,
	}

	// Collect namespace declarations from this template
	c.collectNamespaces(elem)

	// Inherit or override xpath-default-namespace
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
			return err
		}
		tmpl.Match = p
	}

	tmpl.Name = resolveQName(getAttr(elem, "name"), c.nsBindings)
	tmpl.Mode = getAttr(elem, "mode")
	// XSLT 3.0 §6.7: if the stylesheet has xsl:stylesheet/@default-mode,
	// templates without an explicit mode attribute belong to the default mode.
	if tmpl.Mode == "" && c.stylesheet.defaultMode != "" {
		tmpl.Mode = c.stylesheet.defaultMode
	}

	if prio := getAttr(elem, "priority"); prio != "" {
		f, err := strconv.ParseFloat(prio, 64)
		if err != nil {
			return staticError(errCodeXTSE0010, "invalid priority %q: %v", prio, err)
		}
		tmpl.Priority = f
	} else if tmpl.Match != nil && len(tmpl.Match.Alternatives) == 1 {
		tmpl.Priority = tmpl.Match.Alternatives[0].priority
	}

	// Handle exclude-result-prefixes on xsl:template
	savedExcludes := c.localExcludes
	if erp := getAttr(elem, "exclude-result-prefixes"); erp != "" {
		newExcludes := make(map[string]struct{})
		for k, v := range c.localExcludes {
			newExcludes[k] = v
		}
		if erp == "#all" {
			for prefix := range c.stylesheet.namespaces {
				newExcludes[prefix] = struct{}{}
			}
		} else {
			for _, prefix := range strings.Fields(erp) {
				newExcludes[prefix] = struct{}{}
			}
		}
		c.localExcludes = newExcludes
	}

	// Handle expand-text on xsl:template (using GetAttribute to catch empty values)
	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else {
			return staticError(errCodeXTSE0020, "%q is not a valid value for xsl:template/@expand-text", et)
		}
	}

	// Compile template body: first xsl:param elements, then instructions
	body, params, err := c.compileTemplateBody(elem)
	c.expandText = savedExpandText
	c.localExcludes = savedExcludes
	if err != nil {
		return err
	}
	tmpl.Params = params
	tmpl.Body = body

	// Register the template
	c.stylesheet.templates = append(c.stylesheet.templates, tmpl)

	if tmpl.Name != "" {
		if existing, exists := c.stylesheet.namedTemplates[tmpl.Name]; exists {
			// Same import precedence = error; different = higher precedence wins
			if existing.ImportPrec == tmpl.ImportPrec {
				return staticError(errCodeXTSE0080, "duplicate template name %q", tmpl.Name)
			}
			if tmpl.ImportPrec > existing.ImportPrec {
				c.stylesheet.namedTemplates[tmpl.Name] = tmpl
			}
			// else keep existing (higher precedence)
		} else {
			c.stylesheet.namedTemplates[tmpl.Name] = tmpl
		}
	}

	if tmpl.Match != nil {
		mode := tmpl.Mode
		if mode == "#all" {
			// Register in all existing modes plus default
			for m := range c.stylesheet.modeTemplates {
				c.stylesheet.modeTemplates[m] = append(c.stylesheet.modeTemplates[m], tmpl)
			}
			c.stylesheet.modeTemplates[""] = append(c.stylesheet.modeTemplates[""], tmpl)
		} else {
			// XSLT 2.0+: mode can be a whitespace-separated list of mode names.
			// Each mode name can be a QName, "#default", or "#all".
			modes := strings.Fields(mode)
			if len(modes) <= 1 {
				// Single mode (or empty = default mode)
				if mode == "#default" {
					mode = ""
				}
				c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
			} else {
				for _, m := range modes {
					if m == "#default" {
						m = ""
					}
					c.stylesheet.modeTemplates[m] = append(c.stylesheet.modeTemplates[m], tmpl)
				}
			}
		}
	}

	return nil
}

func (c *compiler) compileTemplateBody(elem *helium.Element) ([]Instruction, []*Param, error) {
	var params []*Param
	var body []Instruction

	inParams := true
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *helium.Element:
			if v.URI() == NSXSLT && v.LocalName() == "param" && inParams {
				p, err := c.compileParamDef(v)
				if err != nil {
					return nil, nil, err
				}
				params = append(params, p)
				continue
			}
			inParams = false
			inst, err := c.compileInstruction(v)
			if err != nil {
				return nil, nil, err
			}
			if inst != nil {
				body = append(body, inst)
			}
		case *helium.Text:
			text := string(v.Content())
			if !c.shouldStripText(text) {
				inParams = false
				inst := &LiteralTextInst{Value: text}
				if c.expandText && strings.ContainsAny(text, "{}") {
					avt, err := compileAVT(text, c.nsBindings)
					if err != nil {
						return nil, nil, err
					}
					inst.TVT = avt
				}
				body = append(body, inst)
			}
		case *helium.CDATASection:
			inParams = false
			text := string(v.Content())
			inst := &LiteralTextInst{Value: text}
			if c.expandText && strings.ContainsAny(text, "{}") {
				avt, err := compileAVT(text, c.nsBindings)
				if err != nil {
					return nil, nil, err
				}
				inst.TVT = avt
			}
			body = append(body, inst)
		}
	}

	return body, params, nil
}

func (c *compiler) compileParamDef(elem *helium.Element) (*Param, error) {
	// Validate attributes on xsl:param
	if err := validateXSLTAttrs(elem, paramAllowedAttrs); err != nil {
		return nil, err
	}

	name := getAttr(elem, "name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
	}

	// Validate boolean attribute values (including empty string)
	if reqAttr, hasReq := elem.GetAttribute("required"); hasReq {
		if err := validateBooleanAttr("xsl:param", "required", reqAttr); err != nil {
			return nil, err
		}
	}
	if tunnelAttr, hasTunnel := elem.GetAttribute("tunnel"); hasTunnel {
		if err := validateBooleanAttr("xsl:param", "tunnel", tunnelAttr); err != nil {
			return nil, err
		}
	}

	required := getAttr(elem, "required") == "yes"

	// XTSE0010: A required parameter must not have a select attribute or body content
	if required {
		selectAttr := getAttr(elem, "select")
		if selectAttr != "" {
			return nil, staticError(errCodeXTSE0010, "xsl:param with required='yes' must not have a select attribute")
		}
		// Check for non-whitespace body content
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			switch child.Type() {
			case helium.ElementNode:
				return nil, staticError(errCodeXTSE0010, "xsl:param with required='yes' must not have content")
			case helium.TextNode, helium.CDATASectionNode:
				if strings.TrimSpace(string(child.Content())) != "" {
					return nil, staticError(errCodeXTSE0010, "xsl:param with required='yes' must not have content")
				}
			}
		}
	}

	p := &Param{
		Name:     resolveQName(name, c.nsBindings),
		As:       getAttr(elem, "as"),
		Required: required,
		Tunnel:   getAttr(elem, "tunnel") == "yes",
	}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return nil, err
		}
		p.Select = expr
	}

	if selectAttr == "" {
		body, err := c.compileChildren(elem)
		if err != nil {
			return nil, err
		}
		p.Body = body
	}

	return p, nil
}

func (c *compiler) compileGlobalVariable(elem *helium.Element) error {
	// Validate attributes on xsl:variable
	if err := validateXSLTAttrs(elem, variableAllowedAttrs); err != nil {
		return err
	}

	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:variable requires name attribute")
	}

	v := &Variable{Name: resolveQName(name, c.nsBindings), As: getAttr(elem, "as")}

	selectAttr := getAttr(elem, "select")
	if selectAttr != "" {
		expr, err := compileXPath(selectAttr, c.nsBindings)
		if err != nil {
			return err
		}
		v.Select = expr
	} else {
		body, err := c.compileChildren(elem)
		if err != nil {
			return err
		}
		v.Body = body
	}

	c.stylesheet.globalVars = append(c.stylesheet.globalVars, v)
	return nil
}

func (c *compiler) compileGlobalParam(elem *helium.Element) error {
	p, err := c.compileParamDef(elem)
	if err != nil {
		return err
	}
	c.stylesheet.globalParams = append(c.stylesheet.globalParams, p)
	return nil
}

func (c *compiler) compileKey(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires name attribute")
	}

	matchAttr := getAttr(elem, "match")
	if matchAttr == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires match attribute")
	}

	matchPat, err := compilePattern(matchAttr, c.nsBindings, c.xpathDefaultNS)
	if err != nil {
		return err
	}

	expandedName := resolveQName(name, c.nsBindings)
	kd := &KeyDef{
		Name:  expandedName,
		Match: matchPat,
	}

	useAttr := getAttr(elem, "use")
	if useAttr != "" {
		useExpr, err := compileXPath(useAttr, c.nsBindings)
		if err != nil {
			return err
		}
		kd.Use = useExpr
	} else {
		// XSLT 2.0+: key may use body content instead of use attribute.
		body, _, err := c.compileTemplateBody(elem)
		if err != nil {
			return err
		}
		kd.Body = body
	}

	c.stylesheet.keys[expandedName] = kd
	return nil
}

func (c *compiler) compileOutput(elem *helium.Element) error {
	name := getAttr(elem, "name")
	outDef := &OutputDef{
		Name:     name,
		Method:   strings.ToLower(getAttr(elem, "method")),
		Encoding: getAttr(elem, "encoding"),
		Version:  getAttr(elem, "version"),
	}

	if outDef.Method == "" {
		outDef.Method = "xml"
	}
	if outDef.Encoding == "" {
		outDef.Encoding = "UTF-8"
	}

	outDef.Indent = getAttr(elem, "indent") == "yes"
	outDef.OmitDeclaration = getAttr(elem, "omit-xml-declaration") == "yes"
	outDef.Standalone = getAttr(elem, "standalone")
	outDef.DoctypePublic = getAttr(elem, "doctype-public")
	outDef.DoctypeSystem = getAttr(elem, "doctype-system")
	outDef.MediaType = getAttr(elem, "media-type")

	cdataStr := getAttr(elem, "cdata-section-elements")
	if cdataStr != "" {
		outDef.CDATASections = strings.Fields(cdataStr)
	}

	c.stylesheet.outputs[name] = outDef
	return nil
}

func (c *compiler) compileFunction(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:function requires name attribute")
	}

	// Collect namespace declarations from this element
	c.collectNamespaces(elem)

	// Resolve the prefixed name to a QualifiedName
	var qn xpath3.QualifiedName
	if strings.HasPrefix(name, "Q{") {
		// EQName: Q{uri}local
		closeBrace := strings.IndexByte(name, '}')
		if closeBrace < 0 {
			return staticError(errCodeXTSE0010, "malformed EQName in xsl:function name %q", name)
		}
		uri := name[2:closeBrace]
		local := name[closeBrace+1:]
		if uri == "" {
			return staticError(errCodeXTSE0010, "xsl:function name %q must be in a non-null namespace", name)
		}
		qn = xpath3.QualifiedName{URI: uri, Name: local}
	} else if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		uri := c.nsBindings[prefix]
		if uri == "" {
			uri = c.stylesheet.namespaces[prefix]
		}
		if uri == "" {
			return staticError(errCodeXTSE0010, "unresolved namespace prefix %q in xsl:function name %q", prefix, name)
		}
		qn = xpath3.QualifiedName{URI: uri, Name: local}
	} else {
		return staticError(errCodeXTSE0010, "xsl:function name %q must have a namespace prefix", name)
	}

	// Handle expand-text on xsl:function (using GetAttribute to catch empty values)
	savedExpandText := c.expandText
	if et, hasET := elem.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else {
			return staticError(errCodeXTSE0020, "%q is not a valid value for xsl:function/@expand-text", et)
		}
	}

	// Compile function body (params + instructions)
	body, params, err := c.compileTemplateBody(elem)
	c.expandText = savedExpandText
	if err != nil {
		return err
	}

	fn := &XSLFunction{
		Name:          qn,
		Params:        params,
		Body:          body,
		As:            getAttr(elem, "as"),
		Streamability: getAttr(elem, "streamability"),
	}

	c.stylesheet.functions[qn] = fn
	return nil
}

func (c *compiler) compileMode(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		name = "#default"
	}

	// Validate boolean attributes on xsl:mode (using GetAttribute to catch empty values)
	for _, boolAttr := range []string{"streamable", "warning-on-no-match", "warning-on-multiple-match", "typed"} {
		if v, has := elem.GetAttribute(boolAttr); has {
			if err := validateBooleanAttr("xsl:mode", boolAttr, v); err != nil {
				return err
			}
		}
	}

	// XTSE0020: visibility constraints on mode
	if vis := getAttr(elem, "visibility"); vis != "" {
		if name == "#default" && (vis == "public" || vis == "final") {
			return staticError(errCodeXTSE0020, "the unnamed mode cannot have visibility=%q", vis)
		}
		if name != "#default" && vis == "abstract" {
			return staticError(errCodeXTSE0020, "a named mode cannot have visibility=%q", vis)
		}
	}

	onNoMatch := getAttr(elem, "on-no-match")
	if onNoMatch == "" {
		onNoMatch = "text-only-copy" // XSLT 3.0 default
	}

	useAccum := getAttr(elem, "use-accumulators")
	onMultipleMatch := getAttr(elem, "on-multiple-match")

	md := &ModeDef{
		Name:            name,
		OnNoMatch:       onNoMatch,
		Streamable:      getAttr(elem, "streamable") == "yes",
		UseAccumulators: useAccum,
		OnMultipleMatch: onMultipleMatch,
	}

	if c.stylesheet.modeDefs == nil {
		c.stylesheet.modeDefs = make(map[string]*ModeDef)
	}

	// XTSE0545: conflicting mode declarations at same import precedence
	if existing, ok := c.stylesheet.modeDefs[name]; ok {
		explicitOnNoMatch := getAttr(elem, "on-no-match")
		// Check for conflicting on-no-match values
		if explicitOnNoMatch != "" && existing.OnNoMatch != "" && existing.OnNoMatch != onNoMatch {
			return staticError(errCodeXTSE0545,
				"conflicting on-no-match values for mode %q: %q vs %q", name, existing.OnNoMatch, onNoMatch)
		}
		// Check for conflicting streamable values
		if getAttr(elem, "streamable") != "" && existing.Streamable != md.Streamable {
			return staticError(errCodeXTSE0545,
				"conflicting streamable values for mode %q", name)
		}
		// Check for conflicting use-accumulators values
		if useAccum != "" && existing.UseAccumulators != "" && existing.UseAccumulators != useAccum {
			return staticError(errCodeXTSE0545,
				"conflicting use-accumulators values for mode %q: %q vs %q", name, existing.UseAccumulators, useAccum)
		}
		// Check for conflicting on-multiple-match values
		if onMultipleMatch != "" && existing.OnMultipleMatch != "" && existing.OnMultipleMatch != onMultipleMatch {
			return staticError(errCodeXTSE0545,
				"conflicting on-multiple-match values for mode %q: %q vs %q", name, existing.OnMultipleMatch, onMultipleMatch)
		}
		// Merge: keep existing, only update explicitly specified attributes
		if explicitOnNoMatch != "" {
			existing.OnNoMatch = onNoMatch
		}
		if getAttr(elem, "streamable") != "" {
			existing.Streamable = md.Streamable
		}
		if useAccum != "" {
			existing.UseAccumulators = useAccum
		}
		if onMultipleMatch != "" {
			existing.OnMultipleMatch = onMultipleMatch
		}
	} else {
		c.stylesheet.modeDefs[name] = md
	}
	return nil
}

func (c *compiler) compileAttributeSet(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:attribute-set requires name attribute")
	}
	name = resolveQName(name, c.nsBindings)

	asd := &AttributeSetDef{Name: name}

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
				return err
			}
			asd.Attrs = append(asd.Attrs, inst)
		}
	}

	if c.stylesheet.attributeSets == nil {
		c.stylesheet.attributeSets = make(map[string]*AttributeSetDef)
	}
	// Merge same-named attribute-sets (XSLT spec: union of all definitions)
	if existing, ok := c.stylesheet.attributeSets[name]; ok {
		existing.Attrs = append(existing.Attrs, asd.Attrs...)
		existing.UseAttrSets = append(existing.UseAttrSets, asd.UseAttrSets...)
	} else {
		c.stylesheet.attributeSets[name] = asd
	}
	return nil
}

func (c *compiler) compileDecimalFormat(elem *helium.Element) {
	df := xpath3.DefaultDecimalFormat()

	if v := getAttr(elem, "decimal-separator"); v != "" {
		df.DecimalSeparator = firstRune(v)
	}
	if v := getAttr(elem, "grouping-separator"); v != "" {
		df.GroupingSeparator = firstRune(v)
	}
	if v := getAttr(elem, "percent"); v != "" {
		df.Percent = firstRune(v)
	}
	if v := getAttr(elem, "per-mille"); v != "" {
		df.PerMille = firstRune(v)
	}
	if v := getAttr(elem, "zero-digit"); v != "" {
		df.ZeroDigit = firstRune(v)
	}
	if v := getAttr(elem, "digit"); v != "" {
		df.Digit = firstRune(v)
	}
	if v := getAttr(elem, "pattern-separator"); v != "" {
		df.PatternSeparator = firstRune(v)
	}
	if v := getAttr(elem, "exponent-separator"); v != "" {
		df.ExponentSeparator = firstRune(v)
	}
	if v := getAttr(elem, "infinity"); v != "" {
		df.Infinity = v
	}
	if v := getAttr(elem, "NaN"); v != "" {
		df.NaN = v
	}
	if v := getAttr(elem, "minus-sign"); v != "" {
		df.MinusSign = firstRune(v)
	}

	name := getAttr(elem, "name")
	if name == "" {
		// Default (unnamed) decimal format
		if c.stylesheet.decimalFormats == nil {
			c.stylesheet.decimalFormats = make(map[xpath3.QualifiedName]xpath3.DecimalFormat)
		}
		c.stylesheet.decimalFormats[xpath3.QualifiedName{}] = df
		return
	}

	qn := xpath3.QualifiedName{Name: name}
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		if uri, ok := c.nsBindings[prefix]; ok {
			qn = xpath3.QualifiedName{URI: uri, Name: local}
		} else {
			qn.Name = local
		}
	}
	if c.stylesheet.decimalFormats == nil {
		c.stylesheet.decimalFormats = make(map[xpath3.QualifiedName]xpath3.DecimalFormat)
	}
	c.stylesheet.decimalFormats[qn] = df
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func (c *compiler) compileSpaceHandling(elem *helium.Element, strip bool) {
	elements := getAttr(elem, "elements")
	if elements == "" {
		return
	}

	for _, name := range strings.Fields(elements) {
		nt := NameTest{}
		if idx := strings.IndexByte(name, ':'); idx >= 0 {
			nt.Prefix = name[:idx]
			nt.Local = name[idx+1:]
		} else {
			nt.Local = name
		}
		if strip {
			c.stylesheet.stripSpace = append(c.stylesheet.stripSpace, nt)
		} else {
			c.stylesheet.preserveSpace = append(c.stylesheet.preserveSpace, nt)
		}
	}
}

func (c *compiler) compileImport(elem *helium.Element) error {
	href := getAttr(elem, "href")
	if href == "" {
		return staticError(errCodeXTSE0110, "xsl:import requires href attribute")
	}
	return c.loadExternalStylesheet(href, true)
}

func (c *compiler) compileInclude(elem *helium.Element) error {
	href := getAttr(elem, "href")
	if href == "" {
		return staticError(errCodeXTSE0110, "xsl:include requires href attribute")
	}
	return c.loadExternalStylesheet(href, false)
}

func (c *compiler) loadExternalStylesheet(href string, isImport bool) error {
	// Resolve URI relative to base
	uri := href
	if c.baseURI != "" && !strings.Contains(href, "://") {
		baseDir := filepath.Dir(c.baseURI)
		uri = filepath.Join(baseDir, href)
	}

	// Circular import detection
	if _, ok := c.importStack[uri]; ok {
		return staticError(errCodeXTSE0210, "circular import/include: %s", uri)
	}
	c.importStack[uri] = struct{}{}
	defer delete(c.importStack, uri)

	// Load the document
	ctx := context.Background()
	var data []byte
	var err error

	if c.resolver != nil {
		rc, resolveErr := c.resolver.Resolve(uri)
		if resolveErr != nil {
			return fmt.Errorf("cannot resolve %q: %w", uri, resolveErr)
		}
		defer func() { _ = rc.Close() }()
		data, err = io.ReadAll(rc)
		if err != nil {
			return fmt.Errorf("cannot read %q: %w", uri, err)
		}
	} else {
		// Try direct file loading
		data, err = os.ReadFile(uri)
		if err != nil {
			return fmt.Errorf("cannot load %q: %w", uri, err)
		}
	}

	doc, err := helium.Parse(ctx, data)
	if err != nil {
		return fmt.Errorf("cannot parse %q: %w", uri, err)
	}

	savedBase := c.baseURI
	c.baseURI = uri
	defer func() { c.baseURI = savedBase }()

	importedRoot := doc.DocumentElement()
	if importedRoot == nil || importedRoot.URI() != NSXSLT {
		return staticError(errCodeXTSE0010, "imported document %q is not a stylesheet", uri)
	}

	if isImport {
		// For imports: the imported stylesheet gets current (lower) precedence.
		// After compiling, increment so the importing module's remaining
		// templates get a higher precedence.
		c.collectNamespaces(importedRoot)
		if err := c.compileTopLevel(importedRoot); err != nil {
			return err
		}
		c.importPrec++
	} else {
		// Include: same precedence as the including module.
		c.collectNamespaces(importedRoot)
		if err := c.compileTopLevel(importedRoot); err != nil {
			return err
		}
	}
	return nil
}

// compileSimplified compiles a simplified stylesheet (literal result element
// as root).
func compileSimplified(doc *helium.Document, root *helium.Element, cfg *compileConfig) (*Stylesheet, error) {
	c := &compiler{
		stylesheet: &Stylesheet{
			version:        "3.0",
			namedTemplates: make(map[string]*Template),
			modeTemplates:  make(map[string][]*Template),
			keys:           make(map[string]*KeyDef),
			outputs:        make(map[string]*OutputDef),
			functions:      make(map[xpath3.QualifiedName]*XSLFunction),
			namespaces:     make(map[string]string),
			accumulators:   make(map[string]*AccumulatorDef),
		},
		nsBindings:    make(map[string]string),
		importStack:   make(map[string]struct{}),
		localExcludes: make(map[string]struct{}),
	}

	if cfg != nil {
		c.baseURI = cfg.baseURI
		c.resolver = cfg.resolver
	}

	c.collectNamespaces(root)

	inst, err := c.compileLiteralResultElement(root)
	if err != nil {
		return nil, err
	}

	tmpl := &Template{
		Match: &Pattern{
			source: "/",
			Alternatives: []*PatternAlt{
				{
					expr:     xpath3.RootExpr{},
					priority: -0.5,
				},
			},
		},
		Body: []Instruction{inst},
	}

	c.stylesheet.templates = append(c.stylesheet.templates, tmpl)
	c.stylesheet.modeTemplates[""] = append(c.stylesheet.modeTemplates[""], tmpl)

	// Store the stylesheet source document and base URI
	c.stylesheet.sourceDoc = doc
	c.stylesheet.baseURI = c.baseURI

	return c.stylesheet, nil
}

func (c *compiler) compileImportSchema(elem *helium.Element) error {
	schemaLoc := getAttr(elem, "schema-location")
	if schemaLoc != "" {
		// File-backed schema
		uri := schemaLoc
		if c.baseURI != "" && !strings.Contains(schemaLoc, "://") && !filepath.IsAbs(schemaLoc) {
			baseDir := filepath.Dir(c.baseURI)
			uri = filepath.Join(baseDir, schemaLoc)
		}

		ctx := context.Background()
		schema, err := xsd.CompileFile(ctx, uri)
		if err != nil {
			return fmt.Errorf("xsl:import-schema: cannot compile %q: %w", uri, err)
		}
		c.stylesheet.schemas = append(c.stylesheet.schemas, schema)
		return nil
	}

	// Look for inline xs:schema child
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.LocalName() == "schema" && childElem.URI() == "http://www.w3.org/2001/XMLSchema" {
			inlineDoc := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(childElem, inlineDoc)
			if err != nil {
				return fmt.Errorf("xsl:import-schema: cannot copy inline schema: %w", err)
			}
			if err := inlineDoc.AddChild(copied); err != nil {
				return fmt.Errorf("xsl:import-schema: cannot build inline schema doc: %w", err)
			}
			ctx := context.Background()
			schema, err := xsd.Compile(ctx, inlineDoc)
			if err != nil {
				return fmt.Errorf("xsl:import-schema: cannot compile inline schema: %w", err)
			}
			c.stylesheet.schemas = append(c.stylesheet.schemas, schema)
			return nil
		}
	}

	// Namespace-only declaration — no schema to compile, accepted silently
	return nil
}

// resolveXSDTypeName normalizes a QName type reference (e.g., "xs:ID",
// "xsd:integer", or "Q{http://www.w3.org/2001/XMLSchema}ID") to the
// canonical "xs:..." prefix form used by xpath3 constants.
func resolveXSDTypeName(qname string, nsBindings map[string]string) string {
	qname = strings.TrimSpace(qname)
	if qname == "" {
		return ""
	}
	// Handle EQName Q{uri}local
	if strings.HasPrefix(qname, "Q{") {
		closeIdx := strings.IndexByte(qname, '}')
		if closeIdx > 0 {
			uri := qname[2:closeIdx]
			local := qname[closeIdx+1:]
			if uri == "http://www.w3.org/2001/XMLSchema" {
				return "xs:" + local
			}
			return qname
		}
	}
	// Handle prefix:local
	if idx := strings.IndexByte(qname, ':'); idx >= 0 {
		prefix := qname[:idx]
		local := qname[idx+1:]
		if prefix == "xs" || prefix == "xsd" {
			return "xs:" + local
		}
		if uri, ok := nsBindings[prefix]; ok && uri == "http://www.w3.org/2001/XMLSchema" {
			return "xs:" + local
		}
	}
	return qname
}

func (c *compiler) sortTemplates() {
	for mode := range c.stylesheet.modeTemplates {
		templates := c.stylesheet.modeTemplates[mode]
		sort.SliceStable(templates, func(i, j int) bool {
			// Higher import precedence first
			if templates[i].ImportPrec != templates[j].ImportPrec {
				return templates[i].ImportPrec > templates[j].ImportPrec
			}
			// Higher priority first
			if templates[i].Priority != templates[j].Priority {
				return templates[i].Priority > templates[j].Priority
			}
			// Same priority: later declaration order wins (XSLT spec §6.4)
			return i > j
		})
	}
}
