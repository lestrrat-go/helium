package xslt3

import (
	"context"
	"fmt"
	"path/filepath"
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
				resolved := resolveSchemaURI(fields[i], baseURI)
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
			resolved := resolveSchemaURI(strings.TrimSpace(attr.Value()), baseURI)
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
		schemaDoc, err := helium.NewParser().Parse(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("parse source schema %q: %w", uri, err)
		}
		// Root the XSD compiler's base directory at the schema's location so
		// the schema's own relative xs:include/xs:import references resolve,
		// and route those nested loads through the invocation's resolver
		// (default-deny) instead of the xsd compiler's default os.Open.
		fsys := schemaResolverFS{ctx: ctx, load: ec.retrieveDocumentBytes, baseURI: uri}
		schema, err := xsd.NewCompiler().BaseDir(filepath.Dir(uri)).FS(fsys).Compile(ctx, schemaDoc)
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
