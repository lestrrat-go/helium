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
		schemaCompiler := xsd.NewCompiler().BaseDir(schemaCompileBaseDir(uri)).FS(fsys)
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
