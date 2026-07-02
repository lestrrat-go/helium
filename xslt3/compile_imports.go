package xslt3

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/uripath"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

func (c *compiler) compileImport(ctx context.Context, elem *helium.Element) error {
	if err := c.validateXSLTAttrs(ctx, elem, map[string]struct{}{
		"href": {}, xslAttrUseWhen: {},
	}); err != nil {
		return err
	}
	href := getAttr(elem, "href")
	if href == "" {
		return staticError(errCodeXTSE0110, "xsl:import requires href attribute")
	}
	return c.loadExternalStylesheet(ctx, stylesheetBaseURI(elem, c.baseURI, c.moduleRoot), href, true)
}

// resolveIncludeURI resolves an xsl:include's href to an absolute URI.
func (c *compiler) resolveIncludeURI(_ context.Context, elem *helium.Element) (string, string, error) {
	href := getAttr(elem, "href")
	if href == "" {
		return "", "", staticError(errCodeXTSE0110, "xsl:include requires href attribute")
	}
	baseURI := stylesheetBaseURI(elem, c.baseURI, c.moduleRoot)

	var fragment string
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		fragment = href[idx+1:]
		href = href[:idx]
	}

	uri, err := resolveModuleURI(href, baseURI)
	if err != nil {
		return "", "", err
	}

	importKey := uri
	if fragment != "" {
		importKey = uri + "#" + fragment
	}
	return uri, importKey, nil
}

// resolveModuleURI resolves an xsl:include / xsl:import href (with any fragment
// already stripped) against the stylesheet base URI.
//
// An absolute-URI href (it carries its own scheme, e.g. "urn:shared",
// "file:/modules/m.xsl", "mem:/m.xsl") addresses its own location and is
// returned UNCHANGED — it must never be filepath.Join'ed onto the base, which
// would corrupt it (e.g. "/styles/urn:shared"). A relative href against a URI
// base is resolved with RFC 3986 semantics; against a local filesystem base it
// keeps historical filepath.Join handling. Resolution is delegated to
// [xsd.ResolveSchemaURI] / [xsd.URIScheme] — the single canonical URI helper
// shared with the schema loader — so the two layers cannot drift. Windows
// drive-letter paths (single-letter "scheme") stay filesystem paths.
func resolveModuleURI(href, baseURI string) (string, error) {
	if href == "" || baseURI == "" {
		return href, nil
	}
	// A Windows drive-letter href ("C:/..." or "C:\\...") is a filesystem path,
	// not a URI: its single-letter "scheme" is rejected by xsd.URIScheme, so
	// without this branch it would fall through to ResolveSchemaURI and be
	// lowercased / dot-segment-mangled by RFC 3986 resolution against a URI
	// base. Address its own location verbatim. This must run BEFORE the URI
	// branch below.
	if isWindowsDrivePath(href) {
		return href, nil
	}
	// Absolute-URI href, or any href against a URI base: defer to the shared
	// canonical resolver (RFC 3986 + OmitHost preservation).
	if xsd.URIScheme(href) != "" || xsd.URIScheme(baseURI) != "" {
		return xsd.ResolveSchemaURI(href, baseURI) //nolint:wrapcheck // static error passthrough
	}
	// Local filesystem base (a FILE path): resolve with forward-slash (path)
	// semantics so the result uses '/' on every OS. A Windows-absolute href was
	// already handled by isWindowsDrivePath above; uripath.IsAbsolutePath here
	// keeps a POSIX-absolute href verbatim. On Windows filepath.Dir/Join would
	// emit '\' and corrupt a virtual or POSIX-shaped base (e.g. "/virtual/x.xsl"
	// against href "common.xsl" -> "\\virtual\\common.xsl"), missing a resolver
	// keyed on the forward-slash form.
	if uripath.IsAbsolutePath(href) {
		return href, nil
	}
	return uripath.JoinLocalBaseDir(path.Dir(uripath.ToSlash(baseURI)), href), nil
}

