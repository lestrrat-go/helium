package xslt3

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xsd"
)

// loadSchemasFromSchemaLocation loads schemas referenced by the source
// document's xsi:schemaLocation / xsi:noNamespaceSchemaLocation attributes.
// Schema bytes are fetched through the transformation's configured
// URIResolver / HTTPClient (default-deny: with nothing configured the load
// is refused) rather than via a raw os.ReadFile, so runtime schema loads
// obey the same secure-by-default policy as fn:doc and document().
func (ec *execContext) loadSchemasFromSchemaLocation(ctx context.Context, doc *helium.Document) ([]*xsd.Schema, error) {
	root := doc.DocumentElement()
	if root == nil {
		return nil, nil
	}

	baseURI := doc.URL()
	seen := make(map[string]struct{})
	var paths []string
	for _, attr := range root.Attributes() {
		if attr.URI() != lexicon.NamespaceXSI {
			continue
		}
		switch attr.LocalName() {
		case "schemaLocation":
			fields := strings.Fields(attr.Value())
			for i := 1; i < len(fields); i += 2 {
				resolved, err := resolveSchemaURI(fields[i], baseURI)
				if err != nil {
					return nil, fmt.Errorf("resolve source schema-location %q against base %q: %w", fields[i], baseURI, err)
				}
				if resolved == "" {
					continue
				}
				if _, ok := seen[resolved]; ok {
					continue
				}
				seen[resolved] = struct{}{}
				paths = append(paths, resolved)
			}
		case "noNamespaceSchemaLocation":
			ref := strings.TrimSpace(attr.Value())
			resolved, err := resolveSchemaURI(ref, baseURI)
			if err != nil {
				return nil, fmt.Errorf("resolve source schema-location %q against base %q: %w", ref, baseURI, err)
			}
			if resolved == "" {
				continue
			}
			if _, ok := seen[resolved]; ok {
				continue
			}
			seen[resolved] = struct{}{}
			paths = append(paths, resolved)
		}
	}

	if len(paths) == 0 {
		return nil, nil
	}

	schemas := make([]*xsd.Schema, 0, len(paths))
	for _, uri := range paths {
		data, err := ec.retrieveDocumentBytes(ctx, uri)
		if err != nil {
			return nil, fmt.Errorf("load source schema %q: %w", uri, err)
		}
		// Parse with the schema's own URI as the base so doc.URL() carries the
		// canonical location, mirroring compileSchemaFromURI. The xsd compiler's
		// Compile derives the circular-include root key from doc.URL(); without a
		// base, a local-path schema URI leaves doc.URL() empty and a nested
		// xs:include/xs:redefine pointing back at this schema (main -> inc -> main)
		// is re-parsed into duplicate components instead of being skipped.
		schemaDoc, err := secureXMLParser(ec.injectedParser(), uri, ec.resourceLimit()).Parse(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("parse source schema %q: %w", uri, err)
		}
		// Root the XSD compiler's base directory at the schema's location so
		// the schema's own relative xs:include/xs:import references resolve,
		// and route those nested loads through the invocation's resolver
		// (default-deny) instead of the xsd compiler's default os.Open.
		fsys := schemaResolverFS{ctx: ctx, load: ec.retrieveDocumentBytes}
		schemaCompiler := xsd.NewCompiler().DefaultVersion(xsd.Version11).BaseDir(schemaCompileBaseDir(uri)).FS(fsys)
		if p := ec.injectedParser(); p != nil {
			schemaCompiler = schemaCompiler.Parser(*p)
		}
		schema, err := schemaCompiler.Compile(ctx, schemaDoc)
		if err != nil {
			return nil, fmt.Errorf("compile source schema %q: %w", uri, err)
		}
		schemas = append(schemas, schema)
	}
	return schemas, nil
}

// sourceHasSchemaLocation reports whether the source document's root element
// carries an xsi:schemaLocation or xsi:noNamespaceSchemaLocation attribute,
// i.e. whether loadSchemasFromSchemaLocation could discover a schema that types
// the source even when the stylesheet itself is not schema-aware. It is a cheap
// root-attribute scan (no resolution or compilation) used to decide, before the
// runtime schema registry is built, whether the strip copy must carry an
// original->copy node map so source validation annotations can be remapped onto
// the copy the transform navigates.
func sourceHasSchemaLocation(doc *helium.Document) bool {
	if doc == nil {
		return false
	}
	root := doc.DocumentElement()
	if root == nil {
		return false
	}
	for _, attr := range root.Attributes() {
		if attr.URI() != lexicon.NamespaceXSI {
			continue
		}
		switch attr.LocalName() {
		case "schemaLocation", "noNamespaceSchemaLocation":
			return true
		}
	}
	return false
}

// collectPackageSchemas returns the schemas of ss followed by the schemas of
// every package it uses, transitively, deduped by schema-object identity. The
// main stylesheet's own schemas come first so its declarations take precedence
// on a name collision. Identity (not target-namespace) dedup is deliberate:
// distinct packages may each import a no-namespace schema declaring different
// types, and both sets must stay resolvable in the runtime registry.
func collectPackageSchemas(ss *Stylesheet) []*xsd.Schema {
	return appendPackageSchemas(nil, ss, make(map[*xsd.Schema]struct{}), make(map[*Stylesheet]struct{}))
}

// appendPackageSchemas appends pkg's schemas (then its used packages',
// transitively) to out, skipping schema objects already in seen and packages
// already in visited.
func appendPackageSchemas(out []*xsd.Schema, pkg *Stylesheet, seen map[*xsd.Schema]struct{}, visited map[*Stylesheet]struct{}) []*xsd.Schema {
	if pkg == nil {
		return out
	}
	if _, ok := visited[pkg]; ok {
		return out
	}
	visited[pkg] = struct{}{}
	for _, schema := range pkg.schemas {
		if schema == nil {
			continue
		}
		if _, ok := seen[schema]; ok {
			continue
		}
		seen[schema] = struct{}{}
		out = append(out, schema)
	}
	for _, sub := range pkg.usedPackages {
		out = appendPackageSchemas(out, sub, seen, visited)
	}
	return out
}

func mergeRuntimeSchemas(existing []*xsd.Schema, extra []*xsd.Schema) []*xsd.Schema {
	if len(extra) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing))
	merged := make([]*xsd.Schema, 0, len(existing)+len(extra))
	for _, schema := range existing {
		if schema == nil {
			continue
		}
		merged = append(merged, schema)
		seen[schema.TargetNamespace()] = struct{}{}
	}
	for _, schema := range extra {
		if schema == nil {
			continue
		}
		if _, ok := seen[schema.TargetNamespace()]; ok {
			continue
		}
		merged = append(merged, schema)
		seen[schema.TargetNamespace()] = struct{}{}
	}
	return merged
}
