package xslt3

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/lestrrat-go/helium/xslt3/internal/elements"
)

// elems is the package-level XSLT element registry, providing metadata
// about all recognized XSLT elements (version, context, allowed attrs, etc.).
var elems = elements.NewRegistry()

// compiler holds state during stylesheet compilation.
type compiler struct {
	ctx                       context.Context
	stylesheet                *Stylesheet
	nsBindings                map[string]string
	xpathDefaultNS            string // current xpath-default-namespace
	preserveSpace             bool   // xml:space="preserve" in effect
	expandText                bool   // expand-text="yes" — text value templates enabled
	importPrec                int
	minImportPrec             int                 // importPrec value before processing this module's xsl:import elements
	importStack               map[string]struct{} // circular import detection
	baseURI                   string
	moduleKey                 string
	resolver                  URIResolver
	packageResolver           PackageResolver
	localExcludes             map[string]struct{}        // accumulated LRE-level exclude-result-prefixes (stores URIs, not prefixes)
	extensionURIs             map[string]struct{}        // namespace URIs declared as extension-element-prefixes
	defaultMode               string                     // current default-mode (inherited through instruction nesting)
	defaultCollation          string                     // current default-collation URI (resolved from space-separated list)
	iterateDepth              int                        // nesting depth of xsl:iterate (for xsl:break/next-iteration validation)
	breakAllowed              bool                       // true when xsl:break/xsl:next-iteration are allowed at this position
	includeRoots              map[string]*helium.Element // cached parsed roots from collectIncludeImports for reuse in compileIncludeTemplates
	useWhenExcluded           map[string]struct{}        // import keys excluded by use-when="false()" on the root element
	staticVars                map[string]xpath3.Sequence // static="yes" params evaluated at compile time
	staticVarKinds            map[string]string          // tracks "param" or "variable" for each static var (XTSE3450)
	mainStaticVarValues       map[string]xpath3.Sequence // values set by main module only (for XTSE3450 value conflict)
	importedStaticVarKinds    map[string]string          // kinds set by imported modules only
	importedStaticVarValues   map[string]xpath3.Sequence // values set by imported modules only
	externalStaticParams      map[string]xpath3.Sequence // externally supplied static param overrides
	insideImport              bool                       // true when compiling an imported module (for XTSE3008)
	charMapModuleKeys         map[string]string          // character-map name -> moduleKey that first defined it (for XTSE1580)
	effectiveVersion          string                     // effective XSLT version for forwards-compat processing
	importSchemas             []*xsd.Schema              // pre-compiled schemas for xsl:import-schema namespace resolution
	pendingPatternValidations []pendingPatternValidation // deferred pattern function validations
	usedModes                 map[string]struct{}        // all mode names referenced (for XTSE3085)
	usedAttrSetRefs           []string                   // all use-attribute-sets names referenced (for XTSE0710)
	localTemplateNames        map[string]struct{}        // pre-scanned named templates in this module (for XTSE3055)
	localVarNames             map[string]struct{}        // pre-scanned variable names in this module (for XTSE3050)
	localModeNames            map[string]struct{}        // pre-scanned mode names in this module (for XTSE3050)
	// Declared visibility snapshots (before xsl:expose modification)
	declaredTemplateVis map[string]string
	declaredFunctionVis map[string]string
	declaredVariableVis map[string]string
	declaredAttrSetVis  map[string]string
	declaredParamVis    map[string]string
	declaredModeVis     map[string]string
}

type pendingPatternValidation struct {
	pattern *pattern
	source  string
}

// Compiler configures XSLT 3.0 stylesheet compilation.
// It is a value-style wrapper: fluent methods return updated copies
// and the original is never mutated. The terminal method Compile
// creates an internal compileCtx immediately; downstream compilation
// uses that context, never the Compiler itself.
type Compiler struct {
	cfg *xsltCompilerCfg
}

type xsltCompilerCfg struct {
	baseURI         string
	uriResolver     URIResolver
	packageResolver PackageResolver
	staticParams    *Parameters
	importSchemas   []*xsd.Schema
}

// NewCompiler creates a new Compiler with default settings.
func NewCompiler() Compiler {
	return Compiler{cfg: &xsltCompilerCfg{}}
}

func (c Compiler) clone() Compiler {
	if c.cfg == nil {
		return Compiler{cfg: &xsltCompilerCfg{}}
	}
	cp := *c.cfg
	return Compiler{cfg: &cp}
}

// BaseURI sets the base URI for resolving relative URIs in xsl:import
// and xsl:include.
func (c Compiler) BaseURI(uri string) Compiler {
	c = c.clone()
	c.cfg.baseURI = uri
	return c
}

// URIResolver sets a custom URI resolver for loading external
// stylesheets during compilation.
func (c Compiler) URIResolver(r URIResolver) Compiler {
	c = c.clone()
	c.cfg.uriResolver = r
	return c
}

// PackageResolver sets a package resolver for xsl:use-package references.
func (c Compiler) PackageResolver(r PackageResolver) Compiler {
	c = c.clone()
	c.cfg.packageResolver = r
	return c
}

// StaticParameters sets the static parameters for compilation.
// The Parameters collection is cloned.
func (c Compiler) StaticParameters(p *Parameters) Compiler {
	c = c.clone()
	c.cfg.staticParams = p.Clone()
	return c
}

// SetStaticParameter sets a single static parameter value.
// Clone-on-write: the existing parameters are cloned before mutation
// if shared with another Compiler value.
func (c Compiler) SetStaticParameter(name string, value xpath3.Sequence) Compiler {
	c = c.clone()
	if c.cfg.staticParams == nil {
		c.cfg.staticParams = NewParameters()
	} else {
		c.cfg.staticParams = c.cfg.staticParams.Clone()
	}
	c.cfg.staticParams.Set(name, value)
	return c
}

// ImportSchemas provides pre-compiled schemas that satisfy xsl:import-schema
// declarations by target namespace when schema-location cannot be resolved.
func (c Compiler) ImportSchemas(schemas ...*xsd.Schema) Compiler {
	c = c.clone()
	c.cfg.importSchemas = append(c.cfg.importSchemas[:0:0], schemas...)
	return c
}

// ClearStaticParameters removes all static parameter bindings.
func (c Compiler) ClearStaticParameters() Compiler {
	c = c.clone()
	c.cfg.staticParams = nil
	return c
}

// Compile compiles a parsed XSLT stylesheet document into a reusable
// Stylesheet. ctx is used for cancellation/deadlines.
func (c Compiler) Compile(ctx context.Context, doc *helium.Document) (*Stylesheet, error) {
	if doc == nil {
		return nil, errNilDocument
	}
	return compile(ctx, doc, c.toCompileConfig())
}