// isWindowsDrivePath reports whether s begins with a Windows drive-letter prefix
// ("C:/" or "C:\\"). Such paths are local filesystem paths, not URI references:
// their single-letter "scheme" is deliberately not recognized by xsd.URIScheme,
// so they must be handled as filesystem paths before any URI resolution.
func isWindowsDrivePath(s string) bool {
	if len(s) < 3 {
		return false
	}
	c := s[0]
	if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
		return false
	}
	return s[1] == ':' && (s[2] == '/' || s[2] == '\\')
}

// collectIncludeImports is the first phase of two-phase include processing.
// It loads the included document, caches the parsed root, and recursively
// processes only xsl:import elements (and imports within nested includes).
// This ensures importPrec is finalized before any templates are compiled.
func (c *compiler) collectIncludeImports(ctx context.Context, elem *helium.Element) error {
	// Check use-when before loading the included file (avoids loading
	// non-existent files when use-when="false()").
	if uw := getAttr(elem, xslAttrUseWhen); uw != "" {
		include, err := c.evaluateUseWhen(ctx, uw)
		if err != nil {
			return err
		}
		if !include {
			return nil
		}
	}

	uri, importKey, err := c.resolveIncludeURI(ctx, elem)
	if err != nil {
		return err
	}

	if _, ok := c.importStack[importKey]; ok {
		return staticError(errCodeXTSE0210, "circular import/include: %s", importKey)
	}
	c.importStack[importKey] = struct{}{}
	defer delete(c.importStack, importKey)

	root, err := c.loadAndCacheInclude(ctx, uri, importKey)
	if err != nil {
		return err
	}
	if root == nil {
		// Simplified stylesheet — no imports to collect.
		return nil
	}

	savedBase := c.baseURI
	c.baseURI = moduleEffectiveBaseURI(root, uri)
	defer func() { c.baseURI = savedBase }()
	savedModuleKey := c.moduleKey
	c.moduleKey = importKey
	defer func() { c.moduleKey = savedModuleKey }()
	savedModuleRoot := c.moduleRoot
	c.moduleRoot = embeddedModuleRoot(root)
	defer func() { c.moduleRoot = savedModuleRoot }()

	c.collectNamespaces(ctx, root)

	// Process namespace-alias declarations from the included module.
	for child := range helium.Children(root) {
		ce, ok := child.(*helium.Element)
		if !ok || ce.URI() != lexicon.NamespaceXSLT {
			continue
		}
		if ce.LocalName() == lexicon.XSLTElementNamespaceAlias {
			if err := c.compileNamespaceAlias(ctx, ce); err != nil {
				return err
			}
		}
	}

	// Process imports and recursively collect imports from nested includes.
	for child := range helium.Children(root) {
		ce, ok := child.(*helium.Element)
		if !ok || ce.URI() != lexicon.NamespaceXSLT {
			continue
		}
		switch ce.LocalName() {
		case lexicon.XSLTElementImport:
			if err := c.compileImport(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementInclude:
			if err := c.collectIncludeImports(ctx, ce); err != nil {
				return err
			}
		}
	}
	return nil
}

// loadModuleDoc resolves, reads, and parses a stylesheet module document at uri.
// It enforces the opt-in URIResolver requirement (XTSE0165) and returns the
// parsed document; callers handle module-doc registration and base-URI/fragment
// bookkeeping. The resolved resource is closed before returning.
func (c *compiler) loadModuleDoc(ctx context.Context, uri string) (*helium.Document, error) {
	if c.resolver == nil {
		return nil, staticError(errCodeXTSE0165, "cannot load %q: no URIResolver configured (filesystem access is opt-in; set Compiler.URIResolver)", uri)
	}

	rc, resolveErr := c.resolver.Resolve(uri)
	if resolveErr != nil {
		return nil, fmt.Errorf("cannot resolve %q: %w", uri, resolveErr)
	}
	defer func() { _ = rc.Close() }()
	data, err := readResourceBounded(rc, c.maxResourceBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", uri, err)
	}

	doc, err := parseStylesheetDocument(ctx, c.parser, data, uri, c.allowExternalEntities, c.loadResourceBytes, c.maxResourceBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot parse %q: %w", uri, err)
	}
	return doc, nil
}

// loadAndCacheInclude loads a stylesheet document and caches its root element.
func (c *compiler) loadAndCacheInclude(ctx context.Context, uri, importKey string) (*helium.Element, error) {
	if root, ok := c.includeRoots[importKey]; ok {
		return root, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	doc, err := c.loadModuleDoc(ctx, uri)
	if err != nil {
		return nil, err
	}

	if c.stylesheet.moduleDocs == nil {
		c.stylesheet.moduleDocs = make(map[string]*helium.Document)
	}
	c.stylesheet.moduleDocs[uri] = doc

	// Handle fragment identifiers (importKey = uri + "#" + fragment)
	var fragment string
	if uri != importKey && len(importKey) > len(uri)+1 && importKey[len(uri)] == '#' {
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
	if root.URI() != lexicon.NamespaceXSLT {
		if _, ok := root.GetAttributeNS("version", lexicon.NamespaceXSLT); ok {
			// Simplified stylesheet — no XSLT root, cache as nil to signal
			// that compileIncludeTemplates should fall back to compileInclude.
			c.includeRoots[importKey] = nil
			return nil, nil //nolint:nilnil
		}
		return nil, staticError(errCodeXTSE0010, "included document %q is not a stylesheet", uri)
	}

	// Also cache the module document under its FOLDED effective base (the
	// xml:base ancestor chain folded into the module URI — the module root's own
	// xml:base for a document root, plus wrapper/document xml:base for an embedded
	// root). This module's templates compile with BaseURI =
	// moduleEffectiveBaseURI(root, uri), and doc('')/document('') from within the
	// module looks up moduleDocs by that folded key; without this entry the lookup
	// misses and wrongly falls back to the principal stylesheet. The bare-uri
	// entry above is kept for other lookups; when no xml:base applies the folded
	// base equals uri and this is a no-op.
	if effBase := moduleEffectiveBaseURI(root, uri); effBase != uri {
		c.stylesheet.moduleDocs[effBase] = doc
	}

	// Check use-when on the included/imported stylesheet's root element.
	// If use-when evaluates to false, skip the entire module. Evaluate against
	// the module's effective static base (its root xml:base folded into the
	// module URI) so doc-available()/doc() in the root use-when resolve like the
	// module's own globals, not the including module's base.
	if uw := getAttr(root, xslAttrUseWhen); uw != "" {
		savedBase := c.baseURI
		c.baseURI = moduleEffectiveBaseURI(root, uri)
		include, err := c.evaluateUseWhen(ctx, uw)
		c.baseURI = savedBase
		if err != nil {
			return nil, err
		}
		if !include {
			c.includeRoots[importKey] = nil
			if c.useWhenExcluded == nil {
				c.useWhenExcluded = make(map[string]struct{})
			}
			c.useWhenExcluded[importKey] = struct{}{}
			return nil, nil //nolint:nilnil
		}
	}

	c.includeRoots[importKey] = root
	return root, nil
}

// compileIncludeTemplates is the second phase of two-phase include processing.
// It compiles all non-import declarations from the cached included document
// in document order, interleaving with nested includes' templates.
func (c *compiler) compileIncludeTemplates(ctx context.Context, elem *helium.Element) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	uri, importKey, err := c.resolveIncludeURI(ctx, elem)
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
		return c.loadExternalStylesheet(ctx, stylesheetBaseURI(elem, c.baseURI, c.moduleRoot), getAttr(elem, "href"), false)
	}

	savedBase := c.baseURI
	c.baseURI = moduleEffectiveBaseURI(root, uri)
	defer func() { c.baseURI = savedBase }()
	savedModuleKey := c.moduleKey
	c.moduleKey = importKey
	defer func() { c.moduleKey = savedModuleKey }()
	savedModuleRoot := c.moduleRoot
	c.moduleRoot = embeddedModuleRoot(root)
	defer func() { c.moduleRoot = savedModuleRoot }()

	savedDefaultMode := c.defaultMode
	if dm := getAttr(root, "default-mode"); dm != "" {
		c.defaultMode = resolveQName(dm, c.nsBindings)
	}
	defer func() { c.defaultMode = savedDefaultMode }()

	// Apply xpath-default-namespace from included module root. Presence (even an
	// empty value, which resets to no-namespace) overrides the importing
	// module's default; absence inherits it.
	savedXPathDefaultNS := c.xpathDefaultNS
	savedHasXPathDefaultNS := c.hasXPathDefaultNS
	if xdn, ok := root.GetAttribute("xpath-default-namespace"); ok {
		c.xpathDefaultNS = xdn
		c.hasXPathDefaultNS = true
	}
	defer func() {
		c.xpathDefaultNS = savedXPathDefaultNS
		c.hasXPathDefaultNS = savedHasXPathDefaultNS
	}()

	// The included module's version governs backwards-/forwards-compatible
	// processing of its own declarations (XSLT 3.0 §3.10), independent of the
	// including module's version. A _version shadow attribute takes precedence
	// over the literal (§3.5.2), so resolve it first — mirroring the top-level
	// root — so an included module whose computed version is < 2.0 enters
	// backwards-compatible mode.
	savedVersion := c.effectiveVersion
	if _, hasShadow := root.GetAttribute("_version"); hasShadow {
		if err := c.resolveSingleShadowAttribute(ctx, root, paramVersion); err != nil {
			return err
		}
	}
	if ver := getAttr(root, "version"); ver != "" {
		c.effectiveVersion = ver
	}
	defer func() { c.effectiveVersion = savedVersion }()

	// XTSE0265: conflicting input-type-annotations across included modules.
	if includedITA := getAttr(root, "input-type-annotations"); includedITA != "" && includedITA != validationUnspecified {
		mainITA := c.stylesheet.inputTypeAnnotations
		if mainITA != "" && mainITA != validationUnspecified && mainITA != includedITA {
			return staticError(errCodeXTSE0265,
				"conflicting input-type-annotations: main module has %q, included module has %q",
				mainITA, includedITA)
		}
		if mainITA == "" || mainITA == validationUnspecified {
			c.stylesheet.inputTypeAnnotations = includedITA
		}
	}

	// Evaluate static params and variables so they're available for shadow attributes.
	for child := range helium.Children(root) {
		elem, ok := child.(*helium.Element)
		if !ok || elem.URI() != lexicon.NamespaceXSLT {
			continue
		}
		ln := elem.LocalName()
		if ln != "param" && ln != "variable" {
			continue
		}
		if getAttr(elem, "static") == lexicon.ValueYes {
			name := getAttr(elem, "name")
			sel := getAttr(elem, "select")
			if name != "" && sel != "" {
				compiled, err := xpath3.NewCompiler().Compile(sel)
				if err == nil {
					eval := c.staticEvaluator(ctx)
					if len(c.staticVars) > 0 {
						eval = eval.Variables(c.staticVars)
					}
					result, err := eval.Evaluate(ctx, compiled, nil)
					if err == nil {
						if setErr := c.setStaticVarWithKind(ctx, name, ln, result.Sequence()); setErr != nil {
							return setErr
						}
					}
				}
			}
		}
	}

	// Compile non-import elements in document order, with includes
	// interleaved to preserve effective document order.
	for child := range helium.Children(root) {
		ce, ok := child.(*helium.Element)
		if !ok || ce.URI() != lexicon.NamespaceXSLT {
			continue
		}

		// Resolve shadow attributes before processing.
		if err := c.resolveShadowAttributes(ctx, ce); err != nil {
			return err
		}

		switch ce.LocalName() {
		case lexicon.XSLTElementImport, lexicon.XSLTElementNamespaceAlias:
			// Already processed in collectIncludeImports
		case lexicon.XSLTElementInclude:
			if err := c.compileIncludeTemplates(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementUsePackage:
			if err := c.compileUsePackage(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementTemplate:
			if err := c.compileTemplate(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementVariable:
			if err := c.compileGlobalVariable(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementParam:
			if err := c.compileGlobalParam(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementKey:
			if err := c.compileKey(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementOutput:
			if err := c.compileOutput(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementStripSpace:
			if err := c.compileSpaceHandling(ctx, ce, true); err != nil {
				return err
			}
		case lexicon.XSLTElementPreserveSpace:
			if err := c.compileSpaceHandling(ctx, ce, false); err != nil {
				return err
			}
		case lexicon.XSLTElementFunction:
			if err := c.compileFunction(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementDecimalFormat:
			if err := c.compileDecimalFormat(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementMode:
			if err := c.compileMode(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementImportSchema:
			if err := c.compileImportSchema(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementAccumulator:
			if err := c.compileAccumulator(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementAttributeSet:
			if err := c.compileAttributeSet(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementGlobalContextItem:
			if err := c.compileGlobalContextItem(ctx, ce); err != nil {
				return err
			}
		case lexicon.XSLTElementCharacterMap:
			if err := c.compileCharacterMap(ctx, ce); err != nil {
				return err
			}
		}
	}
	return nil
}

// moduleEffectiveBaseURI computes the effective static base URI against which an
// included/imported stylesheet module's root use-when, globals, and templates
// resolve relative references (doc(), unparsed-text(), document(”), etc.).
//
// The MAIN module gets this treatment in compile() via resolveRootXMLBase, but
// external modules are loaded with c.baseURI set to the bare module URI and
// compiled directly (loadExternalStylesheet → compileTopLevel; or the two-phase
// include path). Without this, a root xml:base on an included/imported module
// is silently dropped and its globals resolve against the bare module URI
// instead of the declaration-site static base.
//
// For a module root that IS the document element, only its own xml:base is
// folded (via resolveRootXMLBase). For an EMBEDDED module root (one selected by
// a fragment identifier, whose parent is another element rather than the
// Document) the FULL xml:base ancestor chain — the embedded root's own xml:base,
// any wrapper element xml:base(s), and the document element's xml:base — is
// folded onto the module URI, because all of those ancestors lie above the
// module root and so are NOT visited by the descendant stylesheetBaseURI walk
// (which stops at the module root). The computed base is therefore the single
// authoritative base for the whole module: c.baseURI, the moduleDocs key, and
// the boundary that descendant walks stop at.
func moduleEffectiveBaseURI(root *helium.Element, uri string) string {
	if _, isDoc := root.Parent().(*helium.Document); isDoc {
		if xmlBase := getAttr(root, lexicon.QNameXMLBase); xmlBase != "" {
			return resolveRootXMLBase(uri, xmlBase)
		}
		return uri
	}
	// Embedded module root: fold every xml:base from the embedded root up
	// through the document element onto the module URI.
	var bases []string
	for cur := helium.Node(root); cur != nil; cur = cur.Parent() {
		elem, ok := cur.(*helium.Element)
		if !ok {
			continue
		}
		if xmlBase, ok := elem.GetAttributeNS("base", lexicon.NamespaceXML); ok && xmlBase != "" {
			bases = append(bases, xmlBase)
		}
	}
	return foldXMLBases(uri, bases)
}

// embeddedModuleRoot reports root as an EMBEDDED (fragment-selected) stylesheet
// module root — its parent is an element rather than the Document — otherwise
// nil. The result is assigned to c.moduleRoot so descendant xml:base walks stop
// at the embedded root (whose xml:base, plus any wrapper/document xml:base, is
// already folded into the module's effective base URI by moduleEffectiveBaseURI).
// A nil result restores the document-root boundary ("stop before the document
// element"), which is correct for a non-embedded module's own document tree.
func embeddedModuleRoot(root *helium.Element) *helium.Element {
	if _, isDoc := root.Parent().(*helium.Document); isDoc {
		return nil
	}
	return root
}

// stylesheetBaseURI folds the xml:base of n and its ancestors onto fallback,
// yielding the effective static base URI for a node WITHIN a stylesheet module.
// The walk stops at the module root (stopAt, the embedded stylesheet element for
// a fragment-selected module) — or, when stopAt is nil, before the document
// element — because the module root's xml:base (and everything above it) is
// already folded into fallback (the module's effective base URI), so re-applying
// it would double-count.
func stylesheetBaseURI(n helium.Node, fallback string, stopAt *helium.Element) string {
	var bases []string
	for cur := n; cur != nil; cur = cur.Parent() {
		elem, ok := cur.(*helium.Element)
		if !ok {
			continue
		}
		if stopAt != nil {
			if elem == stopAt {
				break
			}
		} else if p := elem.Parent(); p != nil {
			// Stop before the document element (stylesheet root). Its xml:base is
			// already factored into fallback, so including it again double-counts.
			if _, isDoc := p.(*helium.Document); isDoc {
				break
			}
		}
		if xmlBase, ok := elem.GetAttributeNS("base", lexicon.NamespaceXML); ok && xmlBase != "" {
			bases = append(bases, xmlBase)
		}
	}
	return foldXMLBases(fallback, bases)
}

// foldXMLBases folds a bottom-up-ordered slice of xml:base reference values onto
// base per RFC 3986. bases[0] is the nearest (deepest) xml:base, so the slice is
// applied from the topmost ancestor down and the deepest value wins.
func foldXMLBases(base string, bases []string) string {
	for _, v := range slices.Backward(bases) {
		if base == "" {
			base = v
			continue
		}
		resolved := helium.BuildURI(v, base)
		// Per RFC 3986, resolving a directory-denoting reference (one that ends
		// in '/', or whose last segment is "." / "..") yields a base that itself
		// names a directory and so ends in '/'. helium.BuildURI strips that
		// trailing slash, which would make a directory base such as the result of
		// xml:base=".." ("…/tests/fn/") indistinguishable from a stylesheet FILE
		// base ("…/tests/fn") — so a later sibling reference resolved through
		// documentBaseDir (path.Dir) would drop the real "fn" segment. Restore the
		// RFC-correct trailing slash so the directory form is preserved, inserting
		// it into the PATH (before any '?'/'#') so a directory base carrying a
		// query ("…/dir?v") becomes "…/dir/?v", never "…/dir?v/".
		if resolved != "" && refDenotesDirectory(v) {
			resolved = ensureDirSlash(resolved)
		}
		base = resolved
	}
	return base
}

// refDenotesDirectory reports whether a URI reference names a directory rather
// than a file, i.e. resolving it per RFC 3986 produces a base ending in '/'. A
// reference denotes a directory when its PATH portion ends in '/' or its last
// path segment is "." or ".." (a pure dot-segment that resolves to the
// containing directory). The query/fragment is excluded from the test, and a
// reference with an empty path portion (query-only "?v" or fragment-only
// "#frag") is NOT directory-denoting — path.Base("") == "." must not be
// mistaken for a "." dot-segment.
func refDenotesDirectory(ref string) bool {
	pathPart := ref
	if i := strings.IndexAny(pathPart, "?#"); i >= 0 {
		pathPart = pathPart[:i]
	}
	if pathPart == "" {
		return false
	}
	if strings.HasSuffix(pathPart, "/") {
		return true
	}
	last := path.Base(pathPart)
	return last == "." || last == ".."
}

// ensureDirSlash guarantees that the PATH portion of a resolved URI ends in
// '/', inserting the slash before any query ('?') or fragment ('#') component
// rather than at the very end. A directory base carrying a query ("…/dir?v")
// thus becomes "…/dir/?v", never "…/dir?v/" (which would corrupt the query and
// misplace the directory boundary). Idempotent when the path already ends in
// '/'.
func ensureDirSlash(uri string) string {
	pathEnd := len(uri)
	if i := strings.IndexAny(uri, "?#"); i >= 0 {
		pathEnd = i
	}
	if pathEnd > 0 && uri[pathEnd-1] == '/' {
		return uri
	}
	return uri[:pathEnd] + "/" + uri[pathEnd:]
}

func (c *compiler) loadExternalStylesheet(ctx context.Context, baseURI, href string, isImport bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Extract fragment identifier (e.g. "file.xml#embedded" → fragment="embedded").
	// The fragment selects an embedded stylesheet element by ID within the document.
	var fragment string
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		fragment = href[idx+1:]
		href = href[:idx]
	}

	// Resolve URI relative to base. An absolute-URI href is passed through
	// uncorrupted; a relative href is resolved against the base.
	uri, err := resolveModuleURI(href, baseURI)
	if err != nil {
		return err
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
	doc, err := c.loadModuleDoc(ctx, uri)
	if err != nil {
		return err
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
	// XTSE0165: a package must not import another package via xsl:import.
	if isImport && importedRoot.URI() == lexicon.NamespaceXSLT && importedRoot.LocalName() == lexicon.XSLTElementPackage {
		return staticError(errCodeXTSE0165,
			"cannot import xsl:package %q (use xsl:use-package instead)", uri)
	}
	// If the root is not in the XSLT namespace, check for simplified stylesheet
	if importedRoot.URI() != lexicon.NamespaceXSLT {
		if _, ok := importedRoot.GetAttributeNS("version", lexicon.NamespaceXSLT); ok {
			// Simplified stylesheet — compile as a single template matching "/"
			simplified, err := compileSimplified(ctx, doc, importedRoot, &compileConfig{
				baseURI:               uri,
				resolver:              c.resolver,
				packageResolver:       c.packageResolver,
				maxResourceBytes:      c.maxResourceBytes,
				allowExternalEntities: c.allowExternalEntities,
				parser:                c.parser,
			})
			if err != nil {
				return err
			}
			// Carry over the simplified module's backwards-compatible expression set
			// so its expressions still evaluate in XPath 1.0 compatibility mode (the
			// module's xsl:version < 2.0 marks them compat during compileSimplified).
			for e := range simplified.compatExprs {
				c.markCompatExpr(e)
			}
			// Merge the simplified stylesheet's templates into ours
			for _, tmpl := range simplified.templates {
				tmpl.ImportPrec = c.importPrec
				tmpl.MinImportPrec = c.importPrec // simplified stylesheets have no sub-imports
				c.stylesheet.templates = append(c.stylesheet.templates, tmpl)
				mode := tmpl.Mode
				c.stylesheet.modeTemplates[mode] = append(c.stylesheet.modeTemplates[mode], tmpl)
			}
			// Bump import precedence so the importing module's templates
			// have higher precedence than the imported simplified stylesheet.
			if isImport {
				c.importPrec++
			}
			return nil
		}
		return staticError(errCodeXTSE0010, "imported document %q is not a stylesheet", uri)
	}

	// Fold the module root's xml:base chain into the effective static base URI so
	// this module's globals and templates resolve relative references against the
	// declaration-site base (matching the main module's compile() handling), not
	// the bare module URI. For an EMBEDDED (fragment-selected) root this folds the
	// embedded root's own xml:base plus any wrapper/document xml:base; moduleRoot
	// is set so descendant xml:base walks stop at the embedded root and don't
	// re-apply that chain. moduleDocs stays keyed on the unmodified uri too.
	c.baseURI = moduleEffectiveBaseURI(importedRoot, uri)
	savedModuleRoot := c.moduleRoot
	c.moduleRoot = embeddedModuleRoot(importedRoot)
	defer func() { c.moduleRoot = savedModuleRoot }()

	// Also cache the module document under its FOLDED effective base so doc('') /
	// document('') from within this module (whose templates compile under
	// c.baseURI) resolves to the module's own document rather than falling back to
	// the principal stylesheet. The bare-uri entry stored above is kept for other
	// lookups; when no xml:base applies the folded base equals uri and this is a
	// no-op.
	if c.baseURI != uri {
		c.stylesheet.moduleDocs[c.baseURI] = doc
	}

	// Check use-when on the imported/included stylesheet's root element. If
	// use-when evaluates to false, skip the entire module. moduleEffectiveBaseURI
	// already gives the module's full effective base (folding the embedded root's
	// xml:base and any wrapper/document xml:base for a fragment-selected root), so
	// c.baseURI is the correct base for the root use-when.
	if uw := getAttr(importedRoot, xslAttrUseWhen); uw != "" {
		savedUseWhenBase := c.baseURI
		c.baseURI = moduleEffectiveBaseURI(importedRoot, uri)
		include, err := c.evaluateUseWhen(ctx, uw)
		c.baseURI = savedUseWhenBase
		if err != nil {
			return err
		}
		if !include {
			return nil
		}
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

	// Save/restore effective version; a _version shadow attribute takes precedence
	// over the literal (§3.5.2), so resolve it first — mirroring the top-level root
	// and included modules — so an imported module whose computed version is < 2.0
	// enters backwards-compatible mode.
	savedVersion := c.effectiveVersion
	if _, hasShadow := importedRoot.GetAttribute("_version"); hasShadow {
		if err := c.resolveSingleShadowAttribute(ctx, importedRoot, paramVersion); err != nil {
			return err
		}
	}
	if ver := getAttr(importedRoot, "version"); ver != "" {
		c.effectiveVersion = ver
	}
	defer func() { c.effectiveVersion = savedVersion }()

	// XTSE0265: conflicting input-type-annotations across modules.
	// Error only when one module says "preserve" and another says "strip".
	// "unspecified" is compatible with either.
	if importedITA := getAttr(importedRoot, "input-type-annotations"); importedITA != "" && importedITA != validationUnspecified {
		mainITA := c.stylesheet.inputTypeAnnotations
		if mainITA != "" && mainITA != validationUnspecified && mainITA != importedITA {
			return staticError(errCodeXTSE0265,
				"conflicting input-type-annotations: main module has %q, imported module has %q",
				mainITA, importedITA)
		}
		// Propagate "strip" or "preserve" from imported module when the main
		// module's effective value is still "unspecified" (or unset).
		if mainITA == "" || mainITA == validationUnspecified {
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
		c.collectNamespaces(ctx, importedRoot)
		if err := c.compileTopLevel(ctx, importedRoot); err != nil {
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
		c.collectNamespaces(ctx, importedRoot)
		if err := c.compileTopLevel(ctx, importedRoot); err != nil {
			return err
		}
	}
	return nil
}

// compileSimplified compiles a simplified stylesheet (literal result element
// as root).
func compileSimplified(ctx context.Context, doc *helium.Document, root *helium.Element, cfg *compileConfig) (*Stylesheet, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// XTSE0150: simplified stylesheet must have xsl:version attribute
	if _, ok := root.GetAttributeNS("version", lexicon.NamespaceXSLT); !ok {
		return nil, staticError(errCodeXTSE0150,
			"simplified stylesheet (literal result element) must have an xsl:version attribute")
	}
	c := &compiler{
		stylesheet: &Stylesheet{
			version:          lexicon.XSLTVersion30,
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
	}

	if cfg != nil {
		c.baseURI = cfg.baseURI
		c.resolver = cfg.resolver
		c.packageResolver = cfg.packageResolver
		c.maxResourceBytes = cfg.maxResourceBytes
		c.parser = cfg.parser
	}
	c.stylesheet.maxResourceBytes = c.maxResourceBytes
	c.stylesheet.parser = c.parser

	c.collectNamespaces(ctx, root)

	inst, err := c.compileLiteralResultElement(ctx, root)
	if err != nil {
		return nil, err
	}

	tmpl := &template{
		Match: &pattern{
			source: "/",
			Alternatives: []*patternAlt{
				{
					expr:     xpath3.RootExpr{},
					priority: -0.5,
				},
			},
		},
		Body:    []instruction{inst},
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
		for child := range helium.Children(n) {
			elem, ok := child.(*helium.Element)
			if !ok {
				continue
			}
			if v, found := elem.GetAttribute("id"); found && v == id {
				return elem
			}
			if v, found := elem.GetAttributeNS("id", lexicon.NamespaceXML); found && v == id {
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
