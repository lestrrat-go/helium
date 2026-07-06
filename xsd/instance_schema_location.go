package xsd

import (
	"context"
	"io/fs"
	"maps"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

type instanceSchemaHint struct {
	ns       string
	location string
}

func schemaWithInstanceHints(ctx context.Context, base *Schema, doc *helium.Document) *Schema {
	if base == nil || doc == nil || base.loaderFS == nil {
		return base
	}
	if _, denyAll := base.loaderFS.(iofs.DenyAll); denyAll {
		return base
	}

	hints := collectInstanceSchemaHints(doc)
	if len(hints) == 0 {
		return base
	}

	instanceBase := schemaBaseDir(doc.URL())
	if doc.URL() == "" {
		instanceBase = base.loaderBaseDir
	}

	var out *Schema
	loaded := map[string]struct{}{}
	for _, hint := range hints {
		path, err := validateSchemaPath(instanceBase, hint.location)
		if err != nil {
			continue
		}
		if _, seen := loaded[path]; seen {
			continue
		}
		loaded[path] = struct{}{}

		hinted := compileInstanceHintSchema(ctx, base, path)
		if hinted == nil || hinted.targetNamespace != hint.ns {
			continue
		}
		if out == nil {
			out = cloneSchemaSymbolTables(base)
		}
		mergeHintSchema(out, hinted)
	}

	if out == nil {
		return base
	}
	return out
}

func collectInstanceSchemaHints(doc *helium.Document) []instanceSchemaHint {
	var hints []instanceSchemaHint
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		elem, ok := n.(*helium.Element)
		if !ok {
			return nil
		}
		for _, a := range elem.Attributes() {
			if a.URI() != lexicon.NamespaceXSI {
				continue
			}
			switch a.LocalName() {
			case attrSchemaLocation:
				fields := value.XSDFields(a.Value())
				if len(fields)%2 != 0 {
					continue
				}
				for i := 0; i < len(fields); i += 2 {
					hints = append(hints, instanceSchemaHint{ns: fields[i], location: fields[i+1]})
				}
			case attrNoNSSchemaLocation:
				loc := value.XSDFields(a.Value())
				if len(loc) == 1 {
					hints = append(hints, instanceSchemaHint{location: loc[0]})
				}
			}
		}
		return nil
	}))
	return hints
}

func compileInstanceHintSchema(ctx context.Context, base *Schema, path string) *Schema {
	data, err := readInstanceHintSchema(base.loaderFS, path)
	if err != nil {
		return nil
	}

	parser := defaultSchemaParser()
	if base.loaderParser != nil {
		parser = *base.loaderParser
	}
	doc, err := parser.Parse(ctx, data)
	if err != nil {
		return nil
	}

	cfg := &compileConfig{
		fsys:       base.loaderFS,
		parser:     base.loaderParser,
		version:    base.version,
		versionSet: true,
	}
	schema, err := compileSchema(ctx, doc, schemaBaseDir(path), cfg)
	if err != nil {
		return nil
	}
	return schema
}

func readInstanceHintSchema(fsys fs.FS, path string) ([]byte, error) {
	c := &compiler{fsys: fsys}
	return c.readNestedSchema(path)
}

func cloneSchemaSymbolTables(s *Schema) *Schema {
	cp := *s
	cp.elements = maps.Clone(s.elements)
	cp.types = maps.Clone(s.types)
	cp.groups = maps.Clone(s.groups)
	cp.attrGroups = maps.Clone(s.attrGroups)
	cp.globalAttrs = maps.Clone(s.globalAttrs)
	cp.substGroups = maps.Clone(s.substGroups)
	return &cp
}

func mergeHintSchema(dst, src *Schema) {
	for qn, decl := range src.elements {
		if _, exists := dst.elements[qn]; !exists {
			dst.elements[qn] = decl
		}
	}
	for qn, td := range src.types {
		if _, exists := dst.types[qn]; !exists {
			dst.types[qn] = td
		}
	}
	for qn, mg := range src.groups {
		if _, exists := dst.groups[qn]; !exists {
			dst.groups[qn] = mg
		}
	}
	for qn, attrs := range src.attrGroups {
		if _, exists := dst.attrGroups[qn]; !exists {
			dst.attrGroups[qn] = attrs
		}
	}
	for qn, au := range src.globalAttrs {
		if _, exists := dst.globalAttrs[qn]; !exists {
			dst.globalAttrs[qn] = au
		}
	}
	for qn, members := range src.substGroups {
		if len(members) == 0 {
			continue
		}
		dst.substGroups[qn] = append(dst.substGroups[qn], members...)
	}
}