// MustCompile is like Compile but panics on error.
// Note: context cancellation or timeout will cause a panic.
func (c Compiler) MustCompile(ctx context.Context, doc *helium.Document) *Stylesheet {
	ss, err := c.Compile(ctx, doc)
	if err != nil {
		panic("xslt3: Compile: " + err.Error())
	}
	return ss
}

// toCompileConfig converts the Compiler config to the internal compileConfig
// used by the existing compile function.
func (c Compiler) toCompileConfig() *compileConfig {
	if c.cfg == nil {
		return &compileConfig{}
	}
	cfg := &compileConfig{
		baseURI:         c.cfg.baseURI,
		resolver:        c.cfg.uriResolver,
		packageResolver: c.cfg.packageResolver,
		importSchemas:   c.cfg.importSchemas,
	}
	if c.cfg.staticParams != nil {
		cfg.staticParams = maps.Clone(c.cfg.staticParams.toMap())
	}
	return cfg
}

// recordModeUsage records a mode name (or space-separated list of mode names)
// as being referenced, for later XTSE3085 validation.
func (c *compiler) recordModeUsage(mode string) {
	if c.usedModes == nil {
		c.usedModes = make(map[string]struct{})
	}
	for _, m := range strings.Fields(mode) {
		c.usedModes[m] = struct{}{}
	}
	if mode == "" {
		// Empty mode means unnamed mode is used
		c.usedModes[""] = struct{}{}
	}
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

// resolveDefaultCollation takes a space-separated list of collation URIs
// (as specified by XSLT's default-collation attribute) and returns the first
// URI that is supported by the XPath engine.  Returns "" if none is supported.
func resolveDefaultCollation(list string) string {
	for _, uri := range strings.Fields(list) {
		if xpath3.IsCollationSupported(uri) {
			return uri
		}
	}
	return ""
}

// shouldStripText returns true if a whitespace-only text node should be stripped
// during compilation (i.e., xml:space is not "preserve").
// Uses the XML definition of whitespace (#x20, #x9, #xD, #xA) so that
// characters like U+00A0 (non-breaking space) are NOT stripped.
func (c *compiler) shouldStripText(text string) bool {
	if c.preserveSpace {
		return false
	}
	return isXMLWhitespaceOnly(text)
}

// isXMLWhitespaceOnly returns true if s is empty or contains only XML
// whitespace characters: space (#x20), tab (#x9), CR (#xD), LF (#xA).
func isXMLWhitespaceOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ' ', '\t', '\r', '\n':
			continue
		default:
			return false
		}
	}
	return true
}

// isNonWhitespaceTextNode returns true if the node is a text or CDATA node
// containing non-whitespace content.
func isNonWhitespaceTextNode(n helium.Node) bool {
	switch n.(type) {
	case *helium.Text, *helium.CDATASection:
		return !isXMLWhitespaceOnly(string(n.Content()))
	}
	return false
}

// hasNonEmptyContent returns true if the element has any child elements or
// non-whitespace text content.
func hasNonEmptyContent(elem *helium.Element) bool {
	for child := range helium.Children(elem) {
		if _, ok := child.(*helium.Element); ok {
			return true
		}
		if isNonWhitespaceTextNode(child) {
			return true
		}
	}
	return false
}

// getAttr returns the value of an attribute or "" if not found.
func getAttr(elem *helium.Element, name string) string {
	v, _ := elem.GetAttribute(name)
	return v
}

// parseDOEAttr validates and parses a disable-output-escaping attribute value.
// Valid values are "yes"/"no"/"true"/"false"/"1"/"0" (with whitespace trimming).
// Returns SEPM0016 for invalid values.
func parseDOEAttr(v string) (bool, error) {
	v = strings.TrimSpace(v)
	switch v {
	case lexicon.ValueYes, "true", "1":
		return true, nil
	case lexicon.ValueNo, "false", "0":
		return false, nil
	default:
		return false, staticError(errCodeSEPM0016,
			"%q is not a valid value for disable-output-escaping", v)
	}
}

// hasSignificantContent returns true if the element has child elements or
// non-whitespace text content (i.e., it has a sequence constructor body).
func hasSignificantContent(elem *helium.Element) bool {
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.ElementNode:
			return true
		case helium.TextNode, helium.CDATASectionNode:
			if !isXMLWhitespaceOnly(string(child.Content())) {
				return true
			}
		}
	}
	return false
}

// resolveAsType resolves an "as" attribute sequence type string using
// the current xpath-default-namespace. When the xpath-default-namespace
// is the XSD namespace and the type name is unqualified, prefix it
// with "xs:" so runtime type checks resolve correctly.
func (c *compiler) resolveAsType(asAttr string) string {
	if asAttr == "" || c.xpathDefaultNS == "" {
		return asAttr
	}
	if c.xpathDefaultNS != lexicon.NamespaceXSD {
		return asAttr
	}
	// Only resolve unqualified type names (no prefix, no Q{}).
	// Strip occurrence indicator for the check.
	typePart := strings.TrimRight(asAttr, "?*+")
	if typePart == "" || strings.ContainsAny(typePart, ":{") {
		return asAttr
	}
	// Check if it's a known XSD type name
	candidate := "xs:" + typePart
	if xpath3.IsKnownXSDType(candidate) {
		suffix := asAttr[len(typePart):]
		return candidate + suffix
	}
	return asAttr
}

// hasSignificantChildren returns true if elem has any non-whitespace-only
// text children, or element children other than xsl:fallback.
func hasSignificantChildren(elem *helium.Element) bool {
	for child := range helium.Children(elem) {
		switch c := child.(type) {
		case *helium.Element:
			if c.URI() == lexicon.NamespaceXSLT && c.LocalName() == lexicon.XSLTElementFallback {
				continue
			}
			return true
		case *helium.Text:
			if !isXMLWhitespaceOnly(string(c.Content())) {
				return true
			}
		}
	}
	return false
}

// parseXSDBool parses an xs:boolean value ("yes"/"no", "true"/"false", "1"/"0")
// with whitespace trimming per the XSD specification.
func parseXSDBool(s string) (bool, bool) {
	switch strings.TrimSpace(s) {
	case lexicon.ValueYes, "true", "1":
		return true, true
	case lexicon.ValueNo, "false", "0":
		return false, true
	default:
		return false, false
	}
}

// xsdBoolTrue is a convenience wrapper: returns true when s parses as an XSD
// boolean with a true value, false otherwise (including empty/absent).
func xsdBoolTrue(s string) bool {
	v, _ := parseXSDBool(s)
	return v
}

// isXSDecimal returns true if s is a valid xs:decimal lexical form:
// optional sign, digits, optional dot followed by digits. No exponent.
func isXSDecimal(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	i := 0
	if s[i] == '+' || s[i] == '-' {
		i++
	}
	if i >= len(s) {
		return false
	}
	hasDigit := false
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		hasDigit = true
		i++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			hasDigit = true
			i++
		}
	}
	return hasDigit && i == len(s)
}

