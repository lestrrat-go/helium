package xslt3

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

// compileSchemaFromURI loads a schema document through the compiler's
// configured URIResolver (default-deny: with no resolver, the load is
// refused rather than falling back to os.ReadFile) and compiles it
// in-memory. The resolver provides the same secure-by-default and
// path-traversal sandboxing model that xsl:import/include stylesheet
// loads use.
func (c *compiler) compileSchemaFromURI(ctx context.Context, uri string) (*xsd.Schema, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := c.loadSchemaBytes(ctx, uri)
	if err != nil {
		return nil, err
	}
	// Imported XSD schemas are ALWAYS parsed XXE-blocked: the entity opt-in
	// (Compiler.AllowExternalEntities) does NOT extend to schema documents.
	// External DTDs / general entities in a schema are neither loaded nor
	// substituted regardless of the compiler's allowExternalEntities setting.
	//
	// Parse with the schema's own URI as the base so doc.URL() carries the
	// canonical location. The xsd compiler's Compile derives the circular-include
	// root key from doc.URL(), so a nested xs:include/xs:redefine that points back
	// at this schema (main -> inc -> main) is recognized as already-loaded instead
	// of being re-parsed into duplicate components.
	doc, err := parseExternalXML(ctx, c.parser, data, uri, false, nil, nil, c.maxResourceBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot parse schema %q: %w", uri, err)
	}
	// Preserve relative include/import resolution within the schema by
	// rooting the XSD compiler's base directory at the schema's location,
	// and route the schema's nested xs:include/xs:import/xs:redefine loads
	// through the same compile-time resolver (default-deny) instead of the
	// xsd compiler's default os.Open.
	//
	// Install a fatalErrorCounter ErrorHandler so fatal schema-construction
	// diagnostics (e.g. an unresolved referenced type, or a nested xs:import
	// that fails to load) are not discarded. Without it the xsd compiler
	// installs a recovery placeholder for the unresolved type and reports
	// success, silently producing an invalid schema. This mirrors the
	// inline-schema path in compileImportSchema.
	errCounter := &fatalErrorCounter{}
	fsys := schemaResolverFS{ctx: ctx, load: c.loadSchemaBytes}
	schemaCompiler := xsd.NewCompiler().
		ErrorHandler(errCounter).
		BaseDir(schemaCompileBaseDir(uri)).
		FS(fsys)
	if c.parser != nil {
		schemaCompiler = schemaCompiler.Parser(*c.parser)
	}
	schema, err := schemaCompiler.Compile(ctx, doc)
	// XTSE0220: the schema could not be constructed (e.g. an unresolved
	// referenced type or a nested xs:import miss). The xsd compiler now
	// reports this as ErrCompilationFailed (nil schema); the fatalErrorCounter
	// carries the diagnostic count for the message.
	if errors.Is(err, xsd.ErrCompilationFailed) {
		return nil, staticError(errCodeXTSE0220,
			"schema %q has %d schema construction error(s)", uri, errCounter.count.Load())
	}
	if err != nil {
		return nil, err
	}
	// Defensive: a fatal diagnostic without ErrCompilationFailed should not
	// happen, but keep the XTSE0220 guard so an invalid schema never leaks.
	if errCounter.count.Load() > 0 {
		return nil, staticError(errCodeXTSE0220,
			"schema %q has %d schema construction error(s)", uri, errCounter.count.Load())
	}
	return schema, nil
}

// loadResourceBytes loads an arbitrary resource (e.g. an opted-in external
// DTD/general entity referenced by a stylesheet module) through the compile-time
// URIResolver, preserving the default-deny policy (no resolver → refused, no
// os.Open fallback) and the per-resource read cap. It is the compile-time
// entity loader handed to parseExternalXML so that opted-in external entities go
// through the SAME resolver-mediated, bounded channel as the parent module.
func (c *compiler) loadResourceBytes(_ context.Context, uri string) ([]byte, error) {
	if c.resolver == nil {
		return nil, staticError(errCodeXTSE0165,
			"cannot load %q: no URIResolver configured (filesystem access is opt-in; set Compiler.URIResolver)", uri)
	}
	rc, err := c.resolver.Resolve(uri)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve %q: %w", uri, err)
	}
	return readCloserToBytes(rc, c.maxResourceBytes)
}

