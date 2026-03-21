package xslt3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

func (c *compiler) compileImport(elem *helium.Element) error {
	if err := c.validateXSLTAttrs(elem, map[string]struct{}{
		"href": {}, "use-when": {},
	}); err != nil {
		return err
	}
	href := getAttr(elem, "href")
	if href == "" {
		return staticError(errCodeXTSE0110, "xsl:import requires href attribute")
	}
	return c.loadExternalStylesheet(stylesheetBaseURI(elem, c.baseURI), href, true)
}

// resolveIncludeURI resolves an xsl:include's href to an absolute URI.
func (c *compiler) resolveIncludeURI(elem *helium.Element) (string, string, error) {
	href := getAttr(elem, "href")
	if href == "" {
		return "", "", staticError(errCodeXTSE0110, "xsl:include requires href attribute")
	}
	baseURI := stylesheetBaseURI(elem, c.baseURI)

	var fragment string
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		fragment = href[idx+1:]
		href = href[:idx]
	}

	uri := href
	if baseURI != "" && !strings.Contains(href, "://") && !filepath.IsAbs(href) {
		baseDir := filepath.Dir(baseURI)
		uri = filepath.Join(baseDir, href)
	}

	importKey := uri
	if fragment != "" {
		importKey = uri + "#" + fragment
	}
	return uri, importKey, nil
}

// collectIncludeImports is the first phase of two-phase include processing.
// It loads the included document, caches the parsed root, and recursively
// processes only xsl:import elements (and imports within nested includes).
// This ensures importPrec is finalized before any templates are compiled.
func (c *compiler) collectIncludeImports(elem *helium.Element) error {
	// Check use-when before loading the included file (avoids loading
	// non-existent files when use-when="false()").
	if uw := getAttr(elem, "use-when"); uw != "" {
		include, err := c.evaluateUseWhen(uw)
		if err != nil {
			return err
		}
		if !include {
			return nil
		}
	}

	uri, importKey, err := c.resolveIncludeURI(elem)
	if err != nil {
		return err
	}

	if _, ok := c.importStack[importKey]; ok {
		return staticError(errCodeXTSE0210, "circular import/include: %s", importKey)
	}
	c.importStack[importKey] = struct{}{}
	defer delete(c.importStack, importKey)

	root, err := c.loadAndCacheInclude(uri, importKey)
	if err != nil {
		return err
	}
	if root == nil {
		// Simplified stylesheet — no imports to collect.
		return nil
	}

	savedBase := c.baseURI
	c.baseURI = uri
	defer func() { c.baseURI = savedBase }()
	savedModuleKey := c.moduleKey
	c.moduleKey = importKey
	defer func() { c.moduleKey = savedModuleKey }()

	c.collectNamespaces(root)

	// Process namespace-alias declarations from the included module.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		ce, ok := child.(*helium.Element)
		if !ok || ce.URI() != NSXSLT {
			continue
		}
		if ce.LocalName() == "namespace-alias" {
			if err := c.compileNamespaceAlias(ce); err != nil {
				return err
			}
		}
	}

	// Process imports and recursively collect imports from nested includes.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		ce, ok := child.(*helium.Element)
		if !ok || ce.URI() != NSXSLT {
			continue
		}
		switch ce.LocalName() {
		case "import":
			if err := c.compileImport(ce); err != nil {
				return err
			}
		case "include":
			if err := c.collectIncludeImports(ce); err != nil {
				return err
			}
		}
	}
	return nil
}

