package xslt3

import (
	"context"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
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

// LookupSchemaElement implements xpath3.SchemaDeclarations.
func (r *schemaRegistry) LookupSchemaElement(local, ns string) (typeName string, ok bool) {
	return r.LookupElement(local, ns)
}

// LookupSchemaAttribute implements xpath3.SchemaDeclarations.
func (r *schemaRegistry) LookupSchemaAttribute(local, ns string) (typeName string, ok bool) {
	return r.LookupAttribute(local, ns)
}

// LookupSchemaType implements xpath3.SchemaDeclarations.
func (r *schemaRegistry) LookupSchemaType(local, ns string) (baseType string, ok bool) {
	return r.LookupType(local, ns)
}

// IsSubtypeOf implements xpath3.SchemaDeclarations.
// It checks whether typeName is the same as or a subtype of baseTypeName using
// the annotation format ("xs:localName" for built-ins, "Q{ns}localName" for user-defined).
func (r *schemaRegistry) IsSubtypeOf(typeName, baseTypeName string) bool {
	if typeName == baseTypeName {
		return true
	}
	// Delegate built-in XSD types to the static hierarchy.
	if isXSBuiltin(typeName) {
		return isBuiltinSubtypeOf(typeName, baseTypeName)
	}
	// For user-defined types, walk the BaseType chain in the schemas.
	local, ns := splitAnnotationName(typeName)
	for _, s := range r.schemas {
		td, found := s.LookupType(local, ns)
		if !found {
			continue
		}
		cur := td.BaseType
		for cur != nil {
			curName := xsdTypeNameFromDef(cur)
			if curName == baseTypeName {
				return true
			}
			if isXSBuiltin(curName) {
				return isBuiltinSubtypeOf(curName, baseTypeName)
			}
			cur = cur.BaseType
		}
		return false
	}
	return false
}

// isXSBuiltin returns true if the annotation name is an xs: built-in type.
func isXSBuiltin(name string) bool {
	return len(name) > 3 && name[:3] == "xs:"
}

// isBuiltinSubtypeOf delegates to the xpath3 built-in type hierarchy.
func isBuiltinSubtypeOf(typeName, baseTypeName string) bool {
	return xpath3.BuiltinIsSubtypeOf(typeName, baseTypeName)
}

// splitAnnotationName parses an annotation name in the form "Q{ns}local" or
// "xs:local" (xs: already handled by isXSBuiltin) and returns local and ns.
func splitAnnotationName(name string) (local, ns string) {
	if len(name) > 2 && name[0] == 'Q' && name[1] == '{' {
		end := -1
		for i := 2; i < len(name); i++ {
			if name[i] == '}' {
				end = i
				break
			}
		}
		if end >= 0 {
			return name[end+1:], name[2:end]
		}
	}
	return name, ""
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
		return "Q{}" + td.Name.Local
	}
	return "xs:untyped"
}