// isForwardsCompatible returns true if the version string indicates
// forwards-compatible mode (version > 3.0).
func isForwardsCompatible(ver string) bool {
	if ver == "" || ver == "3.0" {
		return false
	}
	f, err := strconv.ParseFloat(ver, 64)
	if err != nil {
		return false
	}
	return f > 3.0
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
			return helium.ClarkName(uri, local)
		}
	}
	idx := strings.IndexByte(qname, ':')
	if idx < 0 {
		return qname
	}
	prefix := qname[:idx]
	local := qname[idx+1:]
	if uri, ok := nsBindings[prefix]; ok {
		return helium.ClarkName(uri, local)
	}
	// prefix not found; return as-is
	return qname
}

// isValidQName checks whether s is a valid xs:QName (NCName or NCName:NCName).
// A minimal check: must be non-empty, no whitespace, no leading/trailing dots,
// and if there is a colon there must be valid NCName parts on both sides.

// isValidEQName checks whether s is a valid EQName (Q{uri}local).
func isValidEQName(s string) bool {
	if !strings.HasPrefix(s, "Q{") {
		return false
	}
	closeIdx := strings.IndexByte(s, '}')
	if closeIdx < 0 || closeIdx == len(s)-1 {
		return false
	}
	local := s[closeIdx+1:]
	return xmlchar.IsValidNCName(local)
}

// validateEmptyElement checks that an XSLT element required to be empty has
// no non-whitespace text content or element children. Returns XTSE0260 on violation.
// When xml:space="preserve" is in effect, even whitespace-only text nodes are errors.
func (c *compiler) validateEmptyElement(elem *helium.Element, name string) error {
	xmlSpacePreserve := getAttr(elem, lexicon.QNameXMLSpace) == "preserve"
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.ElementNode:
			return staticError(errCodeXTSE0260, "%s must be empty, found child element", name)
		case helium.TextNode:
			content := string(child.Content())
			if !isXMLWhitespaceOnly(content) {
				return staticError(errCodeXTSE0260, "%s must be empty, found text content", name)
			}
			if xmlSpacePreserve && content != "" {
				return staticError(errCodeXTSE0260, "%s must be empty, whitespace preserved by xml:space", name)
			}
		}
	}
	return nil
}

// hasEffectiveContent returns true if the element has non-whitespace content
// after considering use-when exclusions. Child elements whose use-when evaluates
// to false are not counted as content.
func (c *compiler) hasEffectiveContent(elem *helium.Element) bool {
	for child := range helium.Children(elem) {
		switch v := child.(type) {
		case *helium.Element:
			// Check use-when: XSLT elements use "use-when", LREs use "xsl:use-when"
			var uw string
			if v.URI() == lexicon.NamespaceXSLT {
				uw = getAttr(v, "use-when")
			} else {
				uw, _ = v.GetAttributeNS("use-when", lexicon.NamespaceXSLT)
			}
			if uw != "" {
				include, err := c.evaluateUseWhen(uw)
				if err == nil && !include {
					continue // excluded by use-when
				}
			}
			return true
		case *helium.Text:
			if !isXMLWhitespaceOnly(string(v.Content())) {
				return true
			}
		}
	}
	return false
}

// setStaticVar sets a static variable value and records its kind
// ("param" or "variable") for XTSE3450 conflict detection.
func (c *compiler) setStaticVar(name string, value xpath3.Sequence) {
	if c.staticVars == nil {
		c.staticVars = make(map[string]xpath3.Sequence)
	}
	c.staticVars[name] = value
}

// setStaticVarWithKind sets a static variable value and records its
// kind ("param" or "variable") for XTSE3450 conflict detection.
// When called from an imported module, the kind is recorded in
// importedStaticVarKinds. When called from the main module, it
// checks against previously imported kinds for conflicts.
func (c *compiler) setStaticVarWithKind(name, kind string, value xpath3.Sequence) error { //nolint:unparam // always nil but callers check for future-proofing
	if c.insideImport {
		if c.importedStaticVarKinds == nil {
			c.importedStaticVarKinds = make(map[string]string)
		}
		if c.importedStaticVarValues == nil {
			c.importedStaticVarValues = make(map[string]xpath3.Sequence)
		}
		// Record the import's kind. If two imports disagree, record
		// the conflict (first import wins for kind tracking).
		if _, exists := c.importedStaticVarKinds[name]; !exists {
			c.importedStaticVarKinds[name] = kind
			c.importedStaticVarValues[name] = value
		}
		c.setStaticVar(name, value)
		return nil
	}
	// Main module: check against imported kinds for conflicts.
	if c.staticVarKinds == nil {
		c.staticVarKinds = make(map[string]string)
	}
	if c.mainStaticVarValues == nil {
		c.mainStaticVarValues = make(map[string]xpath3.Sequence)
	}
	c.staticVarKinds[name] = kind
	c.mainStaticVarValues[name] = value
	c.setStaticVar(name, value)
	return nil
}

// detectStaticParamCycles checks for circular references among
// static params/variables. Even if external values break the cycle
// at runtime, the circularity in definitions is XPST0008.
func (c *compiler) detectStaticParamCycles(root *helium.Element) error {
	// Collect static param/variable names and their select expressions.
	type staticDef struct {
		name string
		sel  string
	}
	var defs []staticDef
	names := make(map[string]struct{})
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		ln := elem.LocalName()
		if ln != lexicon.XSLTElementParam && ln != lexicon.XSLTElementVariable {
			continue
		}
		if !xsdBoolTrue(getAttr(elem, "static")) {
			continue
		}
		name := getAttr(elem, "name")
		sel := getAttr(elem, "select")
		if name != "" && sel != "" {
			defs = append(defs, staticDef{name: name, sel: sel})
			names[name] = struct{}{}
		}
	}
	if len(defs) < 2 {
		return nil
	}

	// Build dependency graph: for each static var, find which other
	// static vars its select expression references via $name.
	deps := make(map[string][]string)
	for _, d := range defs {
		for _, other := range defs {
			if other.name == d.name {
				continue
			}
			// Check if d.sel contains a reference to $other.name.
			ref := "$" + other.name
			idx := 0
			for idx < len(d.sel) {
				pos := strings.Index(d.sel[idx:], ref)
				if pos < 0 {
					break
				}
				pos += idx
				end := pos + len(ref)
				// Verify the character after the match is not a
				// name character (to avoid $foobar matching $foo).
				if end < len(d.sel) {
					ch := d.sel[end]
					if ch == '_' || ch == '-' || ch == '.' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
						idx = end
						continue
					}
				}
				deps[d.name] = append(deps[d.name], other.name)
				break
			}
		}
	}

	// Detect cycles using DFS with coloring.
	const (
		white = 0 // unvisited
		gray  = 1 // in progress
		black = 2 // done
	)
	color := make(map[string]int)
	var hasCycle bool
	var dfs func(string)
	dfs = func(name string) {
		if hasCycle {
			return
		}
		color[name] = gray
		for _, dep := range deps[name] {
			switch color[dep] {
			case gray:
				hasCycle = true
				return
			case white:
				dfs(dep)
				if hasCycle {
					return
				}
			}
		}
		color[name] = black
	}
	for _, d := range defs {
		if color[d.name] == white {
			dfs(d.name)
			if hasCycle {
				return staticError(errCodeXPST0008,
					"circular reference among static parameters/variables")
			}
		}
	}
	return nil
}