// loadAndCacheInclude loads a stylesheet document and caches its root element.
func (c *compiler) loadAndCacheInclude(uri, importKey string) (*helium.Element, error) {
	if root, ok := c.includeRoots[importKey]; ok {
		return root, nil
	}

	ctx := context.Background()
	var data []byte
	var err error

	if c.resolver != nil {
		rc, resolveErr := c.resolver.Resolve(uri)
		if resolveErr != nil {
			return nil, fmt.Errorf("cannot resolve %q: %w", uri, resolveErr)
		}
		defer func() { _ = rc.Close() }()
		data, err = io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("cannot read %q: %w", uri, err)
		}
	} else {
		data, err = os.ReadFile(uri)
		if err != nil {
			return nil, fmt.Errorf("cannot load %q: %w", uri, err)
		}
	}

	doc, err := parseStylesheetDocument(ctx, data, uri)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %q: %w", uri, err)
	}

	if c.stylesheet.moduleDocs == nil {
		c.stylesheet.moduleDocs = make(map[string]*helium.Document)
	}
	c.stylesheet.moduleDocs[uri] = doc

	// Handle fragment identifiers
	var fragment string
	if uri != importKey {
		fragment = importKey[len(uri)+1:]
	}

	var root *helium.Element
	if fragment != "" {
		root = findElementByID(doc, fragment)
		if root == nil {
			return nil, staticError(errCodeXTSE0010, "no element with id=%q in %q", fragment, uri)
		}
	} else {
		root = doc.DocumentElement()
	}
	if root == nil {
		return nil, staticError(errCodeXTSE0010, "included document %q is not a stylesheet", uri)
	}
	if root.URI() != NSXSLT {
		if _, ok := root.GetAttributeNS("version", NSXSLT); ok {
			// Simplified stylesheet — no XSLT root, cache as nil to signal
			// that compileIncludeTemplates should fall back to compileInclude.
			c.includeRoots[importKey] = nil
			return nil, nil
		}
		return nil, staticError(errCodeXTSE0010, "included document %q is not a stylesheet", uri)
	}

	// Check use-when on the included/imported stylesheet's root element.
	// If use-when evaluates to false, skip the entire module.
	if uw := getAttr(root, "use-when"); uw != "" {
		include, err := c.evaluateUseWhen(uw)
		if err != nil {
			return nil, err
		}
		if !include {
			c.includeRoots[importKey] = nil
			if c.useWhenExcluded == nil {
				c.useWhenExcluded = make(map[string]struct{})
			}
			c.useWhenExcluded[importKey] = struct{}{}
			return nil, nil
		}
	}

	c.includeRoots[importKey] = root
	return root, nil
}

