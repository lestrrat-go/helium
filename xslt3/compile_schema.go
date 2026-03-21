package xslt3

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

func (c *compiler) compileImportSchema(elem *helium.Element) error {
	// Mark stylesheet as schema-aware whenever any xsl:import-schema is seen,
	// even if it is a namespace-only declaration with no schema to compile.
	c.stylesheet.schemaAware = true

	// Collect namespace declarations from the xsl:import-schema element itself
	// (e.g., xmlns:o="http://example.com/schema") so that XPath expressions in
	// the stylesheet can use prefix:local references to schema-namespace types.
	c.collectNamespaces(elem)

	declaredNS := getAttr(elem, "namespace")

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
		// XTSE0220: namespace attribute must match the schema's targetNamespace.
		if declaredNS != "" && schema.TargetNamespace() != declaredNS {
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
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		if childElem.LocalName() == "schema" && childElem.URI() == lexicon.XSD {
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
			ctx := context.Background()
			var inlineOpts []xsd.CompileOption
			if c.baseURI != "" {
				inlineOpts = append(inlineOpts, xsd.WithBaseDir(filepath.Dir(c.baseURI)))
			}
			schema, err := xsd.Compile(ctx, inlineDoc, inlineOpts...)
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
			if uri == lexicon.XSD {
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
		if uri, ok := nsBindings[prefix]; ok {
			if uri == lexicon.XSD {
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

// validateAsSequenceType checks compile-time validity of an as= SequenceType
// expression when schemas are imported. It detects schema-element(Q) and
// schema-attribute(Q) references and verifies that Q is declared in at least
// one imported schema. Raises XTSE0590 when a referenced element or attribute
// is not found.
//
// This covers the most common case where compile-time static errors arise:
// using schema-element() or schema-attribute() with an undeclared name.
func (c *compiler) validateAsSequenceType(as string, context string) error {
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
		return staticError("XPST0003", "%s: invalid 'as' type: %s", context, err)
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
	if idx := strings.IndexByte(qname, ':'); idx >= 0 {
		prefix := qname[:idx]
		loc := qname[idx+1:]
		uri, ok := nsBindings[prefix]
		if !ok {
			return loc, ""
		}
		return loc, uri
	}
	return qname, ""
}
