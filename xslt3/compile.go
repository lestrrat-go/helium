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
	defaultMode    string              // current default-mode (inherited through instruction nesting)
	iterateDepth   int                 // nesting depth of xsl:iterate (for xsl:break/next-iteration validation)
	breakAllowed   bool                // true when xsl:break/xsl:next-iteration are allowed at this position
}

// resolveMode resolves mode name QNames to expanded Clark notation.
// Special mode names (#all, #default, #unnamed, #current) are returned as-is.
// Whitespace-separated lists are handled: each token is resolved independently.
func (c *compiler) resolveMode(mode string) string {
	modes := strings.Fields(mode)
	if len(modes) <= 1 {
		mode = strings.TrimSpace(mode)
		if mode == "" || mode[0] == '#' {
			return mode // special names
		}
		return resolveQName(mode, c.nsBindings)
	}
	resolved := make([]string, len(modes))
	for i, m := range modes {
		if m[0] == '#' {
			resolved[i] = m
		} else {
			resolved[i] = resolveQName(m, c.nsBindings)
		}
	}
	return strings.Join(resolved, " ")
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
			keys:           make(map[string][]*KeyDef),
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

	// Process xml:base on the stylesheet root element to adjust the
	// static base URI for all expressions in this module.
	// xml:base="innerdoc/" on a stylesheet at /a/b/style.xsl means
	// the effective base URI becomes /a/b/innerdoc/style.xsl, so that
	// filepath.Dir() returns /a/b/innerdoc for relative URI resolution.
	if xmlBase := getAttr(root, "xml:base"); xmlBase != "" && c.baseURI != "" {
		baseDir := filepath.Dir(c.baseURI)
		baseName := filepath.Base(c.baseURI)
		c.baseURI = filepath.Join(baseDir, xmlBase, baseName)
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
		resolved := resolveQName(dm, c.nsBindings)
		c.stylesheet.defaultMode = resolved
		c.defaultMode = resolved
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

	// Store the main module document in moduleDocs for document("") resolution.
	if c.stylesheet.moduleDocs == nil {
		c.stylesheet.moduleDocs = make(map[string]*helium.Document)
	}
	if c.baseURI != "" {
		c.stylesheet.moduleDocs[c.baseURI] = doc
	}

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
	// First pass: namespace-alias declarations, then imports and includes.
	// Namespace-alias must be collected before imports/includes because
	// included stylesheets' templates need the including module's aliases.
	// Process namespace-alias first for THIS module, then process imports/includes
	// which recursively collect aliases from their modules.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != NSXSLT {
			continue
		}
		if elem.LocalName() == "namespace-alias" {
			if err := c.compileNamespaceAlias(elem); err != nil {
				return err
			}
		}
	}
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
			// Already processed in pass 1
		default:
			// Unknown top-level element - ignore for forward compatibility
		}
	}

	return nil
}

// resolveNamespaceAlias looks up the namespace alias for a given stylesheet URI.
// Returns the result URI, result prefix, and true if an alias was found.
// If multiple aliases target the same stylesheet URI, the one with the highest
// import precedence wins.
func (c *compiler) resolveNamespaceAlias(stylesheetURI string) (string, string, bool) {
	bestPrec := -1
	var bestURI, bestPrefix string
	found := false
	for _, alias := range c.stylesheet.namespaceAliases {
		if alias.StylesheetURI == stylesheetURI {
			if !found || alias.ImportPrec > bestPrec {
				bestPrec = alias.ImportPrec
				bestURI = alias.ResultURI
				bestPrefix = alias.ResultPrefix
				found = true
			}
		}
	}
	return bestURI, bestPrefix, found
}

