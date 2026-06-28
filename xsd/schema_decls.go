package xsd

import (
	"context"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

// schemaDecls adapts a compiled *Schema to xpath3.SchemaDeclarations so an
// xs:assert / xs:assertion XPath atomizes a PSVI-typed node through its schema
// type — a node annotated with a NAMED user-defined simple type (e.g. a
// restriction of xs:integer) resolves to its builtin base, and instance-of /
// schema-element tests see the type hierarchy. It mirrors the lookup behavior of
// xslt3's schemaRegistry but over a single schema. Type names use the annotation
// format: "xs:local" for builtins, "Q{ns}local" (or "Q{}local") for user types.
type schemaDecls struct {
	schema *Schema
}

// LookupSchemaElement returns the annotation-format type name of a global element.
func (d schemaDecls) LookupSchemaElement(local, ns string) (string, bool) {
	edecl, ok := d.schema.LookupElement(local, ns)
	if !ok {
		return "", false
	}
	if edecl.Type != nil {
		return xsdTypeName(edecl.Type), true
	}
	if td, ok := d.schema.LookupType(local, ns); ok {
		return xsdTypeName(td), true
	}
	return lexicon.XSAnyType, true
}

// LookupSchemaAttribute returns the annotation-format type name of a global attribute.
func (d schemaDecls) LookupSchemaAttribute(local, ns string) (string, bool) {
	au, ok := d.schema.LookupAttribute(local, ns)
	if !ok {
		return "", false
	}
	if au.Type != nil {
		return xsdTypeName(au.Type), true
	}
	if au.TypeName.Local != "" {
		if td, ok := d.schema.LookupType(au.TypeName.Local, au.TypeName.NS); ok {
			return xsdTypeName(td), true
		}
	}
	return lexicon.XSUntypedAtomic, true
}

// LookupSchemaType returns the annotation-format name of td's BASE type (its
// supertype), which is what xpath3 atomization walks to reach a builtin base.
func (d schemaDecls) LookupSchemaType(local, ns string) (string, bool) {
	td, ok := d.schema.LookupType(local, ns)
	if !ok {
		return "", false
	}
	if td.BaseType != nil {
		return xsdTypeName(td.BaseType), true
	}
	return xsdTypeName(td), true
}

// IsSubtypeOf reports whether typeName is the same as, or a subtype of,
// baseTypeName (annotation format).
func (d schemaDecls) IsSubtypeOf(typeName, baseTypeName string) bool {
	if typeName == baseTypeName || baseTypeName == lexicon.XSAnyType {
		return true
	}
	if xpath3.IsKnownXSDType(typeName) {
		return xpath3.BuiltinIsSubtypeOf(typeName, baseTypeName)
	}
	local, ns := annotationParts(typeName)
	td, ok := d.schema.LookupType(local, ns)
	if !ok {
		return false
	}
	for cur := td.BaseType; cur != nil; cur = cur.BaseType {
		name := xsdTypeName(cur)
		if name == baseTypeName {
			return true
		}
		if xpath3.IsKnownXSDType(name) {
			return xpath3.BuiltinIsSubtypeOf(name, baseTypeName)
		}
	}
	return false
}

// ValidateCast validates value against a user-defined simple type's facets.
func (d schemaDecls) ValidateCast(ctx context.Context, value, typeName string) error {
	return d.validateCast(ctx, value, typeName, nil)
}

// ValidateCastWithNS validates value against a schema type using namespace context.
func (d schemaDecls) ValidateCastWithNS(ctx context.Context, value, typeName string, nsMap map[string]string) error {
	return d.validateCast(ctx, value, typeName, nsMap)
}

func (d schemaDecls) validateCast(ctx context.Context, value, typeName string, nsMap map[string]string) error {
	local, ns := annotationParts(typeName)
	td, ok := d.schema.LookupType(local, ns)
	if !ok || td.ContentType != ContentTypeSimple {
		return nil
	}
	return td.Validate(ctx, value, nsMap)
}

// ListItemType returns the item type name for a list type.
func (d schemaDecls) ListItemType(typeName string) (string, bool) {
	local, ns := annotationParts(typeName)
	td, ok := d.schema.LookupType(local, ns)
	if !ok {
		return "", false
	}
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Variety == TypeVarietyList && cur.ItemType != nil {
			return xsdTypeName(cur.ItemType), true
		}
	}
	return "", false
}

// UnionMemberTypes returns the member type names for a union type.
func (d schemaDecls) UnionMemberTypes(typeName string) []string {
	local, ns := annotationParts(typeName)
	td, ok := d.schema.LookupType(local, ns)
	if !ok || td.Variety != TypeVarietyUnion {
		return nil
	}
	members := make([]string, 0, len(td.MemberTypes))
	for _, m := range td.MemberTypes {
		members = append(members, xsdTypeName(m))
	}
	return members
}

// annotationParts parses an annotation-format type name. "xs:local" returns
// (local, XSD namespace); "Q{ns}local" returns (local, ns); a bare name returns
// (name, "").
func annotationParts(name string) (local, ns string) {
	if len(name) > 3 && name[:3] == "xs:" {
		return name[3:], lexicon.NamespaceXSD
	}
	if len(name) > 2 && name[0] == 'Q' && name[1] == '{' {
		for i := 2; i < len(name); i++ {
			if name[i] == '}' {
				return name[i+1:], name[2:i]
			}
		}
	}
	return name, ""
}

// assertSchemaDecls returns the SchemaDeclarations adapter for this validation's
// schema, or nil when no schema is available.
func (vc *validationContext) assertSchemaDecls() xpath3.SchemaDeclarations {
	if vc.schema == nil {
		return nil
	}
	return schemaDecls{schema: vc.schema}
}
