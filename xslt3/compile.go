package xslt3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// compiler holds state during stylesheet compilation.
type compiler struct {
	stylesheet      *Stylesheet
	nsBindings      map[string]string
	importPrec      int
	importStack     map[string]struct{} // circular import detection
	baseURI         string
	resolver        URIResolver
	localExcludes   map[string]struct{} // accumulated LRE-level exclude-result-prefixes
}

// getAttr returns the value of an attribute or "" if not found.
func getAttr(elem *helium.Element, name string) string {
	v, _ := elem.GetAttribute(name)
	return v
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
			namespaces:     make(map[string]string),
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

	// Read version
	c.stylesheet.version = getAttr(root,"version")
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

	// Process top-level elements
	if err := c.compileTopLevel(root); err != nil {
		return nil, err
	}

	// Sort templates by import precedence (desc) then priority (desc)
	c.sortTemplates()

	return c.stylesheet, nil
}

// collectNamespaces gathers namespace declarations from an element.
func (c *compiler) collectNamespaces(elem *helium.Element) {
	for _, ns := range elem.Namespaces() {
		prefix := ns.Prefix()
		uri := ns.URI()
		if uri != NSXSLT {
			c.nsBindings[prefix] = uri
			c.stylesheet.namespaces[prefix] = uri
		}
	}
}

// compileTopLevel processes all top-level elements in the stylesheet.
func (c *compiler) compileTopLevel(root *helium.Element) error {
	// First pass: imports (must come first)
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != NSXSLT {
			continue
		}
		if elem.LocalName() == "import" {
			if err := c.compileImport(elem); err != nil {
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
		case "import":
			// Already processed
		case "include":
			if err := c.compileInclude(elem); err != nil {
				return err
			}
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
		case "decimal-format", "namespace-alias", "attribute-set":
			// TODO: implement in later phases
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

	matchAttr := getAttr(elem,"match")
	if matchAttr != "" {
		p, err := compilePattern(matchAttr, c.nsBindings)
		if err != nil {
			return err
		}
		tmpl.Match = p
	}

	tmpl.Name = getAttr(elem,"name")
	tmpl.Mode = getAttr(elem,"mode")

	if prio := getAttr(elem,"priority"); prio != "" {
		f, err := strconv.ParseFloat(prio, 64)
		if err != nil {
			return staticError(errCodeXTSE0010, "invalid priority %q: %v", prio, err)
		}
		tmpl.Priority = f
	} else if tmpl.Match != nil && len(tmpl.Match.Alternatives) == 1 {
		tmpl.Priority = tmpl.Match.Alternatives[0].priority
	}

	// Compile template body: first xsl:param elements, then instructions
	body, params, err := c.compileTemplateBody(elem)
	if err != nil {
		return err
	}
	tmpl.Params = params
	tmpl.Body = body

	// Register the template
	c.stylesheet.templates = append(c.stylesheet.templates, tmpl)

	if tmpl.Name != "" {
		if _, exists := c.stylesheet.namedTemplates[tmpl.Name]; exists {
			return staticError(errCodeXTSE0080, "duplicate template name %q", tmpl.Name)
		}
		c.stylesheet.namedTemplates[tmpl.Name] = tmpl
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
			c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
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
			inParams = false
			text := string(v.Content())
			if strings.TrimSpace(text) != "" {
				body = append(body, &LiteralTextInst{Value: text})
			}
		case *helium.CDATASection:
			inParams = false
			body = append(body, &LiteralTextInst{Value: string(v.Content())})
		}
	}

	return body, params, nil
}

func (c *compiler) compileParamDef(elem *helium.Element) (*Param, error) {
	name := getAttr(elem,"name")
	if name == "" {
		return nil, staticError(errCodeXTSE0110, "xsl:param requires name attribute")
	}

	p := &Param{
		Name:     name,
		Required: getAttr(elem,"required") == "yes",
	}

	selectAttr := getAttr(elem,"select")
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
	name := getAttr(elem,"name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:variable requires name attribute")
	}

	v := &Variable{Name: name}

	selectAttr := getAttr(elem,"select")
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
	name := getAttr(elem,"name")
	if name == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires name attribute")
	}

	matchAttr := getAttr(elem,"match")
	if matchAttr == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires match attribute")
	}

	useAttr := getAttr(elem,"use")
	if useAttr == "" {
		return staticError(errCodeXTSE0110, "xsl:key requires use attribute")
	}

	matchPat, err := compilePattern(matchAttr, c.nsBindings)
	if err != nil {
		return err
	}

	useExpr, err := compileXPath(useAttr, c.nsBindings)
	if err != nil {
		return err
	}

	c.stylesheet.keys[name] = &KeyDef{
		Name:  name,
		Match: matchPat,
		Use:   useExpr,
	}
	return nil
}

func (c *compiler) compileOutput(elem *helium.Element) error {
	name := getAttr(elem,"name")
	outDef := &OutputDef{
		Name:     name,
		Method:   strings.ToLower(getAttr(elem,"method")),
		Encoding: getAttr(elem,"encoding"),
		Version:  getAttr(elem,"version"),
	}

	if outDef.Method == "" {
		outDef.Method = "xml"
	}
	if outDef.Encoding == "" {
		outDef.Encoding = "UTF-8"
	}

	outDef.Indent = getAttr(elem,"indent") == "yes"
	outDef.OmitDeclaration = getAttr(elem,"omit-xml-declaration") == "yes"
	outDef.Standalone = getAttr(elem,"standalone")
	outDef.DoctypePublic = getAttr(elem,"doctype-public")
	outDef.DoctypeSystem = getAttr(elem,"doctype-system")
	outDef.MediaType = getAttr(elem,"media-type")

	cdataStr := getAttr(elem,"cdata-section-elements")
	if cdataStr != "" {
		outDef.CDATASections = strings.Fields(cdataStr)
	}

	c.stylesheet.outputs[name] = outDef
	return nil
}

func (c *compiler) compileSpaceHandling(elem *helium.Element, strip bool) {
	elements := getAttr(elem,"elements")
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
	href := getAttr(elem,"href")
	if href == "" {
		return staticError(errCodeXTSE0110, "xsl:import requires href attribute")
	}
	return c.loadExternalStylesheet(href, true)
}

func (c *compiler) compileInclude(elem *helium.Element) error {
	href := getAttr(elem,"href")
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
		defer rc.Close()
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 4096)
		for {
			n, readErr := rc.Read(tmp)
			buf = append(buf, tmp[:n]...)
			if readErr != nil {
				break
			}
		}
		data = buf
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

	if isImport {
		c.importPrec++
	}

	savedBase := c.baseURI
	c.baseURI = uri
	defer func() { c.baseURI = savedBase }()

	importedRoot := doc.DocumentElement()
	if importedRoot == nil || importedRoot.URI() != NSXSLT {
		return staticError(errCodeXTSE0010, "imported document %q is not a stylesheet", uri)
	}

	c.collectNamespaces(importedRoot)
	return c.compileTopLevel(importedRoot)
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
			namespaces:     make(map[string]string),
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

	return c.stylesheet, nil
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