// compileNamespaceAlias compiles an xsl:namespace-alias declaration.
func (c *compiler) compileNamespaceAlias(elem *helium.Element) error {
	stylesheetPrefix, hasStylesheetPrefix := elem.GetAttribute("stylesheet-prefix")
	resultPrefix, hasResultPrefix := elem.GetAttribute("result-prefix")

	if !hasStylesheetPrefix {
		return staticError(errCodeXTSE0010, "xsl:namespace-alias requires stylesheet-prefix attribute")
	}
	if !hasResultPrefix {
		return staticError(errCodeXTSE0010, "xsl:namespace-alias requires result-prefix attribute")
	}

	// Build a local namespace map from the element's in-scope namespaces.
	// This is critical because namespace-alias prefixes are resolved in the
	// namespace context of the xsl:namespace-alias element itself, not the
	// stylesheet root (test namespace-alias-0903).
	localNS := make(map[string]string)
	for prefix, uri := range c.nsBindings {
		localNS[prefix] = uri
	}
	for _, ns := range elem.Namespaces() {
		localNS[ns.Prefix()] = ns.URI()
	}

	// Resolve stylesheet-prefix to a URI
	var stylesheetURI string
	if stylesheetPrefix == "#default" {
		stylesheetURI = localNS[""]
	} else {
		uri, ok := localNS[stylesheetPrefix]
		if !ok {
			return staticError(errCodeXTSE0010, "xsl:namespace-alias: stylesheet-prefix %q is not bound to a namespace", stylesheetPrefix)
		}
		stylesheetURI = uri
	}

	// Resolve result-prefix to a URI and preferred prefix
	var resultURI string
	var resultPfx string
	switch resultPrefix {
	case "#default":
		resultURI = localNS[""]
		resultPfx = ""
	case "xml":
		resultURI = "http://www.w3.org/XML/1998/namespace"
		resultPfx = "xml"
	default:
		uri, ok := localNS[resultPrefix]
		if !ok {
			return staticError(errCodeXTSE0010, "xsl:namespace-alias: result-prefix %q is not bound to a namespace", resultPrefix)
		}
		resultURI = uri
		resultPfx = resultPrefix
	}

	c.stylesheet.namespaceAliases = append(c.stylesheet.namespaceAliases, NamespaceAlias{
		StylesheetURI: stylesheetURI,
		ResultURI:     resultURI,
		ResultPrefix:  resultPfx,
		ImportPrec:    c.importPrec,
	})

	return nil
}