// checkStaticVarKindConflicts checks for XTSE3450 conflicts between
// the main module's static declarations and imported modules' declarations.
// Must be called after all imports have been processed.
func (c *compiler) checkStaticVarKindConflicts(root *helium.Element) error {
	if len(c.importedStaticVarKinds) == 0 || len(c.staticVarKinds) == 0 {
		return nil
	}
	// Walk the main module's children in document order. When we see
	// an xsl:import followed (later in document order) by a static
	// param/variable with the same name but different kind than what
	// the import declared, raise XTSE3450.
	// Also raise XTSE3450 when the kind matches but the value differs.
	seenImport := false
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		ln := elem.LocalName()
		if ln == "import" {
			seenImport = true
			continue
		}
		if !seenImport {
			continue
		}
		if (ln == lexicon.XSLTElementParam || ln == lexicon.XSLTElementVariable) && xsdBoolTrue(getAttr(elem, "static")) {
			name := getAttr(elem, "name")
			if importedKind, ok := c.importedStaticVarKinds[name]; ok {
				if importedKind != ln {
					return staticError(errCodeXTSE3450,
						"static %s $%s conflicts with imported declaration as %s", ln, name, importedKind)
				}
				// Same kind: check for value conflict.
				if importedVal, haveVal := c.importedStaticVarValues[name]; haveVal {
					mainVal := c.mainStaticVarValues[name]
					if !staticSequenceEqual(mainVal, importedVal) {
						return staticError(errCodeXTSE3450,
							"static %s $%s has conflicting values between main module and imported module", ln, name)
					}
				}
			}
		}
	}
	return nil
}

// staticSequenceEqual compares two xpath3.Sequence values for equality
// by comparing their string representations. Suitable for static variable
// values which are simple atomic types.
func staticSequenceEqual(a, b xpath3.Sequence) bool {
	aLen := 0
	if a != nil {
		aLen = sequence.Len(a)
	}
	bLen := 0
	if b != nil {
		bLen = b.Len()
	}
	if aLen != bLen {
		return false
	}
	for i := range aLen {
		if fmt.Sprint(a.Get(i)) != fmt.Sprint(b.Get(i)) {
			return false
		}
	}
	return true
}

// isForwardRef checks whether an ErrUndefinedVariable error references a
// variable that has NOT been declared yet in the module. moduleStaticNames
// lists the names seen so far (including currentName). If the undefined
// variable name is NOT in that list, it may come from an import that has
// not been processed yet, which is not a forward-reference error.
func isForwardRef(err error, moduleStaticNames []string, currentName string) bool {
	msg := err.Error()
	// Extract the variable name from "xpath3: undefined variable: $NAME"
	const prefix = "$"
	idx := strings.LastIndex(msg, prefix)
	if idx < 0 {
		return false
	}
	refName := msg[idx+1:]

	// The referenced name must be declared in this module's static
	// names list but AFTER the current name.
	foundCurrent := false
	for _, n := range moduleStaticNames {
		if n == currentName {
			foundCurrent = true
			continue
		}
		if foundCurrent && n == refName {
			return true // forward reference within same module
		}
	}
	return false
}

func isReservedNamespace(uri string) bool {
	switch uri {
	case lexicon.NamespaceXSLT,
		lexicon.NamespaceXSD,
		lexicon.NamespaceFn,
		lexicon.NamespaceMath,
		lexicon.NamespaceMap,
		lexicon.NamespaceArray,
		lexicon.NamespaceXML,
		lexicon.NamespaceXMLNS:
		return true
	}
	return false
}

// checkQNamePrefix validates that if a QName has a prefix, that prefix is declared
// in the current namespace bindings. Returns XTSE0280 for undeclared prefixes.
func (c *compiler) checkQNamePrefix(name, context string) error {
	if strings.HasPrefix(name, "Q{") {
		return nil // EQName, no prefix to resolve
	}
	if idx := strings.IndexByte(name, ':'); idx > 0 {
		prefix := name[:idx]
		if _, ok := c.nsBindings[prefix]; !ok {
			if _, ok := c.stylesheet.namespaces[prefix]; !ok {
				return staticError(errCodeXTSE0280, "undeclared namespace prefix %q in %s name %q", prefix, context, name)
			}
		}
	}
	return nil
}

func isValidQName(s string) bool {
	if s == "" {
		return false
	}
	parts := strings.SplitN(s, ":", 2)
	for _, p := range parts {
		if !xmlchar.IsValidNCName(p) {
			return false
		}
	}
	return true
}

// resolveShadowAttributes processes shadow attributes on an XSLT element.
// A shadow attribute _foo="avt" is evaluated at compile time and replaces
// the regular foo attribute. Per XSLT 3.0 §3.5.2: "A shadow attribute
// takes precedence over the corresponding regular attribute."
func (c *compiler) resolveShadowAttributes(elem *helium.Element) error {
	attrs := elem.Attributes()
	for _, attr := range attrs {
		name := attr.LocalName()
		if attr.URI() != "" || len(name) < 2 || name[0] != '_' {
			continue
		}
		realName := name[1:]
		avtStr := attr.Value()
		avt, err := compileAVT(avtStr, c.nsBindings)
		if err != nil {
			return staticError(errCodeXTSE0020,
				"invalid AVT in shadow attribute _%s: %v", realName, err)
		}

		// Evaluate using static variables and XSLT static functions
		// (system-property, function-available, etc. per XSLT 3.0 §3.5.2)
		eval := c.useWhenEvaluator()
		val, err := avt.evaluateStatic(eval, nil)
		if err != nil {
			return staticError(errCodeXTSE0020,
				"error evaluating shadow attribute _%s: %v", realName, err)
		}

		// Remove the shadow attribute and set the real one.
		// Use literal mode: avt values are plain text that may contain &.
		elem.RemoveAttribute("_" + realName)
		_ = elem.SetLiteralAttribute(realName, val)
	}
	return nil
}