// compileIncludeTemplates is the second phase of two-phase include processing.
// It compiles all non-import declarations from the cached included document
// in document order, interleaving with nested includes' templates.
func (c *compiler) compileIncludeTemplates(elem *helium.Element) error {
	uri, importKey, err := c.resolveIncludeURI(elem)
	if err != nil {
		return err
	}

	root := c.includeRoots[importKey]
	if root == nil {
		// If this module was excluded by use-when on its root element, skip entirely.
		if _, excluded := c.useWhenExcluded[importKey]; excluded {
			return nil
		}
		// Simplified stylesheet — fall back to the original full include path.
		// Simplified stylesheets have no imports, so document order is trivially correct.
		return c.loadExternalStylesheet(stylesheetBaseURI(elem, c.baseURI), getAttr(elem, "href"), false)
	}

	savedBase := c.baseURI
	c.baseURI = uri
	defer func() { c.baseURI = savedBase }()
	savedModuleKey := c.moduleKey
	c.moduleKey = importKey
	defer func() { c.moduleKey = savedModuleKey }()

	savedDefaultMode := c.defaultMode
	if dm := getAttr(root, "default-mode"); dm != "" {
		c.defaultMode = resolveQName(dm, c.nsBindings)
	}
	defer func() { c.defaultMode = savedDefaultMode }()

	// XTSE0265: conflicting input-type-annotations across included modules.
	if includedITA := getAttr(root, "input-type-annotations"); includedITA != "" && includedITA != "unspecified" {
		mainITA := c.stylesheet.inputTypeAnnotations
		if mainITA != "" && mainITA != "unspecified" && mainITA != includedITA {
			return staticError(errCodeXTSE0265,
				"conflicting input-type-annotations: main module has %q, included module has %q",
				mainITA, includedITA)
		}
		if mainITA == "" || mainITA == "unspecified" {
			c.stylesheet.inputTypeAnnotations = includedITA
		}
	}

	// Evaluate static params and variables so they're available for shadow attributes.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != NSXSLT {
			continue
		}
		ln := elem.LocalName()
		if ln != "param" && ln != "variable" {
			continue
		}
		if getAttr(elem, "static") == "yes" {
			name := getAttr(elem, "name")
			sel := getAttr(elem, "select")
			if name != "" && sel != "" {
				compiled, err := xpath3.Compile(sel)
				if err == nil {
					ctx := context.Background()
					if len(c.staticVars) > 0 {
						ctx = xpath3.WithVariables(ctx, c.staticVars)
					}
					result, err := compiled.Evaluate(ctx, nil)
					if err == nil {
						if setErr := c.setStaticVarWithKind(name, ln, result.Sequence()); setErr != nil {
							return setErr
						}
					}
				}
			}
		}
	}

	// Compile non-import elements in document order, with includes
	// interleaved to preserve effective document order.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		ce, ok := child.(*helium.Element)
		if !ok || ce.URI() != NSXSLT {
			continue
		}

		// Resolve shadow attributes before processing.
		if err := c.resolveShadowAttributes(ce); err != nil {
			return err
		}

		switch ce.LocalName() {
		case "import", "namespace-alias":
			// Already processed in collectIncludeImports
		case "include":
			if err := c.compileIncludeTemplates(ce); err != nil {
				return err
			}
		case "use-package":
			if err := c.compileUsePackage(ce); err != nil {
				return err
			}
		case "template":
			if err := c.compileTemplate(ce); err != nil {
				return err
			}
		case "variable":
			if err := c.compileGlobalVariable(ce); err != nil {
				return err
			}
		case "param":
			if err := c.compileGlobalParam(ce); err != nil {
				return err
			}
		case "key":
			if err := c.compileKey(ce); err != nil {
				return err
			}
		case "output":
			if err := c.compileOutput(ce); err != nil {
				return err
			}
		case "strip-space":
			if err := c.compileSpaceHandling(ce, true); err != nil {
				return err
			}
		case "preserve-space":
			if err := c.compileSpaceHandling(ce, false); err != nil {
				return err
			}
		case "function":
			if err := c.compileFunction(ce); err != nil {
				return err
			}
		case "decimal-format":
			if err := c.compileDecimalFormat(ce); err != nil {
				return err
			}
		case "mode":
			if err := c.compileMode(ce); err != nil {
				return err
			}
		case "import-schema":
			if err := c.compileImportSchema(ce); err != nil {
				return err
			}
		case "accumulator":
			if err := c.compileAccumulator(ce); err != nil {
				return err
			}
		case "attribute-set":
			if err := c.compileAttributeSet(ce); err != nil {
				return err
			}
		case "global-context-item":
			if err := c.compileGlobalContextItem(ce); err != nil {
				return err
			}
		case "character-map":
			if err := c.compileCharacterMap(ce); err != nil {
				return err
			}
		}
	}
	return nil
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
	// Extract fragment identifier (e.g. "file.xml#embedded" → fragment="embedded").
	// The fragment selects an embedded stylesheet element by ID within the document.
	var fragment string
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		fragment = href[idx+1:]
		href = href[:idx]
	}

	// Resolve URI relative to base
	uri := href
	if baseURI != "" && !strings.Contains(href, "://") && !filepath.IsAbs(href) {
		baseDir := filepath.Dir(baseURI)
		uri = filepath.Join(baseDir, href)
	}

	// Circular import detection (use uri+fragment for uniqueness)
	importKey := uri
	if fragment != "" {
		importKey = uri + "#" + fragment
	}
	if _, ok := c.importStack[importKey]; ok {
		return staticError(errCodeXTSE0210, "circular import/include: %s", importKey)
	}
	c.importStack[importKey] = struct{}{}
	defer delete(c.importStack, importKey)

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
	savedModuleKey := c.moduleKey
	c.moduleKey = importKey
	defer func() { c.moduleKey = savedModuleKey }()

	// Store the module document for document("") resolution.
	if c.stylesheet.moduleDocs == nil {
		c.stylesheet.moduleDocs = make(map[string]*helium.Document)
	}
	c.stylesheet.moduleDocs[uri] = doc

	// When a fragment identifier is present, locate the embedded stylesheet
	// element by its ID attribute instead of using the document element.
	var importedRoot *helium.Element
	if fragment != "" {
		importedRoot = findElementByID(doc, fragment)
		if importedRoot == nil {
			return staticError(errCodeXTSE0010, "no element with id=%q in %q", fragment, uri)
		}
	} else {
		importedRoot = doc.DocumentElement()
	}
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
				tmpl.MinImportPrec = c.importPrec // simplified stylesheets have no sub-imports
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

	// Save/restore expand-text: each stylesheet module has its own setting.
	savedExpandText := c.expandText
	c.expandText = false // default for new module
	if et, hasET := importedRoot.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		}
	}
	defer func() { c.expandText = savedExpandText }()

	// Save/restore effective version for forwards-compatible processing.
	savedVersion := c.effectiveVersion
	if ver := getAttr(importedRoot, "version"); ver != "" {
		c.effectiveVersion = ver
	}
	defer func() { c.effectiveVersion = savedVersion }()

	// XTSE0265: conflicting input-type-annotations across modules.
	// Error only when one module says "preserve" and another says "strip".
	// "unspecified" is compatible with either.
	if importedITA := getAttr(importedRoot, "input-type-annotations"); importedITA != "" && importedITA != "unspecified" {
		mainITA := c.stylesheet.inputTypeAnnotations
		if mainITA != "" && mainITA != "unspecified" && mainITA != importedITA {
			return staticError(errCodeXTSE0265,
				"conflicting input-type-annotations: main module has %q, imported module has %q",
				mainITA, importedITA)
		}
		// Propagate "strip" or "preserve" from imported module when the main
		// module's effective value is still "unspecified" (or unset).
		if mainITA == "" || mainITA == "unspecified" {
			c.stylesheet.inputTypeAnnotations = importedITA
		}
	}

	if isImport {
		// For imports: the imported stylesheet gets current (lower) precedence.
		// After compiling, increment so the importing module's remaining
		// templates get a higher precedence.
		//
		// Set minImportPrec so the imported module's templates know the
		// boundary of their own import tree (for xsl:apply-imports).
		savedMinImportPrec := c.minImportPrec
		c.minImportPrec = c.importPrec
		savedInsideImport := c.insideImport
		c.insideImport = true
		c.collectNamespaces(importedRoot)
		if err := c.compileTopLevel(importedRoot); err != nil {
			c.minImportPrec = savedMinImportPrec
			c.insideImport = savedInsideImport
			return err
		}
		c.minImportPrec = savedMinImportPrec
		c.insideImport = savedInsideImport
		c.importPrec++
	} else {
		// Include: same precedence as the including module.
		// Inherit minImportPrec from the including module so that
		// xsl:apply-imports in included templates searches the
		// including module's full import tree.
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
	// XTSE0150: simplified stylesheet must have xsl:version attribute
	if _, ok := root.GetAttributeNS("version", NSXSLT); !ok {
		return nil, staticError("XTSE0150",
			"simplified stylesheet (literal result element) must have an xsl:version attribute")
	}
	c := &compiler{
		stylesheet: &Stylesheet{
			version:          "3.0",
			namedTemplates:   make(map[string]*Template),
			modeTemplates:    make(map[string][]*Template),
			keys:             make(map[string][]*KeyDef),
			outputs:          make(map[string]*OutputDef),
			functions:        make(map[funcKey]*XSLFunction),
			namespaces:       make(map[string]string),
			accumulators:     make(map[string]*AccumulatorDef),
			accumulatorOrder: make([]string, 0),
		},
		nsBindings:    make(map[string]string),
		importStack:   make(map[string]struct{}),
		localExcludes: make(map[string]struct{}),
	}

	if cfg != nil {
		c.baseURI = cfg.baseURI
		c.resolver = cfg.resolver
		c.packageResolver = cfg.packageResolver
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

// findElementByID performs a depth-first search for an element whose "id"
// attribute matches the given value. This is used for embedded stylesheet
// modules referenced via fragment identifiers (e.g. "file.xml#embedded").
func findElementByID(doc *helium.Document, id string) *helium.Element {
	var walk func(helium.Node) *helium.Element
	walk = func(n helium.Node) *helium.Element {
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			elem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if v, found := elem.GetAttribute("id"); found && v == id {
				return elem
			}
			if v, found := elem.GetAttributeNS("id", lexicon.XML); found && v == id {
				return elem
			}
			if result := walk(elem); result != nil {
				return result
			}
		}
		return nil
	}
	return walk(doc)
}