func (c *compiler) compileTemplate(elem *helium.Element) error {
	tmpl := &Template{
		ImportPrec: c.importPrec,
		BaseURI:    c.baseURI,
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
	modeAttr := getAttr(elem, "mode")
	if modeAttr != "" {
		// Resolve mode QNames to Clark notation for namespace-aware matching
		tmpl.Mode = c.resolveMode(modeAttr)
	}
	// XSLT 3.0 §6.7: if the stylesheet (or an included/imported module) has
	// default-mode, templates without an explicit mode attribute belong to it.
	if tmpl.Mode == "" && c.defaultMode != "" {
		tmpl.Mode = c.resolveMode(c.defaultMode)
	}

	// XSLT 3.0: default-mode on xsl:template affects apply-templates within
	savedDefaultMode := c.defaultMode
	if dm := getAttr(elem, "default-mode"); dm != "" {
		c.defaultMode = dm
	}
	defer func() { c.defaultMode = savedDefaultMode }()

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
	tmpl.As = getAttr(elem, "as")

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
			// Also store under the "#all" key so findBestTemplate's fallback
			// can find these templates for modes that don't exist yet.
			c.stylesheet.modeTemplates["#all"] = append(c.stylesheet.modeTemplates["#all"], tmpl)
		} else {
			// XSLT 2.0+: mode can be a whitespace-separated list of mode names.
			// Each mode name can be a QName, "#default", "#unnamed", or "#all".
			modes := strings.Fields(mode)
			if len(modes) <= 1 {
				// Single mode (or empty = default mode)
				if mode == "#default" || mode == "#unnamed" {
					mode = ""
				}
				c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
			} else {
				for _, m := range modes {
					if m == "#default" || m == "#unnamed" {
						m = ""
					} else if m == "#all" {
						// In a mode list, #all means register in all modes
						c.stylesheet.modeTemplates["#all"] = append(c.stylesheet.modeTemplates["#all"], tmpl)
						continue
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
	// Collect local namespace declarations (e.g., xmlns:ex="..." on xsl:key)
	c.collectNamespaces(elem)

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
	composite := getAttr(elem, "composite") == "yes"

	// XTSE1222: if there is already a key with the same name, the composite
	// attribute must agree
	if existingDefs, ok := c.stylesheet.keys[expandedName]; ok && len(existingDefs) > 0 {
		if existingDefs[0].Composite != composite {
			return staticError("XTSE1222",
				"xsl:key declarations with name %q have conflicting composite attributes", expandedName)
		}
	}

	kd := &KeyDef{
		Name:      expandedName,
		Match:     matchPat,
		Composite: composite,
	}

	useAttr := getAttr(elem, "use")
	if useAttr != "" {
		// XTSE0010: xsl:key with use attribute must not have child elements
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() == helium.ElementNode {
				return staticError("XTSE0010",
					"xsl:key with use attribute must not have child elements")
			}
		}
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

	c.stylesheet.keys[expandedName] = append(c.stylesheet.keys[expandedName], kd)
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
	// xsl:mode must be empty (no children)
	if elem.FirstChild() != nil {
		return staticError(errCodeXTSE0010, "xsl:mode must be empty")
	}

	name := strings.TrimSpace(getAttr(elem, "name"))
	if name == "" {
		name = "#default"
	}

	// Parse streamable with proper xs:boolean validation
	streamableStr := strings.TrimSpace(getAttr(elem, "streamable"))
	streamable := false
	if streamableStr != "" {
		v, ok := parseXSDBool(streamableStr)
		if !ok {
			return staticError(errCodeXTSE0020, "invalid value %q for streamable on xsl:mode", streamableStr)
		}
		streamable = v
	}

	// Validate on-no-match
	onNoMatch := strings.TrimSpace(getAttr(elem, "on-no-match"))
	if onNoMatch != "" {
		switch onNoMatch {
		case "text-only-copy", "shallow-copy", "deep-copy", "shallow-skip", "deep-skip", "fail":
			// valid
		default:
			return staticError(errCodeXTSE0020, "invalid value %q for on-no-match on xsl:mode", onNoMatch)
		}
	}
	// Note: we keep onNoMatch as "" when not specified, so we can detect
	// conflicts properly. The default "text-only-copy" is applied later.

	// Validate boolean attributes
	if v := getAttr(elem, "warning-on-no-match"); v != "" {
		if _, ok := parseXSDBool(v); !ok {
			return staticError(errCodeXTSE0020, "invalid value %q for warning-on-no-match on xsl:mode", v)
		}
	}
	if v := getAttr(elem, "warning-on-multiple-match"); v != "" {
		if _, ok := parseXSDBool(v); !ok {
			return staticError(errCodeXTSE0020, "invalid value %q for warning-on-multiple-match on xsl:mode", v)
		}
	}
	if v := getAttr(elem, "typed"); v != "" {
		// typed accepts "yes", "no", "true", "false", "1", "0",
		// "strict", "lax", "unspecified"
		switch v {
		case "strict", "lax", "unspecified":
			// valid non-boolean values
		default:
			if _, ok := parseXSDBool(v); !ok {
				return staticError(errCodeXTSE0020, "invalid value %q for typed on xsl:mode", v)
			}
		}
	}

	visibility := strings.TrimSpace(getAttr(elem, "visibility"))
	// XTSE0020: unnamed mode cannot have visibility="public" or "final"
	if name == "#default" && (visibility == "public" || visibility == "final") {
		return staticError(errCodeXTSE0020, "unnamed mode cannot have visibility %q", visibility)
	}

	onMultipleMatch := strings.TrimSpace(getAttr(elem, "on-multiple-match"))
	if onMultipleMatch != "" {
		switch onMultipleMatch {
		case "use-last", "fail":
			// valid
		default:
			return staticError(errCodeXTSE0020, "invalid value %q for on-multiple-match on xsl:mode", onMultipleMatch)
		}
	}

	useAccumulators := getAttr(elem, "use-accumulators")

	md := &ModeDef{
		Name:            name,
		OnNoMatch:       onNoMatch,
		Streamable:      streamable,
		Visibility:      visibility,
		OnMultipleMatch: onMultipleMatch,
		UseAccumulators: useAccumulators,
		ImportPrec:      c.importPrec,
	}

	if c.stylesheet.modeDefs == nil {
		c.stylesheet.modeDefs = make(map[string]*ModeDef)
	}

	// Check for conflicting declarations at the same import precedence
	if existing, ok := c.stylesheet.modeDefs[name]; ok {
		if existing.ImportPrec == c.importPrec {
			// Same precedence: check for conflicting attribute values (XTSE0545)
			if existing.OnNoMatch != "" && md.OnNoMatch != "" && existing.OnNoMatch != md.OnNoMatch {
				return staticError(errCodeXTSE0545, "conflicting on-no-match values for mode %q: %q vs %q", name, existing.OnNoMatch, md.OnNoMatch)
			}
			if streamableStr != "" && existing.Streamable != md.Streamable {
				return staticError(errCodeXTSE0545, "conflicting streamable values for mode %q", name)
			}
			if existing.Visibility != "" && md.Visibility != "" && existing.Visibility != md.Visibility {
				return staticError(errCodeXTSE0545, "conflicting visibility values for mode %q: %q vs %q", name, existing.Visibility, md.Visibility)
			}
			if existing.OnMultipleMatch != "" && md.OnMultipleMatch != "" && existing.OnMultipleMatch != md.OnMultipleMatch {
				return staticError(errCodeXTSE0545, "conflicting on-multiple-match values for mode %q", name)
			}
			if existing.UseAccumulators != "" && md.UseAccumulators != "" && existing.UseAccumulators != md.UseAccumulators {
				return staticError(errCodeXTSE0545, "conflicting use-accumulators values for mode %q", name)
			}
			// Non-conflicting: merge attributes (use non-empty values from new decl)
			if md.OnNoMatch != "" {
				existing.OnNoMatch = md.OnNoMatch
			}
			if md.Visibility != "" {
				existing.Visibility = md.Visibility
			}
			if md.OnMultipleMatch != "" {
				existing.OnMultipleMatch = md.OnMultipleMatch
			}
			if md.UseAccumulators != "" {
				existing.UseAccumulators = md.UseAccumulators
			}
			return nil
		}
		// Different precedence: higher precedence wins
		if c.importPrec > existing.ImportPrec {
			c.stylesheet.modeDefs[name] = md
		}
		return nil
	}

	c.stylesheet.modeDefs[name] = md
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
	name := getAttr(elem, "name")
	qn := xpath3.QualifiedName{}
	if name != "" {
		qn = xpath3.QualifiedName{Name: name}
		if idx := strings.IndexByte(name, ':'); idx >= 0 {
			prefix := name[:idx]
			local := name[idx+1:]
			if uri, ok := c.nsBindings[prefix]; ok {
				qn = xpath3.QualifiedName{URI: uri, Name: local}
			} else {
				qn.Name = local
			}
		}
	}

	if c.stylesheet.decimalFormats == nil {
		c.stylesheet.decimalFormats = make(map[xpath3.QualifiedName]xpath3.DecimalFormat)
	}

	// XSLT allows multiple xsl:decimal-format declarations with the same
	// name as long as they don't conflict. Merge into the existing entry.
	df, ok := c.stylesheet.decimalFormats[qn]
	if !ok {
		df = xpath3.DefaultDecimalFormat()
	}

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
	return c.loadExternalStylesheet(stylesheetBaseURI(elem, c.baseURI), href, true)
}

func (c *compiler) compileInclude(elem *helium.Element) error {
	href := getAttr(elem, "href")
	if href == "" {
		return staticError(errCodeXTSE0110, "xsl:include requires href attribute")
	}
	return c.loadExternalStylesheet(stylesheetBaseURI(elem, c.baseURI), href, false)
}

func stylesheetBaseURI(n helium.Node, fallback string) string {
	base := fallback
	var bases []string
	for cur := n; cur != nil; cur = cur.Parent() {
		elem, ok := cur.(*helium.Element)
		if !ok {
			continue
		}
		if xmlBase, ok := elem.GetAttributeNS("base", helium.XMLNamespace); ok && xmlBase != "" {
			bases = append(bases, xmlBase)
		}
	}
	for i := len(bases) - 1; i >= 0; i-- {
		if filepath.IsAbs(bases[i]) || strings.Contains(bases[i], "://") {
			base = bases[i]
			continue
		}
		if base == "" {
			base = bases[i]
			continue
		}
		base = helium.BuildURI(bases[i], base)
	}
	return base
}

func (c *compiler) loadExternalStylesheet(baseURI, href string, isImport bool) error {
	// Resolve URI relative to base
	uri := href
	if baseURI != "" && !strings.Contains(href, "://") && !filepath.IsAbs(href) {
		baseDir := filepath.Dir(baseURI)
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

	doc, err := parseStylesheetDocument(ctx, data, uri)
	if err != nil {
		return fmt.Errorf("cannot parse %q: %w", uri, err)
	}

	savedBase := c.baseURI
	c.baseURI = uri
	defer func() { c.baseURI = savedBase }()

	// Store the module document for document("") resolution.
	if c.stylesheet.moduleDocs == nil {
		c.stylesheet.moduleDocs = make(map[string]*helium.Document)
	}
	c.stylesheet.moduleDocs[uri] = doc

	importedRoot := doc.DocumentElement()
	if importedRoot == nil {
		return staticError(errCodeXTSE0010, "imported document %q is not a stylesheet", uri)
	}
	// If the root is not in the XSLT namespace, check for simplified stylesheet
	if importedRoot.URI() != NSXSLT {
		if _, ok := importedRoot.GetAttributeNS("version", NSXSLT); ok {
			// Simplified stylesheet — compile as a single template matching "/"
			simplified, err := compileSimplified(doc, importedRoot, &compileConfig{baseURI: uri})
			if err != nil {
				return err
			}
			// Merge the simplified stylesheet's templates into ours
			for _, tmpl := range simplified.templates {
				tmpl.ImportPrec = c.importPrec
				c.stylesheet.templates = append(c.stylesheet.templates, tmpl)
				mode := tmpl.Mode
				c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
			}
			return nil
		}
		return staticError(errCodeXTSE0010, "imported document %q is not a stylesheet", uri)
	}

	// Save/restore default-mode: included/imported stylesheets may have
	// their own default-mode that affects only their templates.
	savedDefaultMode := c.defaultMode
	if dm := getAttr(importedRoot, "default-mode"); dm != "" {
		c.defaultMode = resolveQName(dm, c.nsBindings)
	}
	defer func() { c.defaultMode = savedDefaultMode }()

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
			keys:           make(map[string][]*KeyDef),
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
		Body:    []Instruction{inst},
		BaseURI: c.baseURI,
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
	// Ensure #all templates are registered in every mode (including modes
	// that were created after the #all template was compiled).
	allTemplates := c.stylesheet.modeTemplates["#all"]
	if len(allTemplates) > 0 {
		for mode := range c.stylesheet.modeTemplates {
			if mode == "#all" {
				continue
			}
			existing := c.stylesheet.modeTemplates[mode]
			for _, at := range allTemplates {
				found := false
				for _, et := range existing {
					if et == at {
						found = true
						break
					}
				}
				if !found {
					c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], at)
				}
			}
		}
	}

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