// loadSchemaBytes loads a nested-schema document referenced by
// xs:include/xs:import/xs:redefine through the compile-time URIResolver,
// preserving the default-deny policy (no resolver → refused, no os.Open
// fallback).
func (c *compiler) loadSchemaBytes(_ context.Context, uri string) ([]byte, error) {
	if c.resolver == nil {
		return nil, staticError(errCodeXTSE0165,
			"cannot load schema %q: no URIResolver configured (filesystem access is opt-in; set Compiler.URIResolver)", uri)
	}
	rc, err := c.resolver.Resolve(uri)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve schema %q: %w", uri, err)
	}
	data, err := readCloserToBytes(rc, c.maxResourceBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot read schema %q: %w", uri, err)
	}
	return data, nil
}

// fatalErrorCounter is an error handler that counts fatal errors during schema compilation.
type fatalErrorCounter struct {
	count atomic.Int32
}

func (f *fatalErrorCounter) Handle(_ context.Context, err error) {
	if le, ok := err.(helium.ErrorLeveler); ok && le.ErrorLevel() == helium.ErrorLevelFatal {
		f.count.Add(1)
	}
}

func (c *compiler) compileImportSchema(ctx context.Context, elem *helium.Element) error {
	// Mark stylesheet as schema-aware whenever any xsl:import-schema is seen,
	// even if it is a namespace-only declaration with no schema to compile.
	c.stylesheet.schemaAware = true

	// Collect namespace declarations from the xsl:import-schema element itself
	// (e.g., xmlns:o="http://example.com/schema") so that XPath expressions in
	// the stylesheet can use prefix:local references to schema-namespace types.
	c.collectNamespaces(ctx, elem)

	declaredNS := getAttr(elem, "namespace")

	schemaLoc := getAttr(elem, "schema-location")
	// Detect whether an inline xs:schema child is present.
	hasInlineSchema := false
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.LocalName() == "schema" && childElem.URI() == lexicon.NamespaceXSD {
			hasInlineSchema = true
			break
		}
	}

	// XTSE0215: it is a static error if an xsl:import-schema element that
	// contains an xs:schema element has a schema-location attribute.
	if schemaLoc != "" && hasInlineSchema {
		return staticError(errCodeXTSE0215,
			"xsl:import-schema has both schema-location attribute and inline xs:schema child")
	}

	// Compute the effective base URI for resolving this import-schema's
	// schema-location and the nested xs:include/xs:import/xs:redefine loads of
	// an inline schema, folding in an xml:base attribute on the
	// xsl:import-schema element (c.baseURI already accounts for xml:base on the
	// stylesheet root and includes). Without this, a nested import in an inline
	// schema would resolve against the wrong directory, silently fail to load,
	// and leave the referenced type unresolved.
	baseURI := c.baseURI
	if xmlBase, ok := elem.GetAttributeNS("base", lexicon.NamespaceXML); ok && xmlBase != "" {
		// Fold xml:base into the effective base using the URI-aware schema
		// resolver rather than helium.BuildURI. BuildURI filepath.Join's for the
		// file: scheme, collapsing a canonical "file:///tmp/styles/main.xsl" base
		// to a bare local path and dropping the scheme/authority; the resolver
		// preserves file: and other no-authority URI spellings via RFC 3986 so
		// the downstream schema-location resolution and resolver FS see the same
		// canonical URI.
		resolved, err := resolveSchemaURI(xmlBase, c.baseURI)
		if err != nil {
			return fmt.Errorf("xsl:import-schema: cannot resolve xml:base %q against base %q: %w", xmlBase, c.baseURI, err)
		}
		// An xml:base ending in "/" denotes a directory; preserve that trailing
		// slash (filepath.Join in the local resolver branch strips it) so the
		// downstream schemaCompileBaseDir's filepath.Dir keeps the directory
		// segment instead of treating it as a filename to discard. This matches
		// the directory semantics helium.BuildURI used to provide.
		if strings.HasSuffix(xmlBase, "/") && !strings.HasSuffix(resolved, "/") {
			resolved += "/"
		}
		baseURI = resolved
	}

	if schemaLoc != "" {
		// File-backed schema. Resolve the schema-location against the
		// stylesheet base URI using RFC 3986 URI semantics when the base is a
		// URL (so the authority survives and nested includes can be recovered),
		// falling back to filepath joins for local filesystem bases.
		uri := schemaLoc
		if baseURI != "" {
			resolved, err := resolveSchemaURI(schemaLoc, baseURI)
			if err != nil {
				return fmt.Errorf("xsl:import-schema: cannot resolve schema-location %q against base %q: %w", schemaLoc, baseURI, err)
			}
			uri = resolved
		}

		schema, err := c.compileSchemaFromURI(ctx, uri)
		if err != nil {
			// A genuine "schema not found / not applicable" error may fall back
			// to a pre-compiled import schema registered for the namespace. But a
			// fatal schema-load (resource-limit breach, path escape, or
			// import-depth overflow) must NOT be papered over by the fallback —
			// doing so would let an over-cap or path-traversal schema-location
			// silently succeed, defeating the guard. The single classifier
			// recognizes every fatal-load condition (including a nested path
			// escape, which surfaces as a plain xsd sentinel that an interface-only
			// check would miss); propagate those, preserving the sentinel for
			// errors.Is. Decided BEFORE findImportSchema so no fatal load can fall
			// through to the precompiled-schema path.
			if isFatalSchemaLoadError(err) {
				return fmt.Errorf("xsl:import-schema: cannot compile %q: %w", uri, err)
			}
			// File not found — try pre-compiled import schemas by namespace.
			if declaredNS != "" {
				if resolved := c.findImportSchema(ctx, declaredNS); resolved != nil {
					c.stylesheet.schemas = append(c.stylesheet.schemas, resolved)
					return nil
				}
			}
			return fmt.Errorf("xsl:import-schema: cannot compile %q: %w", uri, err)
		}
		// XTSE0220: namespace attribute must match the schema's targetNamespace.
		if declaredNS != "" && schema.TargetNamespace() != declaredNS {
			// Try pre-compiled import schemas by namespace before erroring.
			if resolved := c.findImportSchema(ctx, declaredNS); resolved != nil {
				c.stylesheet.schemas = append(c.stylesheet.schemas, resolved)
				return nil
			}
			return staticError(errCodeXTSE0220,
				"xsl:import-schema namespace %q does not match schema targetNamespace %q",
				declaredNS, schema.TargetNamespace())
		}
		// XTSE0220: when schema has a non-empty targetNamespace, the namespace
		// attribute is required (per XSLT spec section 3.13).
		if declaredNS == "" && schema.TargetNamespace() != "" {
			return staticError(errCodeXTSE0220,
				"xsl:import-schema: namespace attribute is required when schema has targetNamespace %q",
				schema.TargetNamespace())
		}
		c.stylesheet.schemas = append(c.stylesheet.schemas, schema)
		return nil
	}

	// Look for inline xs:schema child
	for child := range helium.Children(elem) {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.LocalName() == "schema" && childElem.URI() == lexicon.NamespaceXSD {
			inlineDoc := helium.NewDefaultDocument()
			copied, err := helium.CopyNode(childElem, inlineDoc)
			if err != nil {
				return fmt.Errorf("xsl:import-schema: cannot copy inline schema: %w", err)
			}
			// Propagate in-scope namespace declarations from ancestor
			// elements (e.g., xmlns:foo on xsl:stylesheet) to the
			// inline schema element so that prefix references like
			// ref="foo:type" can be resolved by the XSD compiler.
			if copiedElem, ok := copied.(*helium.Element); ok {
				for prefix, uri := range c.nsBindings {
					if prefix != "" && !hasNSDeclForPrefix(copiedElem, prefix) {
						_ = copiedElem.DeclareNamespace(prefix, uri)
					}
				}
			}
			if err := inlineDoc.AddChild(copied); err != nil {
				return fmt.Errorf("xsl:import-schema: cannot build inline schema doc: %w", err)
			}
			errCounter := &fatalErrorCounter{}
			// Route the inline schema's nested xs:include/xs:import/xs:redefine
			// loads through the SAME compile-time resolver (default-deny) used by
			// the schema-location path, instead of the xsd compiler's default
			// os.Open. The inline schema bytes themselves are already in-memory
			// (inlineDoc); only their nested references reach the resolver FS,
			// rooted at the import-schema element's base URI.
			fsys := schemaResolverFS{ctx: ctx, load: c.loadSchemaBytes}
			compiler := xsd.NewCompiler().ErrorHandler(errCounter).FS(fsys)
			if baseURI != "" {
				compiler = compiler.BaseDir(schemaCompileBaseDir(baseURI))
			}
			if c.parser != nil {
				compiler = compiler.Parser(*c.parser)
			}
			schema, err := compiler.Compile(ctx, inlineDoc)
			// XTSE0220: the synthetic schema document does not satisfy XSD
			// constraints (e.g., duplicate global declarations). The xsd
			// compiler now signals this via ErrCompilationFailed (nil schema).
			if errors.Is(err, xsd.ErrCompilationFailed) {
				return staticError(errCodeXTSE0220,
					"xsl:import-schema: inline schema has %d schema construction error(s)", errCounter.count.Load())
			}
			if err != nil {
				return fmt.Errorf("xsl:import-schema: cannot compile inline schema: %w", err)
			}
			// Defensive guard: a fatal diagnostic without ErrCompilationFailed
			// should not occur, but never let an invalid schema leak through.
			if errCounter.count.Load() > 0 {
				return staticError(errCodeXTSE0220,
					"xsl:import-schema: inline schema has %d schema construction error(s)", errCounter.count.Load())
			}
			// XTSE0215: namespace attribute must not conflict with the
			// targetNamespace of the contained inline schema.
			if declaredNS != "" && schema.TargetNamespace() != declaredNS {
				return staticError(errCodeXTSE0215,
					"xsl:import-schema namespace %q conflicts with inline schema targetNamespace %q",
					declaredNS, schema.TargetNamespace())
			}
			c.stylesheet.schemas = append(c.stylesheet.schemas, schema)
			return nil
		}
	}

	// Namespace-only declaration — try pre-compiled import schemas by namespace.
	if declaredNS != "" {
		if resolved := c.findImportSchema(ctx, declaredNS); resolved != nil {
			c.stylesheet.schemas = append(c.stylesheet.schemas, resolved)
		}
	}
	return nil
}