// resolveSingleShadowAttribute resolves just one shadow attribute (_name) on
// an element, leaving all other shadow attributes intact.
func (c *compiler) resolveSingleShadowAttribute(elem *helium.Element, name string) error {
	shadowName := "_" + name
	avtStr, hasShadow := elem.GetAttribute(shadowName)
	if !hasShadow {
		return nil
	}
	avt, err := compileAVT(avtStr, c.nsBindings)
	if err != nil {
		return staticError(errCodeXTSE0020,
			"invalid AVT in shadow attribute _%s: %v", name, err)
	}
	eval := c.useWhenEvaluator()
	val, err := avt.evaluateStatic(eval, nil)
	if err != nil {
		return staticError(errCodeXTSE0020,
			"error evaluating shadow attribute _%s: %v", name, err)
	}
	elem.RemoveAttribute(shadowName)
	_ = elem.SetLiteralAttribute(name, val)
	return nil
}

// compileXPath compiles an XPath expression with the given namespace bindings.
// staticEvaluator returns an Evaluator suitable for evaluating static
// XPath expressions at compile time (static="yes" params/variables). It
// provides fn:transform and other XSLT static functions so that compile-time
// evaluation can invoke dynamic transforms (used by use-when patterns).
func (c *compiler) staticEvaluator() xpath3.Evaluator {
	fns := map[string]xpath3.Function{
		"transform": &xsltFunc{min: 1, max: 1, fn: c.staticFnTransform},
	}
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(xpath3.FunctionLibraryFromMaps(fns, nil))
	ns := make(map[string]string, len(c.nsBindings)+1)
	for k, v := range c.nsBindings {
		ns[k] = v
	}
	if c.xpathDefaultNS != "" {
		ns[""] = c.xpathDefaultNS
	}
	if len(ns) > 0 {
		eval = eval.Namespaces(ns)
	}
	if c.baseURI != "" {
		eval = eval.BaseURI(c.baseURI)
	}
	return eval
}

// resolveXMLBaseEvaluator resolves an xml:base attribute value against the
// compiler's base URI and returns an evaluator with the updated base URI.
func (c *compiler) resolveXMLBaseEvaluator(eval xpath3.Evaluator, xmlBase string) xpath3.Evaluator {
	if strings.Contains(xmlBase, "://") {
		return eval.BaseURI(xmlBase)
	}
	if c.baseURI == "" {
		return eval
	}
	base, err := url.Parse(c.baseURI)
	if err != nil {
		return eval
	}
	ref, err := url.Parse(xmlBase)
	if err != nil {
		return eval
	}
	resolved := base.ResolveReference(ref).String()
	return eval.BaseURI(resolved)
}

// staticFnTransform implements fn:transform for compile-time evaluation of
// static variables. It compiles and executes the referenced stylesheet in a
// fresh transform context.
func (c *compiler) staticFnTransform(ctx context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
	// Create a temporary execContext with the compiler's base URI so that
	// relative stylesheet-location paths resolve correctly.
	ec := &execContext{
		stylesheet:          &Stylesheet{baseURI: c.baseURI},
		resultDoc:           helium.NewDefaultDocument(),
		globalVars:          make(map[string]xpath3.Sequence),
		outputStack:         []*outputFrame{{doc: helium.NewDefaultDocument(), current: helium.NewDefaultDocument()}},
		keyTables:           make(map[string]*keyTable),
		docCache:            make(map[string]*helium.Document),
		functionResultCache: make(map[string]xpath3.Sequence),
		accumulatorState:    make(map[string]xpath3.Sequence),
		transformCtx:        ctx,
		resultDocuments:     make(map[string]*helium.Document),
		usedResultURIs:      make(map[string]struct{}),
	}
	ec.setCurrentTemplate(nil)
	return ec.fnTransform(ctx, args)
}

// useWhenEvaluator builds a compile-time evaluator for use-when evaluation
// and shadow attribute resolution. It provides XSLT static functions
// (system-property, function-available, type-available, element-available).
func (c *compiler) useWhenEvaluator() xpath3.Evaluator {
	fns := map[string]xpath3.Function{
		"function-available": &xsltFunc{min: 1, max: 2, fn: c.useWhenFunctionAvailable},
		"system-property":    &xsltFunc{min: 1, max: 1, fn: c.useWhenSystemProperty},
		"type-available":     &xsltFunc{min: 1, max: 1, fn: c.useWhenTypeAvailable},
		"element-available":  &xsltFunc{min: 1, max: 1, fn: c.useWhenElementAvailable},
	}
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		Functions(xpath3.FunctionLibraryFromMaps(fns, nil))
	if len(c.nsBindings) > 0 || c.xpathDefaultNS != "" {
		ns := make(map[string]string, len(c.nsBindings)+1)
		for k, v := range c.nsBindings {
			ns[k] = v
		}
		if c.xpathDefaultNS != "" {
			ns[""] = c.xpathDefaultNS
		}
		eval = eval.Namespaces(ns)
	}
	if len(c.staticVars) > 0 {
		eval = eval.Variables(xpath3.VariablesFromMap(c.staticVars))
	}
	if c.baseURI != "" {
		eval = eval.BaseURI(ensureFileURI(c.baseURI))
	}
	return eval
}

func compileXPath(expr string, _ map[string]string) (*xpath3.Expression, error) {
	compiled, err := xpath3.NewCompiler().Compile(expr)
	if err != nil {
		return nil, staticError(errCodeXTSE0165, "invalid XPath %q: %v", expr, err)
	}
	return compiled, nil
}

