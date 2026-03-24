package xslt3

import (
	"errors"
	"sort"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

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
	if err := c.ctx.Err(); err != nil {
		return err
	}

	// XTSE0120: xsl:stylesheet/xsl:transform must not contain non-whitespace text nodes.
	for child := range helium.Children(root) {
		if child.Type() == helium.TextNode {
			if strings.TrimSpace(string(child.Content())) != "" {
				return staticError(errCodeXTSE0120, "non-whitespace text node in xsl:stylesheet")
			}
		}
	}

	// Pass 1: namespace-alias (must be collected before imports/includes
	// so that LRE compilation in Pass 3 can resolve aliases).
	// Record the range of aliases added by THIS module so we can fix up
	// their importPrec after imports are processed in Pass 2.
	aliasStartIdx := len(c.stylesheet.namespaceAliases)
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		if elem.LocalName() == lexicon.XSLTElementNamespaceAlias {
			if err := c.compileNamespaceAlias(elem); err != nil {
				return err
			}
		}
	}
	aliasEndIdx := len(c.stylesheet.namespaceAliases)

	// Pre-scan: detect xsl:import-schema to set schemaAware flag early,
	// before imports/includes are compiled (they may use schema types).
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		if elem.LocalName() == lexicon.XSLTElementImportSchema {
			c.stylesheet.schemaAware = true
			break
		}
	}

	// Validate default-validation: strict/lax require a schema-aware stylesheet
	// (one that has xsl:import-schema). XTSE0020.
	dv := c.stylesheet.defaultValidation
	if (dv == validationStrict || dv == validationLax) && !c.stylesheet.schemaAware {
		return staticError(errCodeXTSE0020,
			"default-validation=%q requires xsl:import-schema", dv)
	}

	// Pre-scan local template names for XTSE3055 validation (override
	// homonymous with local declaration). This runs before use-packages
	// so the names are available during mergePackageComponents.
	c.localTemplateNames = make(map[string]struct{})
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		if elem.LocalName() == lexicon.XSLTElementTemplate {
			if n := getAttr(elem, "name"); n != "" {
				resolved := resolveQName(n, c.nsBindings)
				c.localTemplateNames[resolved] = struct{}{}
			}
		}
	}

	// Collect all static param/variable names in document order so we
	// can detect forward references within the same module.
	var moduleStaticNames []string
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		ln := elem.LocalName()
		if (ln == lexicon.XSLTElementParam || ln == lexicon.XSLTElementVariable) && xsdBoolTrue(getAttr(elem, "static")) {
			if n := getAttr(elem, "name"); n != "" {
				moduleStaticNames = append(moduleStaticNames, n)
			}
		}
	}

	// Pass 2: imports and includes' imports only. Includes are loaded and
	// cached but only their imports are processed (recursively). This
	// Evaluate static params/variables early so they are available for
	// shadow attributes on xsl:include/xsl:import elements.
	// Forward references within the same module (XPST0008) are detected
	// here by checking if the undefined variable is declared later in
	// the module's static names list.
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		ln := elem.LocalName()
		if ln != lexicon.XSLTElementParam && ln != lexicon.XSLTElementVariable {
			continue
		}
		if xsdBoolTrue(getAttr(elem, "static")) {
			name := getAttr(elem, "name")
			sel := getAttr(elem, "select")
			if name != "" && sel != "" {
				_, hasExternal := c.externalStaticParams[name]
				compiled, err := xpath3.NewCompiler().Compile(sel)
				if err == nil {
					eval := c.staticEvaluator()
					// Resolve xml:base on the element to adjust static-base-uri()
					if elemBase := getAttr(elem, lexicon.QNameXMLBase); elemBase != "" {
						eval = c.resolveXMLBaseEvaluator(eval, elemBase)
					}
					if len(c.staticVars) > 0 {
						eval = eval.Variables(xpath3.VariablesFromMap(c.staticVars))
					}
					result, err := eval.Evaluate(c.ctx, compiled, nil)
					if err != nil {
						if errors.Is(err, xpath3.ErrUndefinedVariable) {
							// Forward references within the same module are
							// XPST0008, but not when a param has an external
							// override (the override will be used instead).
							if !hasExternal && isForwardRef(err, moduleStaticNames, name) {
								return staticError(errCodeXPST0008,
									"static %s %q references undefined variable: %v", ln, name, err)
							}
						}
					} else {
						if setErr := c.setStaticVarWithKind(name, ln, result.Sequence()); setErr != nil {
							return setErr
						}
					}
				}
			}
		}
	}

	// Resolve shadow attributes on include/import elements before collection.
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		ln := elem.LocalName()
		if ln == "include" || ln == "import" {
			if err := c.resolveShadowAttributes(elem); err != nil {
				return err
			}
		}
	}

	// determines the final importPrec for this module before any templates
	// are compiled. Use-package is also processed here since it creates
	// separate package modules.
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		switch elem.LocalName() {
		case lexicon.XSLTElementImport:
			if err := c.validateXSLTAttrs(elem, map[string]struct{}{
				"href": {}, "use-when": {},
			}); err != nil {
				return err
			}
			// XTSE0260: xsl:import must be empty.
			if err := c.validateEmptyElement(elem, "xsl:import"); err != nil {
				return err
			}
			// Check use-when before loading imports (avoids loading non-existent files)
			if uw := getAttr(elem, "use-when"); uw != "" {
				include, err := c.evaluateUseWhen(uw)
				if err != nil {
					return err
				}
				if !include {
					continue
				}
			}
			if err := c.compileImport(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementInclude:
			if err := c.validateXSLTAttrs(elem, map[string]struct{}{
				"href": {}, "use-when": {},
			}); err != nil {
				return err
			}
			// XTSE0260: xsl:include must be empty.
			if err := c.validateEmptyElement(elem, "xsl:include"); err != nil {
				return err
			}
			// Check use-when before loading includes (avoids loading non-existent files)
			if uw := getAttr(elem, "use-when"); uw != "" {
				include, err := c.evaluateUseWhen(uw)
				if err != nil {
					return err
				}
				if !include {
					continue
				}
			}
			if err := c.collectIncludeImports(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementUsePackage:
			// XTSE3008: xsl:use-package must not appear in a module that is
			// reached via xsl:import (it is allowed at top level and via
			// xsl:include).
			if c.insideImport {
				return staticError(errCodeXTSE3008,
					"xsl:use-package is not allowed in an imported stylesheet module")
			}
			if uw := getAttr(elem, "use-when"); uw != "" {
				include, err := c.evaluateUseWhen(uw)
				if err != nil {
					return err
				}
				if !include {
					continue
				}
			}
			if err := c.compileUsePackage(elem); err != nil {
				return err
			}
		}
	}

	// XTSE3450: check for conflicts between the main module's static
	// declarations and imported modules' declarations (param vs variable
	// with same name, import earlier in tree order).
	if !c.insideImport {
		if err := c.checkStaticVarKindConflicts(root); err != nil {
			return err
		}
	}

	// Merge externally supplied static params early so they are available
	// during _static shadow attribute resolution and static variable
	// evaluation. They are merged again after defaults to ensure external
	// values override select="..." defaults.
	if len(c.externalStaticParams) > 0 {
		if c.staticVars == nil {
			c.staticVars = make(map[string]xpath3.Sequence, len(c.externalStaticParams))
		}
		for name, seq := range c.externalStaticParams {
			c.staticVars[name] = seq
		}
	}

	// Resolve only the _static shadow attribute on params/variables before
	// evaluating them, so that _static="..." AVTs are expanded first
	// (e.g. shadow-002). Other shadow attrs (like _select) must wait for
	// pass 3, after imports have been processed.
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		ln := elem.LocalName()
		if ln == "param" || ln == "variable" {
			if err := c.resolveSingleShadowAttribute(elem, "static"); err != nil {
				return err
			}
		}
	}

	// Detect circular references among static params/variables.
	// Even if external values break the cycle at runtime, the
	// circularity in the definitions is a static error (XPST0008).
	if err := c.detectStaticParamCycles(root); err != nil {
		return err
	}

	// Evaluate static params and variables before pass 3 so they're
	// available in use-when and shadow attributes.
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		ln := elem.LocalName()
		if ln != lexicon.XSLTElementParam && ln != lexicon.XSLTElementVariable {
			continue
		}
		if xsdBoolTrue(getAttr(elem, "static")) {
			name := getAttr(elem, "name")
			sel := getAttr(elem, "select")
			if name != "" {
				if sel != "" {
					compiled, err := xpath3.NewCompiler().Compile(sel)
					if err == nil {
						eval := c.staticEvaluator()
						// Resolve xml:base on the element to adjust static-base-uri()
						if elemBase := getAttr(elem, lexicon.QNameXMLBase); elemBase != "" {
							eval = c.resolveXMLBaseEvaluator(eval, elemBase)
						}
						if len(c.staticVars) > 0 {
							eval = eval.Variables(xpath3.VariablesFromMap(c.staticVars))
						}
						result, err := eval.Evaluate(c.ctx, compiled, nil)
						if err == nil {
							if setErr := c.setStaticVarWithKind(name, ln, result.Sequence()); setErr != nil {
								return setErr
							}
						}
					}
				} else if ln == "param" && !hasSignificantContent(elem) {
					// XSLT 3.0 spec: a param with no select and no body
					// defaults to a zero-length string when there is no as
					// attribute, or to the empty sequence when there is one.
					var val xpath3.Sequence
					if getAttr(elem, "as") == "" {
						val = xpath3.SingleString("")
					}
					if setErr := c.setStaticVarWithKind(name, ln, val); setErr != nil {
						return setErr
					}
				}
			}
		}
	}

	// Merge externally supplied static params again so external values
	// override any select="..." defaults evaluated above.
	if len(c.externalStaticParams) > 0 {
		for name, seq := range c.externalStaticParams {
			c.staticVars[name] = seq
		}
	}

	// Fix up importPrec for namespace-alias declarations added in Pass 1.
	// Pass 1 runs before imports (Pass 2), so the aliases initially get
	// the pre-import importPrec. After imports bump importPrec, we must
	// update this module's aliases to reflect the module's final importPrec.
	// Only update aliases from aliasStartIdx to aliasEndIdx (this module's
	// own aliases), not those added by imported modules during Pass 2.
	for i := aliasStartIdx; i < aliasEndIdx; i++ {
		c.stylesheet.namespaceAliases[i].ImportPrec = c.importPrec
	}

	// Pass 3: everything else in document order. Includes and templates are
	// interleaved so that effective document order is preserved (XSLT spec
	// §3.10.2: an include acts as if textually inserted at the include point).
	// All templates at this module level share the same importPrec which was
	// finalized in pass 2.
	if err := c.ctx.Err(); err != nil {
		return err
	}
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if elem.URI() != lexicon.NamespaceXSLT {
			// XTSE0130: top-level elements in null namespace are not allowed.
			if elem.URI() == "" {
				return staticError(errCodeXTSE0130,
					"top-level element %q has no namespace; all top-level elements must be in a non-null namespace", elem.Name())
			}
			continue
		}

		// Resolve shadow attributes on top-level XSLT elements.
		if err := c.resolveShadowAttributes(elem); err != nil {
			return err
		}

		// Check use-when on top-level declarations.
		// Templates handle this inside compileTemplate; import/include
		// handle it in pass 2. All other declarations are checked here.
		ln := elem.LocalName()
		if ln != "import" && ln != "include" && ln != "use-package" && ln != "template" {
			if uw := getAttr(elem, "use-when"); uw != "" {
				// A static param/variable must not reference itself in
				// use-when (XPST0008). Temporarily remove it from the
				// static vars context so self-references are detected.
				var restoreStaticVar func()
				if (ln == lexicon.XSLTElementParam || ln == lexicon.XSLTElementVariable) && xsdBoolTrue(getAttr(elem, "static")) {
					if selfName := getAttr(elem, "name"); selfName != "" {
						if oldVal, had := c.staticVars[selfName]; had {
							delete(c.staticVars, selfName)
							restoreStaticVar = func() { c.staticVars[selfName] = oldVal }
						}
					}
				}
				include, err := c.evaluateUseWhen(uw)
				if restoreStaticVar != nil {
					restoreStaticVar()
				}
				if err != nil {
					// In forward-compatible mode, unknown top-level elements
					// with failing use-when expressions are excluded rather
					// than causing a compile error (XSLT 3.0 §3.8).
					if !elems.IsKnown(ln) || !elems.IsTopLevel(ln) {
						elemVer := getAttr(elem, "version")
						if isForwardsCompatible(c.effectiveVersion) || isForwardsCompatible(elemVer) {
							continue
						}
					}
					return err
				}
				if !include {
					continue
				}
			}
		}

		switch elem.LocalName() {
		case lexicon.XSLTElementImport, lexicon.XSLTElementUsePackage:
			// Already processed in pass 2
		case lexicon.XSLTElementInclude:
			// Skip includes excluded by use-when in pass 2.
			if uw := getAttr(elem, "use-when"); uw != "" {
				include, err := c.evaluateUseWhen(uw)
				if err != nil {
					return err
				}
				if !include {
					continue
				}
			}
			if err := c.compileIncludeTemplates(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementTemplate:
			if err := c.compileTemplate(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementVariable:
			if err := c.compileGlobalVariable(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementParam:
			if err := c.compileGlobalParam(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementKey:
			if err := c.compileKey(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementOutput:
			if err := c.compileOutput(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementStripSpace:
			if err := c.compileSpaceHandling(elem, true); err != nil {
				return err
			}
		case lexicon.XSLTElementPreserveSpace:
			if err := c.compileSpaceHandling(elem, false); err != nil {
				return err
			}
		case lexicon.XSLTElementFunction:
			if err := c.compileFunction(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementDecimalFormat:
			if err := c.compileDecimalFormat(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementMode:
			if err := c.compileMode(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementImportSchema:
			if err := c.compileImportSchema(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementAccumulator:
			if err := c.compileAccumulator(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementAttributeSet:
			if err := c.compileAttributeSet(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementCharacterMap:
			if err := c.compileCharacterMap(elem); err != nil {
				return err
			}
		case lexicon.XSLTElementNamespaceAlias:
			// Already processed in pass 1
		case lexicon.XSLTElementExpose:
			// Processed after compilation via processExpose
		case lexicon.XSLTElementGlobalContextItem:
			if err := c.compileGlobalContextItem(elem); err != nil {
				return err
			}
		default:
			// XTSE0010: XSLT-defined element used in a context where it is
			// not permitted. Only recognized top-level declarations and
			// forwards-compatible unknowns (version > 3.0) are allowed.
			// Check both stylesheet-level and element-level version for
			// forward compatibility.
			elemVer := getAttr(elem, "version")
			if !isForwardsCompatible(c.effectiveVersion) && !isForwardsCompatible(elemVer) {
				return staticError(errCodeXTSE0010,
					"element xsl:%s is not allowed as a top-level declaration", elem.LocalName())
			}
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

// checkConflictingNamespaceAliases checks for XTSE0810: multiple namespace-alias
// declarations with the same stylesheet URI, same import precedence, but different
// result URIs, unless overridden by a higher-precedence alias.
func (c *compiler) checkConflictingNamespaceAliases() error {
	aliases := c.stylesheet.namespaceAliases
	// Group by stylesheet URI
	type aliasInfo struct {
		resultURI  string
		importPrec int
	}
	byURI := make(map[string][]aliasInfo)
	maxPrec := make(map[string]int)
	for _, a := range aliases {
		byURI[a.StylesheetURI] = append(byURI[a.StylesheetURI], aliasInfo{a.ResultURI, a.ImportPrec})
		if a.ImportPrec > maxPrec[a.StylesheetURI] {
			maxPrec[a.StylesheetURI] = a.ImportPrec
		}
	}
	for uri, group := range byURI {
		if len(group) < 2 {
			continue
		}
		// For each precedence level, check if all result URIs are the same
		precResults := make(map[int]map[string]struct{})
		for _, a := range group {
			if precResults[a.importPrec] == nil {
				precResults[a.importPrec] = make(map[string]struct{})
			}
			precResults[a.importPrec][a.resultURI] = struct{}{}
		}
		for prec, results := range precResults {
			if len(results) > 1 && prec >= maxPrec[uri] {
				return staticError(errCodeXTSE0810,
					"conflicting namespace-alias declarations for namespace %q with different result namespaces at the same import precedence", uri)
			}
		}
	}
	return nil
}

// compileNamespaceAlias compiles an xsl:namespace-alias declaration.
func (c *compiler) compileNamespaceAlias(elem *helium.Element) error {
	if err := c.validateXSLTAttrs(elem, map[string]struct{}{
		"stylesheet-prefix": {}, "result-prefix": {}, "use-when": {},
	}); err != nil {
		return err
	}
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
	case lexicon.PrefixXML:
		resultURI = lexicon.NamespaceXML
		resultPfx = lexicon.PrefixXML
	default:
		uri, ok := localNS[resultPrefix]
		if !ok {
			return staticError(errCodeXTSE0010, "xsl:namespace-alias: result-prefix %q is not bound to a namespace", resultPrefix)
		}
		resultURI = uri
		resultPfx = resultPrefix
	}

	c.stylesheet.namespaceAliases = append(c.stylesheet.namespaceAliases, namespaceAlias{
		StylesheetURI: stylesheetURI,
		ResultURI:     resultURI,
		ResultPrefix:  resultPfx,
		ImportPrec:    c.importPrec,
	})

	return nil
}

func (c *compiler) sortTemplates() {
	// Ensure #all templates are registered in every mode (including modes
	// that were created after the #all template was compiled).
	allTemplates := c.stylesheet.modeTemplates[modeAll]
	if len(allTemplates) > 0 {
		for mode := range c.stylesheet.modeTemplates {
			if mode == modeAll {
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