// findImportSchema looks up a pre-compiled schema by target namespace from
// the import schemas provided at compile time.
func (c *compiler) findImportSchema(_ context.Context, ns string) *xsd.Schema {
	for _, s := range c.importSchemas {
		if s.TargetNamespace() == ns {
			return s
		}
	}
	return nil
}

// isTypedStrict returns true if the typed attribute value indicates strict
// type checking. Per the XSLT 3.0 spec, "strict", "yes", "true", and "1"
// all enable strict typing.
func isTypedStrict(typed string) bool {
	if typed == validationStrict {
		return true
	}
	v, ok := parseXSDBool(typed)
	return ok && v
}

// checkTypedModePatterns validates XTSE3105: for modes declared with
// typed="strict", every template match pattern whose first step uses an axis
// with principal node kind Element and whose NodeTest is an EQName must
// correspond to a global element declaration in the imported schemas.
func checkTypedModePatterns(ss *Stylesheet) error {
	if len(ss.schemas) == 0 {
		return nil
	}
	reg := &schemaRegistry{schemas: ss.schemas}
	for modeName, md := range ss.modeDefs {
		if !isTypedStrict(md.Typed) {
			continue
		}
		// Map modeDefs key to modeTemplates key: "#default" → ""
		templateKey := modeName
		if templateKey == modeDefault {
			templateKey = ""
		}
		templates := ss.modeTemplates[templateKey]
		// Also include #all templates
		if templateKey != modeAll {
			templates = append(templates, ss.modeTemplates[modeAll]...)
		}
		for _, tmpl := range templates {
			if tmpl.Match == nil {
				continue
			}
			if err := checkPatternAgainstSchema(tmpl.Match, reg); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkPatternAgainstSchema checks all alternatives in a pattern for
// element name tests that don't exist in the imported schemas.
func checkPatternAgainstSchema(p *pattern, reg *schemaRegistry) error {
	for _, alt := range p.Alternatives {
		if err := checkExprAgainstSchema(alt.expr, reg, p.xpathDefaultNS, p.nsBindings); err != nil {
			return err
		}
	}
	return nil
}

// checkExprAgainstSchema walks an expression AST and checks the first step
// of each relative path expression for element name tests against imported
// schemas. Per the XSLT 3.0 spec, XTSE3105 only applies to the first
// StepExprP of a RelativePathExprP.
func checkExprAgainstSchema(expr xpath3.Expr, reg *schemaRegistry, xpathDefaultNS string, nsBindings map[string]string) error {
	switch e := expr.(type) {
	case xpath3.LocationPath:
		return checkStepsAgainstSchema(e.Steps, reg, xpathDefaultNS, nsBindings)
	case *xpath3.LocationPath:
		return checkStepsAgainstSchema(e.Steps, reg, xpathDefaultNS, nsBindings)
	case xpath3.PathStepExpr:
		// For path/step, check the leftmost expression's first step
		return checkExprAgainstSchema(e.Left, reg, xpathDefaultNS, nsBindings)
	case *xpath3.PathStepExpr:
		return checkExprAgainstSchema(e.Left, reg, xpathDefaultNS, nsBindings)
	case xpath3.FilterExpr:
		return checkExprAgainstSchema(e.Expr, reg, xpathDefaultNS, nsBindings)
	case *xpath3.FilterExpr:
		return checkExprAgainstSchema(e.Expr, reg, xpathDefaultNS, nsBindings)
	case xpath3.UnionExpr:
		if err := checkExprAgainstSchema(e.Left, reg, xpathDefaultNS, nsBindings); err != nil {
			return err
		}
		return checkExprAgainstSchema(e.Right, reg, xpathDefaultNS, nsBindings)
	case *xpath3.UnionExpr:
		if err := checkExprAgainstSchema(e.Left, reg, xpathDefaultNS, nsBindings); err != nil {
			return err
		}
		return checkExprAgainstSchema(e.Right, reg, xpathDefaultNS, nsBindings)
	}
	return nil
}

// checkStepsAgainstSchema applies the XTSE3105 element-name check to a location
// path's step sequence. Per the spec the check targets the first StepExprP whose
// axis has principal node kind Element and whose NodeTest is an EQName. Leading
// steps that are not element-name tests — the synthetic descendant-or-self::node()
// produced by the "//" abbreviation, a root/document-node step, an attribute axis,
// or a kind test such as element(*) — are not EQName element-axis steps and are
// skipped so the first genuine element NameTest (e.g. the "foo" in "//foo") is the
// one checked. Once such a step is found the check is applied to it alone; a later
// step in the same path (e.g. the "b" in "a/b") is not an additional XTSE3105 site.
func checkStepsAgainstSchema(steps []xpath3.Step, reg *schemaRegistry, xpathDefaultNS string, nsBindings map[string]string) error {
	for _, step := range steps {
		// Skip steps on axes whose principal node kind is not Element.
		if step.Axis == xpath3.AxisAttribute || step.Axis == xpath3.AxisNamespace {
			continue
		}
		// Only an EQName NodeTest (NameTest, non-wildcard) is an XTSE3105 site;
		// kind tests, node() (from "//"), and wildcards are skipped.
		nt, ok := step.NodeTest.(xpath3.NameTest)
		if !ok || nt.Local == "*" || nt.Prefix == "*" {
			continue
		}
		return checkStepAgainstSchema(step, reg, xpathDefaultNS, nsBindings)
	}
	return nil
}

// checkStepAgainstSchema checks if a step's nameTest is a declared element.
func checkStepAgainstSchema(step xpath3.Step, reg *schemaRegistry, xpathDefaultNS string, nsBindings map[string]string) error {
	// Only check axes whose principal node kind is Element
	if step.Axis == xpath3.AxisAttribute || step.Axis == xpath3.AxisNamespace {
		return nil
	}
	nt, ok := step.NodeTest.(xpath3.NameTest)
	if !ok {
		return nil
	}
	// Wildcard tests match anything
	if nt.Local == "*" {
		return nil
	}
	if nt.Prefix == "*" {
		return nil
	}
	// Determine element namespace
	ns := nt.URI
	if ns == "" && nt.Prefix != "" {
		// Resolve the prefix the same way runtime pattern matching does
		// (execContext.resolvePrefix during pattern matching): the pattern's
		// lexical snapshot first, then the predeclared XPath prefixes. Using only
		// nsBindings here would diverge from runtime — e.g. a predeclared 'math'
		// prefix would resolve to no-namespace and spuriously trip XTSE3105.
		ns = resolvePatternPrefix(nsBindings, nt.Prefix)
	}
	if ns == "" && nt.Prefix == "" {
		ns = xpathDefaultNS
	}
	// Check if element is declared in imported schemas
	if _, found := reg.LookupElement(nt.Local, ns); !found {
		return staticError(errCodeXTSE3105,
			"match pattern uses element name %q which is not declared in any imported schema (mode has typed=\"strict\")",
			nt.Local)
	}
	return nil
}

// hasNSDeclForPrefix checks if an element already declares a namespace for the given prefix.
func hasNSDeclForPrefix(elem *helium.Element, prefix string) bool {
	for _, ns := range elem.Namespaces() {
		if ns.Prefix() == prefix {
			return true
		}
	}
	return false
}

// resolveXSDTypeName normalizes a QName type reference (e.g., "xs:ID",
// "xsd:integer", or "Q{http://www.w3.org/2001/XMLSchema}ID") to the
// canonical "xs:..." prefix form used by xpath3 constants. A bare (unprefixed)
// name is resolved against no namespace.
func resolveXSDTypeName(qname string, nsBindings map[string]string) string {
	return resolveXSDTypeNameNS(qname, nsBindings, "", false)
}

// resolveXSDTypeNameNS is like resolveXSDTypeName but resolves a bare
// (unprefixed) non-builtin type name against the in-scope xpath-default-namespace
// when one is in effect. Per XSLT/XPath, an unprefixed type name in an @as/@type
// attribute takes the default element/type namespace; a built-in xs: name and a
// prefixed/EQName form are unaffected.
func resolveXSDTypeNameNS(qname string, nsBindings map[string]string, xpathDefaultNS string, hasXPathDefaultNS bool) string {
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
			if uri == lexicon.NamespaceXSD {
				return "xs:" + local
			}
			return qname
		}
	}
	// Handle prefix:local
	if prefix, local, ok := strings.Cut(qname, ":"); ok {
		if prefix == "xs" || prefix == "xsd" {
			return "xs:" + local
		}
		if uri, ok := nsBindings[prefix]; ok {
			if uri == lexicon.NamespaceXSD {
				return "xs:" + local
			}
			// User-defined type: resolve to Q{ns}local canonical form.
			return xpath3.QAnnotation(uri, local)
		}
	}
	// Bare name (no prefix, no Q{} wrapper): a user-defined type taking the
	// in-scope default element/type namespace (xpath-default-namespace) when one
	// is in effect, else no namespace. Use Q{} annotation form to match
	// xsdTypeNameFromDef.
	if hasXPathDefaultNS && xpathDefaultNS != "" {
		return xpath3.QAnnotation(xpathDefaultNS, qname)
	}
	return "Q{}" + qname
}

// schemaDeclsForValidation returns the in-scope schema declarations for
// compile-time static checking, or nil when the stylesheet is not schema-aware.
// When non-nil it lets the xpath3 static check permit an unprefixed atomic/schema
// type name (resolved against the default element/type namespace or as a
// no-namespace schema type) instead of raising XPST0081.
func (c *compiler) schemaDeclsForValidation() xpath3.SchemaDeclarations {
	if !c.stylesheet.schemaAware || len(c.stylesheet.schemas) == 0 {
		return nil
	}
	return &schemaRegistry{schemas: c.stylesheet.schemas}
}

// validateAsSequenceType checks compile-time validity of an as= sequenceType
// expression when schemas are imported. It detects schema-element(Q) and
// schema-attribute(Q) references and verifies that Q is declared in at least
// one imported schema. Raises XTSE0590 when a referenced element or attribute
// is not found. An unprefixed schema-element()/schema-attribute() name resolves
// against the in-scope xpath-default-namespace.
//
// This covers the most common case where compile-time static errors arise:
// using schema-element() or schema-attribute() with an undeclared name.
func (c *compiler) validateAsSequenceType(ctx context.Context, as string, context string) error {
	return c.validateAsSequenceTypeWithNS(ctx, as, context, c.nsBindings, c.xpathDefaultNS, c.hasXPathDefaultNS)
}

// validateAsSequenceTypeWithNS is like validateAsSequenceType but resolves
// schema-element()/schema-attribute() QNames against the supplied namespace
// bindings instead of the mutable compiler-wide c.nsBindings, and resolves an
// UNPREFIXED schema-element()/schema-attribute() name against the supplied
// xpath-default-namespace when hasXPathDefaultNS is set. This lets declarations
// such as xsl:global-context-item validate their @as type against the exact
// namespace context in scope at the declaration element itself — matching the
// runtime resolveSchemaQName logic so an unprefixed name with an in-scope
// xpath-default-namespace is not wrongly resolved to {}name at compile time.
func (c *compiler) validateAsSequenceTypeWithNS(_ context.Context, as string, context string, nsBindings map[string]string, xpathDefaultNS string, hasXPathDefaultNS bool) error {
	if as == "" {
		return nil
	}
	// Validate syntax of the sequence type expression (catches errors
	// like function(xs:integer) missing "as ReturnType").
	if _, err := xpath3.ParseSequenceType(as); err != nil {
		var xpErr *xpath3.XPathError
		if errors.As(err, &xpErr) {
			return staticError(xpErr.Code, "%s: invalid 'as' type: %s", context, err)
		}
		return staticError(errCodeXPST0003, "%s: invalid 'as' type: %s", context, err)
	}
	if !c.stylesheet.schemaAware || len(c.stylesheet.schemas) == 0 {
		return nil
	}

	reg := &schemaRegistry{schemas: c.stylesheet.schemas}
	resolve := nsResolverFromMap(nsBindings)

	// Check schema-element(Q) and schema-attribute(Q) references.
	for _, kind := range []string{"schema-element", "schema-attribute"} {
		// schema-element name arguments take the default element namespace; a
		// schema-attribute name argument never does (an unprefixed attribute name
		// is in no namespace).
		nameKind := qnameElementName
		if kind == "schema-attribute" {
			nameKind = qnameAttributeName
		}
		search := kind + "("
		s := as
		for {
			idx := strings.Index(s, search)
			if idx < 0 {
				break
			}
			s = s[idx+len(search):]
			// find closing paren, skipping whitespace
			end := strings.IndexByte(s, ')')
			if end < 0 {
				break
			}
			qname := strings.TrimSpace(s[:end])
			s = s[end+1:]
			if qname == "" {
				continue
			}
			// Resolve the QName to (local, ns) using the single unified resolver
			// with the position-specific rule for this kind.
			local, ns := resolveSequenceTypeQName(qname, nameKind, resolve, xpathDefaultNS, hasXPathDefaultNS)
			if local == "" {
				continue
			}
			// Verify against imported schemas.
			var found bool
			if kind == "schema-element" {
				_, found = reg.LookupElement(local, ns)
			} else {
				_, found = reg.LookupAttribute(local, ns)
			}
			if !found {
				return staticError(errCodeXTSE0590,
					"%s: as=\"%s\" references undeclared %s({%s}%s)",
					context, as, kind, ns, local)
			}
		}
	}
	return nil
}
