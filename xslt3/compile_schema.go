package xslt3

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

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

	if schemaLoc != "" {
		// File-backed schema
		uri := schemaLoc
		if c.baseURI != "" && !strings.Contains(schemaLoc, "://") && !filepath.IsAbs(schemaLoc) {
			baseDir := filepath.Dir(c.baseURI)
			uri = filepath.Join(baseDir, schemaLoc)
		}

		schema, err := xsd.NewCompiler().CompileFile(ctx, uri)
		if err != nil {
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
			compiler := xsd.NewCompiler().ErrorHandler(errCounter)
			if c.baseURI != "" {
				compiler = compiler.BaseDir(filepath.Dir(c.baseURI))
			}
			schema, err := compiler.Compile(ctx, inlineDoc)
			if err != nil {
				return fmt.Errorf("xsl:import-schema: cannot compile inline schema: %w", err)
			}
			// XTSE0220: the synthetic schema document does not satisfy XSD constraints
			// (e.g., duplicate global declarations).
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
func (c *compiler) findImportSchema(ctx context.Context, ns string) *xsd.Schema { //nolint:unparam // ctx threaded through for API consistency
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
		if len(e.Steps) > 0 {
			return checkStepAgainstSchema(e.Steps[0], reg, xpathDefaultNS, nsBindings)
		}
	case *xpath3.LocationPath:
		if len(e.Steps) > 0 {
			return checkStepAgainstSchema(e.Steps[0], reg, xpathDefaultNS, nsBindings)
		}
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
		// Resolve prefix using compile-time namespace bindings
		ns = nsBindings[nt.Prefix]
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
	// Bare name (no prefix, no Q{} wrapper): treat as user-defined type
	// in no namespace. Use Q{} annotation form to match xsdTypeNameFromDef.
	return "Q{}" + qname
}

// validateAsSequenceType checks compile-time validity of an as= sequenceType
// expression when schemas are imported. It detects schema-element(Q) and
// schema-attribute(Q) references and verifies that Q is declared in at least
// one imported schema. Raises XTSE0590 when a referenced element or attribute
// is not found.
//
// This covers the most common case where compile-time static errors arise:
// using schema-element() or schema-attribute() with an undeclared name.
func (c *compiler) validateAsSequenceType(ctx context.Context, as string, context string) error { //nolint:unparam // ctx threaded through for API consistency
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

	// Check schema-element(Q) and schema-attribute(Q) references.
	for _, kind := range []string{"schema-element", "schema-attribute"} {
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
			// Resolve the QName to (local, ns) using current namespace bindings.
			local, ns := resolveQNameToLocalNS(qname, c.nsBindings)
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

// resolveQNameToLocalNS resolves a QName (prefix:local or NCName) using the
// given namespace bindings and returns (local, ns). For bare NCNames with no
// prefix the ns is the empty string (no default namespace for type references).
func resolveQNameToLocalNS(qname string, nsBindings map[string]string) (local, ns string) {
	qname = strings.TrimSpace(qname)
	// EQName form: Q{uri}local
	if strings.HasPrefix(qname, "Q{") {
		closeIdx := strings.IndexByte(qname, '}')
		if closeIdx > 0 {
			return qname[closeIdx+1:], qname[2:closeIdx]
		}
		return "", ""
	}
	if prefix, loc, ok := strings.Cut(qname, ":"); ok {
		uri, ok := nsBindings[prefix]
		if !ok {
			return loc, ""
		}
		return loc, uri
	}
	return qname, ""
}
