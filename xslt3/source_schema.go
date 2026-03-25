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

func loadSchemasFromSchemaLocation(ctx context.Context, doc *helium.Document) ([]*xsd.Schema, error) {
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
				resolved := resolveAgainstBaseURI(fields[i], baseURI)
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
			resolved := resolveAgainstBaseURI(strings.TrimSpace(attr.Value()), baseURI)
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
	for _, path := range paths {
		absPath := path
		if !filepath.IsAbs(absPath) && baseURI != "" {
			absPath = filepath.Join(filepath.Dir(baseURI), path)
		}
		schema, err := xsd.NewCompiler().CompileFile(ctx, absPath)
		if err != nil {
			return nil, fmt.Errorf("compile source schema %q: %w", absPath, err)
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