// compile compiles a stylesheet document into a Stylesheet.
func compile(ctx context.Context, doc *helium.Document, cfg *compileConfig) (*Stylesheet, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root := doc.DocumentElement()
	if root == nil {
		return nil, staticError(errCodeXTSE0010, "empty stylesheet document")
	}

	// Check if this is an XSLT stylesheet
	if root.URI() != lexicon.NamespaceXSLT {
		// Could be a simplified stylesheet (literal result element)
		return compileSimplified(ctx, doc, root, cfg)
	}

	localName := root.LocalName()
	if localName != "stylesheet" && localName != "transform" && localName != "package" {
		return nil, staticError(errCodeXTSE0010, "root element must be xsl:stylesheet, xsl:transform, or xsl:package, got %s", root.Name())
	}

	c := &compiler{
		ctx: ctx,
		stylesheet: &Stylesheet{
			namedTemplates:   make(map[string]*template),
			modeTemplates:    make(map[string][]*template),
			keys:             make(map[string][]*keyDef),
			outputs:          make(map[string]*OutputDef),
			functions:        make(map[funcKey]*xslFunction),
			namespaces:       make(map[string]string),
			accumulators:     make(map[string]*accumulatorDef),
			accumulatorOrder: make([]string, 0),
		},
		nsBindings:    make(map[string]string),
		importStack:   make(map[string]struct{}),
		localExcludes: make(map[string]struct{}),
		includeRoots:  make(map[string]*helium.Element),
	}

	// Parse xsl:package metadata
	if localName == "package" {
		c.stylesheet.packageName = getAttr(root, "name")
		c.stylesheet.packageVersion = getAttr(root, "package-version")
		c.stylesheet.isPackage = true
		c.stylesheet.declaredModes = true // default for xsl:package
		if dm := getAttr(root, "declared-modes"); dm != "" {
			if v, ok := parseXSDBool(dm); ok {
				c.stylesheet.declaredModes = v
			}
		}
	} else {
		// package-version is only allowed on xsl:package, not xsl:stylesheet/xsl:transform.
		// In forwards-compatible mode (version > 3.0), unknown attributes are silently ignored.
		ver := getAttr(root, "version")
		if getAttr(root, "package-version") != "" && !isForwardsCompatible(ver) {
			return nil, staticError(errCodeXTSE0090,
				"attribute package-version is not allowed on xsl:%s", localName)
		}
	}

	if cfg != nil {
		c.baseURI = cfg.baseURI
		c.moduleKey = cfg.baseURI
		c.resolver = cfg.resolver
		c.packageResolver = cfg.packageResolver
		c.externalStaticParams = cfg.staticParams
		c.importSchemas = cfg.importSchemas
	}
	if c.moduleKey == "" {
		c.moduleKey = "<main>"
	}

	// Process xml:base on the stylesheet root element to adjust the
	// static base URI for all expressions in this module.
	// xml:base="innerdoc/" on a stylesheet at /a/b/style.xsl means
	// the effective base URI becomes /a/b/innerdoc/style.xsl, so that
	// filepath.Dir() returns /a/b/innerdoc for relative URI resolution.
	if xmlBase := getAttr(root, lexicon.QNameXMLBase); xmlBase != "" {
		if strings.Contains(xmlBase, "://") {
			// Absolute URI (http://, file://, etc.) — use as-is.
			c.baseURI = xmlBase
		} else if c.baseURI != "" {
			baseDir := filepath.Dir(c.baseURI)
			baseName := filepath.Base(c.baseURI)
			c.baseURI = filepath.Join(baseDir, xmlBase, baseName)
		}
	}

	// Collect namespace declarations from root
	c.collectNamespaces(root)

	// Read default-validation from stylesheet root (XSLT 3.0).
	// Per §3.6, the default value is "strip".
	if dv := getAttr(root, "default-validation"); dv != "" {
		c.stylesheet.defaultValidation = dv
	} else if c.stylesheet.defaultValidation == "" {
		c.stylesheet.defaultValidation = validationStrip
	}

	// Read input-type-annotations from stylesheet root
	if ita := getAttr(root, "input-type-annotations"); ita != "" {
		c.stylesheet.inputTypeAnnotations = ita
	}

	// Read xpath-default-namespace from stylesheet root
	if xdn := getAttr(root, "xpath-default-namespace"); xdn != "" {
		c.xpathDefaultNS = xdn
	}

	// Read default-collation from stylesheet root
	if dc := getAttr(root, "default-collation"); dc != "" {
		if uri := resolveDefaultCollation(dc); uri != "" {
			c.defaultCollation = uri
			c.stylesheet.defaultCollation = uri
		} else {
			return nil, staticError(errCodeXTSE0125,
				"no recognized collation URI in default-collation %q", dc)
		}
	}

	// Read expand-text from stylesheet root (XSLT 3.0, using GetAttribute to catch empty values)
	if et, hasET := root.GetAttribute("expand-text"); hasET {
		if v, ok := parseXSDBool(et); ok {
			c.expandText = v
		} else {
			return nil, staticError(errCodeXTSE0020, "%q is not a valid value for xsl:stylesheet/@expand-text", et)
		}
	}

	// Read default-validation from stylesheet root (XSLT 3.0).
	// Per §3.6, the default value is "strip".
	if dv := getAttr(root, "default-validation"); dv != "" {
		c.stylesheet.defaultValidation = dv
	} else if c.stylesheet.defaultValidation == "" {
		c.stylesheet.defaultValidation = validationStrip
	}

	// Read input-type-annotations from stylesheet root
	if ita := getAttr(root, "input-type-annotations"); ita != "" {
		c.stylesheet.inputTypeAnnotations = ita
	}

	// Read default-mode from stylesheet root (XSLT 3.0)
	if dm := getAttr(root, "default-mode"); dm != "" {
		resolved := resolveQName(dm, c.nsBindings)
		c.stylesheet.defaultMode = resolved
		c.defaultMode = resolved
	}

	// Read version — required on xsl:stylesheet and xsl:transform (XTSE0010).
	// Allow _version shadow attribute (resolved later) to satisfy the check.
	c.stylesheet.version = getAttr(root, "version")
	if c.stylesheet.version == "" {
		if _, hasShadow := root.GetAttribute("_version"); !hasShadow {
			return nil, staticError(errCodeXTSE0010, "xsl:%s requires a version attribute", localName)
		}
		c.stylesheet.version = "3.0" // placeholder until shadow resolution
	} else if !isXSDecimal(c.stylesheet.version) {
		// XTSE0110: version must be a valid xs:decimal value
		return nil, staticError(errCodeXTSE0110,
			"xsl:%s version attribute %q is not a valid xs:decimal", localName, c.stylesheet.version)
	}
	c.effectiveVersion = c.stylesheet.version

	// Validate attributes on root element now that effectiveVersion is known.
	if err := c.validateXSLTAttrs(root, map[string]struct{}{
		"version": {}, "id": {}, "default-mode": {},
		"default-validation": {}, "input-type-annotations": {},
		"default-collation": {}, "use-when": {},
		"name": {}, "package-version": {}, "declared-modes": {},
	}); err != nil {
		return nil, err
	}

	// Parse exclude-result-prefixes
	c.stylesheet.excludePrefixes = make(map[string]struct{})
	c.stylesheet.excludeURIs = make(map[string]struct{})
	if erp := getAttr(root, "exclude-result-prefixes"); erp != "" {
		if erp == "#all" {
			for prefix := range c.stylesheet.namespaces {
				c.stylesheet.excludePrefixes[prefix] = struct{}{}
			}
		} else {
			for _, prefix := range strings.Fields(erp) {
				if prefix == "#default" {
					c.stylesheet.excludePrefixes[""] = struct{}{}
					continue
				}
				// XTSE0808: prefix must be declared
				if _, ok := c.stylesheet.namespaces[prefix]; !ok {
					if _, ok := c.nsBindings[prefix]; !ok {
						return nil, staticError(errCodeXTSE0808,
							"undeclared namespace prefix %q in exclude-result-prefixes", prefix)
					}
				}
				c.stylesheet.excludePrefixes[prefix] = struct{}{}
			}
		}
	}
	// extension-element-prefixes are also excluded from output.
	// XTSE0800: reserved namespaces must not be used as extension namespaces.
	if eep := getAttr(root, "extension-element-prefixes"); eep != "" {
		for _, prefix := range strings.Fields(eep) {
			uri := c.nsBindings[prefix]
			if isReservedNamespace(uri) {
				return nil, staticError(errCodeXTSE0800,
					"reserved namespace %q (prefix %q) must not be used as an extension namespace", uri, prefix)
			}
			c.stylesheet.excludePrefixes[prefix] = struct{}{}
			if uri != "" {
				if c.extensionURIs == nil {
					c.extensionURIs = make(map[string]struct{})
				}
				c.extensionURIs[uri] = struct{}{}
			}
		}
	}
	// Resolve excluded prefixes to URIs now, before template processing
	// can mutate c.stylesheet.namespaces. XSLT spec: exclude-result-prefixes
	// identifies namespace URIs, not prefix names.
	for prefix := range c.stylesheet.excludePrefixes {
		if uri, ok := c.stylesheet.namespaces[prefix]; ok {
			c.stylesheet.excludeURIs[uri] = struct{}{}
		}
	}

	// Process top-level elements
	if err := c.compileTopLevel(root); err != nil {
		return nil, err
	}

	// XTSE0810: check for conflicting namespace-alias declarations
	// (same stylesheet URI, same import precedence, different result URIs)
	// unless a higher-precedence alias overrides. This runs after all
	// imports/includes are processed so import precedences are finalized.
	if err := c.checkConflictingNamespaceAliases(); err != nil {
		return nil, err
	}

	// XTSE0270: check for conflicting strip-space/preserve-space at same precedence
	if err := c.checkSpaceConflicts(); err != nil {
		return nil, err
	}

	// Process xsl:expose declarations for packages (after all components are compiled)
	if err := c.processExpose(root); err != nil {
		return nil, err
	}

	// XTSE1290: check deferred decimal-format conflicts after all imports
	for qn, conflictPrec := range c.stylesheet.decimalFmtConflicts {
		actualPrec := c.stylesheet.decimalFmtPrec[qn]
		if actualPrec == conflictPrec {
			return nil, staticError(errCodeXTSE1290,
				"conflicting xsl:decimal-format declarations")
		}
	}

	// XTSE1300: validate decimal format character uniqueness after all declarations
	for _, df := range c.stylesheet.decimalFormats {
		if err := checkDecimalFormatCharConflicts(df); err != nil {
			return nil, err
		}
	}

	// XTSE0545: check deferred mode property conflicts after all imports
	for name, md := range c.stylesheet.modeDefs {
		if md.conflictOnNoMatch {
			return nil, staticError(errCodeXTSE0545, "conflicting on-no-match values for mode %q", name)
		}
		if md.conflictStreamable {
			return nil, staticError(errCodeXTSE0545, "conflicting streamable values for mode %q", name)
		}
		if md.conflictVisibility {
			return nil, staticError(errCodeXTSE0545, "conflicting visibility values for mode %q", name)
		}
		if md.conflictOnMultiple {
			return nil, staticError(errCodeXTSE0545, "conflicting on-multiple-match values for mode %q", name)
		}
		if md.conflictAccumulator {
			return nil, staticError(errCodeXTSE0545, "conflicting use-accumulators values for mode %q", name)
		}
	}

	// XTSE3350: check deferred accumulator duplicate conflicts
	for name, acc := range c.stylesheet.accumulators {
		if acc.conflictDuplicate {
			return nil, staticError(errCodeXTSE3350,
				"duplicate xsl:accumulator %q at the same import precedence", name)
		}
	}

	// XTSE3080: check that no abstract components remain unimplemented.
	// Only check for the top-level stylesheet, not sub-packages (which
	// may intentionally pass abstract components through).
	if !cfg.isSubPackage {
		if err := c.checkAbstractComponents(); err != nil {
			return nil, err
		}
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

	// Validate deferred pattern function calls now that all xsl:function
	// declarations have been processed.
	for _, pv := range c.pendingPatternValidations {
		if err := c.validatePatternFunctions(pv.pattern, pv.source); err != nil {
			return nil, err
		}
	}

	// XTSE0680: when xsl:call-template passes a non-tunnel parameter that
	// the target template declares as tunnel="yes" (or vice versa), that is
	// a static error.
	if err := checkCallTemplateTunnelMismatch(c.stylesheet); err != nil {
		return nil, err
	}

	// XTSE1590: validate character map references.
	if err := checkCharacterMapRefs(c.stylesheet); err != nil {
		return nil, err
	}

	// XTSE0720: detect cyclic use-attribute-sets references.
	if err := checkAttributeSetCycles(c.stylesheet); err != nil {
		return nil, err
	}

	// XTSE0730: streamable attribute set must only reference other streamable sets.
	if err := checkAttributeSetStreamable(c.stylesheet); err != nil {
		return nil, err
	}

	// XTSE0710: validate all use-attribute-sets references point to declared attribute sets.
	// Hidden/private attribute-sets from used packages are in the map for
	// internal reference resolution but must not be directly used from the
	// using stylesheet. Locally declared private attribute-sets (OwnerPackage
	// == nil) are always accessible from the stylesheet's own code.
	for _, ref := range c.usedAttrSetRefs {
		asd, ok := c.stylesheet.attributeSets[ref]
		if !ok {
			return nil, staticError(errCodeXTSE0710, "use-attribute-sets references undeclared attribute-set %q", ref)
		}
		if asd != nil && asd.OwnerPackage != nil && (asd.Visibility == visHidden || asd.Visibility == visPrivate) {
			return nil, staticError(errCodeXTSE0710, "use-attribute-sets references undeclared attribute-set %q", ref)
		}
	}

	// Post-compilation streamability analysis: check for XTSE3430 errors.
	if err := analyzeStreamability(c.stylesheet); err != nil {
		return nil, err
	}

	// XTSE3085: when declared-modes is true (default for xsl:package),
	// all modes used in templates or xsl:apply-templates must be explicitly
	// declared via xsl:mode.
	if c.stylesheet.declaredModes {
		if err := checkDeclaredModes(c.stylesheet, c.usedModes); err != nil {
			return nil, err
		}
	}

	// XTSE3105: for modes with typed="strict", check that match pattern
	// element names are declared in the imported schemas.
	if err := checkTypedModePatterns(c.stylesheet); err != nil {
		return nil, err
	}

	// Preserve compiler configuration so fn:transform nested compiles
	// behave consistently with top-level compilation.
	if c.packageResolver != nil {
		c.stylesheet.packageResolver = c.packageResolver
	}
	if c.resolver != nil {
		c.stylesheet.uriResolver = c.resolver
	}
	if len(c.importSchemas) > 0 {
		c.stylesheet.compilerImportSchemas = c.importSchemas
	}

	// Topologically sort accumulators so dependencies are evaluated first.
	sortAccumulatorOrder(c.stylesheet)

	return c.stylesheet, nil
}

// checkAbstractComponents checks that no abstract components remain
// unimplemented in the final compiled stylesheet. XTSE3080.
func (c *compiler) checkAbstractComponents() error {
	// Check named templates
	for name, tmpl := range c.stylesheet.namedTemplates {
		if tmpl.Visibility == visAbstract {
			return staticError(errCodeXTSE3080,
				"abstract template %q has no implementation", name)
		}
	}
	// Check functions
	for fk, fn := range c.stylesheet.functions {
		if fn.Visibility == visAbstract {
			return staticError(errCodeXTSE3080,
				"abstract function %s#%d has no implementation",
				fmt.Sprintf("{%s}%s", fk.Name.URI, fk.Name.Name), fk.Arity)
		}
	}
	return nil
}

// checkDeclaredModes verifies that all modes referenced in the stylesheet
// (on xsl:template and xsl:apply-templates) are declared via xsl:mode when
// declared-modes is true (XTSE3085). The usedModes set is populated during
// compilation from both template mode attributes and apply-templates mode
// attributes.
func checkDeclaredModes(ss *Stylesheet, usedModes map[string]struct{}) error {
	for mode := range usedModes {
		if mode == modeCurrent || mode == modeAll {
			continue
		}
		// The unnamed mode ("", "#default", "#unnamed") must also be declared
		// when declared-modes is in effect.
		if mode == "" || mode == modeDefault || mode == modeUnnamed {
			// Check if unnamed mode is declared. It can be declared as "",
			// "#default", or "#unnamed" in modeDefs.
			if ss.modeDefs != nil {
				if _, ok := ss.modeDefs[""]; ok {
					continue
				}
				if _, ok := ss.modeDefs[modeDefault]; ok {
					continue
				}
				if _, ok := ss.modeDefs[modeUnnamed]; ok {
					continue
				}
			}
			displayName := mode
			if displayName == "" {
				displayName = modeUnnamed
			}
			return staticError(errCodeXTSE3085, "mode %q is used but not declared (declared-modes is in effect)", displayName)
		}
		if _, ok := ss.modeDefs[mode]; !ok {
			return staticError(errCodeXTSE3085, "mode %q is used but not declared (declared-modes is in effect)", mode)
		}
	}
	return nil
}

// checkCharacterMapRefs validates XTSE1590: all character map references
// (in use-character-maps attributes) must resolve to defined character maps.
func checkCharacterMapRefs(ss *Stylesheet) error {
	// Check use-character-maps references within character map definitions
	for _, cm := range ss.characterMaps {
		for _, ref := range cm.UseCharacterMaps {
			if _, ok := ss.characterMaps[ref]; !ok {
				return staticError(errCodeXTSE1590,
					"character map %q references undefined character map %q", cm.Name, ref)
			}
		}
	}
	// XTSE1600: detect circular character-map references
	var detectCycle func(name string, visited map[string]struct{}) error
	detectCycle = func(name string, visited map[string]struct{}) error {
		if _, inPath := visited[name]; inPath {
			return staticError(errCodeXTSE1600,
				"circular reference in character-map %q", name)
		}
		cm, ok := ss.characterMaps[name]
		if !ok {
			return nil
		}
		visited[name] = struct{}{}
		for _, ref := range cm.UseCharacterMaps {
			if err := detectCycle(ref, visited); err != nil {
				return err
			}
		}
		delete(visited, name)
		return nil
	}
	for name := range ss.characterMaps {
		if err := detectCycle(name, make(map[string]struct{})); err != nil {
			return err
		}
	}

	// Check use-character-maps references in output definitions
	for _, outDef := range ss.outputs {
		for _, ref := range outDef.UseCharacterMaps {
			if _, ok := ss.characterMaps[ref]; !ok {
				return staticError(errCodeXTSE1590,
					"xsl:output references undefined character map %q", ref)
			}
		}
	}
	return nil
}

// checkCallTemplateTunnelMismatch validates XTSE0680: when xsl:call-template
// passes a parameter, the tunnel attribute on xsl:with-param must match the
// tunnel attribute on the target template's xsl:param.
func checkCallTemplateTunnelMismatch(ss *Stylesheet) error {
	// Build param map for each named template
	type paramInfo struct {
		tunnel   bool
		required bool
	}
	tmplParams := make(map[string]map[string]paramInfo) // template name -> param name -> info
	for _, tmpl := range ss.templates {
		if tmpl.Name == "" {
			continue
		}
		pm := make(map[string]paramInfo)
		for _, p := range tmpl.Params {
			pm[p.Name] = paramInfo{tunnel: p.Tunnel, required: p.Required}
		}
		tmplParams[tmpl.Name] = pm
	}

	// Walk all instruction trees to find callTemplateInst
	var walkBody func(body []instruction) error
	walkBody = func(body []instruction) error {
		for _, inst := range body {
			if ct, ok := inst.(*callTemplateInst); ok {
				if targetParams, found := tmplParams[ct.Name]; found {
					// Build set of supplied non-tunnel param names
					supplied := make(map[string]struct{})
					for _, wp := range ct.Params {
						if wp.Tunnel {
							continue
						}
						supplied[wp.Name] = struct{}{}
						if tp, exists := targetParams[wp.Name]; exists {
							// XTSE0680: a non-tunnel with-param matching a tunnel param is an error.
							if tp.tunnel {
								return staticError(errCodeXTSE0680,
									"xsl:call-template: non-tunnel parameter %q corresponds to tunnel parameter in target template",
									wp.Name)
							}
						} else {
							// XTSE0680: a non-tunnel with-param that has no matching
							// xsl:param in the target template is a static error.
							return staticError(errCodeXTSE0680,
								"xsl:call-template: parameter %q not declared in target template %q",
								wp.Name, ct.Name)
						}
					}
					// XTSE0690: required non-tunnel parameters must be supplied
					for pName, pInfo := range targetParams {
						if pInfo.required && !pInfo.tunnel {
							if _, ok := supplied[pName]; !ok {
								return staticError(errCodeXTSE0690,
									"xsl:call-template: required parameter %q not supplied for template %q",
									pName, ct.Name)
							}
						}
					}
				}
			}
			for _, child := range instructionChildren(inst) {
				if err := walkBody(child); err != nil {
					return err
				}
			}
		}
		return nil
	}

	for _, tmpl := range ss.templates {
		if err := walkBody(tmpl.Body); err != nil {
			return err
		}
	}
	for _, fn := range ss.functions {
		if err := walkBody(fn.Body); err != nil {
			return err
		}
	}
	for _, v := range ss.globalVars {
		if err := walkBody(v.Body); err != nil {
			return err
		}
	}
	return nil
}
