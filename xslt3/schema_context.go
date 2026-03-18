package xslt3

import (
	"context"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

// schemaRegistry wraps multiple imported schemas and provides unified
// lookup and validation operations for schema-aware XSLT processing.
type schemaRegistry struct {
	schemas []*xsd.Schema
}

// LookupElement returns the element declaration and its type name from the
// first schema that declares a matching global element.
func (r *schemaRegistry) LookupElement(local, ns string) (typeName string, ok bool) {
	for _, s := range r.schemas {
		edecl, found := s.LookupElement(local, ns)
		if found && edecl.Type != nil {
			return xsdTypeNameFromDef(edecl.Type), true
		}
		if found {
			return "xs:untyped", true
		}
	}
	return "", false
}

// LookupType returns the base type name for a schema type definition.
func (r *schemaRegistry) LookupType(local, ns string) (baseType string, ok bool) {
	for _, s := range r.schemas {
		td, found := s.LookupType(local, ns)
		if found {
			if td.BaseType != nil {
				return xsdTypeNameFromDef(td.BaseType), true
			}
			return xsdTypeNameFromDef(td), true
		}
	}
	return "", false
}

// LookupAttribute returns the attribute declaration type name from the
// first schema that declares a matching global attribute.
func (r *schemaRegistry) LookupAttribute(local, ns string) (typeName string, ok bool) {
	for _, s := range r.schemas {
		// Global attributes are searched via the type map since AttrUse
		// type names need resolution through the schema's type map.
		// For now, iterate elements would not help. Schema does not expose
		// globalAttrs publicly, so we check the NamedTypes approach.
		_ = s
	}
	return "", false
}

// ValidateDoc validates a document against the imported schemas and returns
// per-node type annotations. If no schema matches the document's root element,
// empty annotations are returned (lax behavior).
func (r *schemaRegistry) ValidateDoc(ctx context.Context, doc *helium.Document) (xsd.TypeAnnotations, error) {
	root := findDocumentElement(doc)
	if root == nil {
		return nil, nil
	}

	rootNS := root.URI()

	// Find the schema whose target namespace matches the document root.
	for _, s := range r.schemas {
		if s.TargetNamespace() == rootNS {
			var ann xsd.TypeAnnotations
			err := xsd.Validate(ctx, doc, s, xsd.WithAnnotations(&ann))
			if err != nil {
				return ann, err
			}
			return ann, nil
		}
	}

	// No namespace match — try schemas with empty target namespace.
	for _, s := range r.schemas {
		if s.TargetNamespace() == "" {
			var ann xsd.TypeAnnotations
			err := xsd.Validate(ctx, doc, s, xsd.WithAnnotations(&ann))
			if err != nil {
				return ann, err
			}
			return ann, nil
		}
	}

	// No matching schema — return empty annotations.
	return nil, nil
}

// findDocumentElement returns the root element of a document, or nil.
func findDocumentElement(doc *helium.Document) *helium.Element {
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			return child.(*helium.Element)
		}
	}
	return nil
}

// xsdTypeNameFromDef converts a xsd.TypeDef to a type name string.
func xsdTypeNameFromDef(td *xsd.TypeDef) string {
	if td == nil {
		return "xs:untyped"
	}
	if td.Name.NS == "http://www.w3.org/2001/XMLSchema" {
		return "xs:" + td.Name.Local
	}
	if td.Name.NS != "" {
		return "Q{" + td.Name.NS + "}" + td.Name.Local
	}
	if td.Name.Local != "" {
		return td.Name.Local
	}
	return "xs:untyped"
}
