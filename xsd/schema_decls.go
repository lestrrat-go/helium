package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
)

// castGuardKey carries, in the evaluation context, the set of (type, value) casts
// currently being validated, so a self-referential schema-aware cast inside a
// type's own xs:assertion is detected and fails closed instead of recursing
// forever. It is per-evaluation (lives only in the derived contexts of one
// validateCast → validateValue → assertion-evaluate chain) and self-clears as the
// recursion unwinds, so nothing leaks between top-level validations.
type castGuardKey struct{}

type castGuardEntry struct {
	typeName string
	value    string
}

// schemaDecls adapts a compiled *Schema to xpath3.SchemaDeclarations so an
// xs:assert / xs:assertion XPath atomizes a PSVI-typed node through its schema
// type — a node annotated with a NAMED user-defined simple type (e.g. a
// restriction of xs:integer) resolves to its builtin base, and instance-of /
// schema-element tests see the type hierarchy. It mirrors the lookup behavior of
// xslt3's schemaRegistry but over a single schema. Type names use the annotation
// format: "xs:local" for builtins, "Q{ns}local" (or "Q{}local") for user types.
type schemaDecls struct {
	schema  *Schema
	version Version
	// anon / anonNames carry inline ANONYMOUS list/union simple types registered
	// for xs:assert node atomization (see validationContext.assertAnonTypes), keyed
	// by their synthetic annotation name and by *TypeDef respectively. They let the
	// list-item / union-member lookups recover metadata an anonymous type has no
	// schema-table entry for.
	anon      map[string]*TypeDef
	anonNames map[*TypeDef]string
}

// lookupTypeName resolves an annotation-format type name to its *TypeDef,
// consulting the anonymous-type registry before the schema's named-type table.
func (d schemaDecls) lookupTypeName(typeName string) (*TypeDef, bool) {
	if d.anon != nil {
		if td, ok := d.anon[typeName]; ok {
			return td, true
		}
	}
	local, ns := annotationParts(typeName)
	return d.schema.LookupType(local, ns)
}

// lookupAtomizationType resolves a type name to the *TypeDef that drives ATOMIZATION
// of a node annotated with it. For a simpleContent COMPLEX type it returns the
// EFFECTIVE content simple type (effectiveContentSimpleType) — the narrowed
// content (a nested/simpleContent restriction to xs:QName, a list, a union, …) —
// rather than the raw complex type, so data()/list/union node atomization matches
// validation and $value (which both type the content via the same effective type).
// A non-simpleContent type is returned unchanged.
func (d schemaDecls) lookupAtomizationType(typeName string) (*TypeDef, bool) {
	td, ok := d.lookupTypeName(typeName)
	if !ok {
		return nil, false
	}
	return effectiveContentSimpleType(td), true
}

// typeName returns the annotation name for td, preferring a registered synthetic
// anonymous name so an inline list item / union member round-trips back to its
// actual *TypeDef.
func (d schemaDecls) typeName(td *TypeDef) string {
	if d.anonNames != nil {
		if name, ok := d.anonNames[td]; ok {
			return name
		}
	}
	return xsdTypeName(td)
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
	var td *TypeDef
	var ok bool
	if d.anon != nil && ns == assertAnonNS {
		td, ok = d.anon["Q{"+ns+"}"+local]
	} else {
		td, ok = d.schema.LookupType(local, ns)
	}
	if !ok {
		return "", false
	}
	// For a simpleContent COMPLEX type, atomization walks the NARROWED content
	// simple type, not the complex type's own (complex) base chain.
	td = effectiveContentSimpleType(td)
	if td.BaseType != nil {
		return d.typeName(td.BaseType), true
	}
	return d.typeName(td), true
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
	// Resolve through the anonymous-type registry too (lookupTypeName), so a
	// synthetic assert annotation (Q{urn:x-helium:assert-anon}N for an inline
	// anonymous list/union) participates in subtype checks; walk bases via
	// d.typeName so an anonymous ancestor keeps its synthetic name.
	td, ok := d.lookupTypeName(typeName)
	if !ok {
		return false
	}
	// Every SIMPLE type derives (ultimately) from xs:anySimpleType, even a list or
	// union whose BaseType pointer is left nil (the implicit anySimpleType root). A
	// simpleContent COMPLEX type (IsComplex with ContentType==ContentTypeSimple) is
	// NOT a simple type — it is not a subtype of xs:anySimpleType for node/instance-of
	// tests — so exclude it (its content is resolved separately for ATOMIZATION).
	if td.ContentType == ContentTypeSimple && !td.IsComplex && baseTypeName == xpath3.TypeAnySimpleType {
		return true
	}
	for cur := td.BaseType; cur != nil; cur = cur.BaseType {
		name := d.typeName(cur)
		// When the ORIGINAL type is a (simpleContent) COMPLEX type, only COMPLEX base
		// ancestors count for node/instance-of/subtype tests. Its {base type definition}
		// may be a SIMPLE type (e.g. a simpleContent extension/restriction of xs:string),
		// but the complex type is NOT a subtype of that simple base or its simple
		// ancestors (xs:string, xs:anySimpleType, …) — its only universal simple-side
		// ancestor is xs:anyType (handled by the early return). Skipping the simple
		// ancestry here keeps `t:c instance of element(*, xs:string)` (or any user simple
		// base) FALSE, while data() still atomizes through the narrowed content type
		// (resolved separately via LookupSchemaType/effectiveContentSimpleType).
		if td.IsComplex && !cur.IsComplex {
			continue
		}
		if name == baseTypeName {
			return true
		}
		// Only fall into the BUILTIN simple-type hierarchy when the ORIGINAL type is
		// itself simple (a complex original never reaches here — its simple ancestors are
		// skipped above).
		if !td.IsComplex && xpath3.IsKnownXSDType(name) {
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
	td, ok := d.lookupTypeName(typeName)
	if !ok || td.ContentType != ContentTypeSimple {
		return nil
	}
	// A simpleContent COMPLEX type is NOT a valid simple/atomic cast target — its
	// content type drives data() atomization (LookupSchemaType resolves that), but
	// `cast`/`castable as` must reject it as a target, not facet-validate its content.
	if td.IsComplex {
		return fmt.Errorf("xsd: %s is a complex type, not a valid cast target", typeName)
	}

	// Guard against a self-referential cast: a `cast`/`castable as t:T` evaluated
	// inside t:T's own xs:assertion would recurse (validateCast → validateValue →
	// checkSimpleTypeAssertions → Evaluate → validateCast …) until the stack
	// overflows. Track the active (type, value) casts in the context and, on a
	// repeat, fail closed — the cast is treated as not castable / a cast failure,
	// which terminates the recursion.
	entry := castGuardEntry{typeName: typeName, value: value}
	guard, _ := ctx.Value(castGuardKey{}).(map[castGuardEntry]struct{})
	if guard == nil {
		guard = make(map[castGuardEntry]struct{})
		ctx = context.WithValue(ctx, castGuardKey{}, guard)
	}
	if _, active := guard[entry]; active {
		return fmt.Errorf("xsd: recursive cast to %s while validating its own assertion", typeName)
	}
	guard[entry] = struct{}{}
	defer delete(guard, entry)

	// Validate with the SCHEMA's version, not TypeDef.Validate's hardcoded 1.0
	// default — inside a 1.1 assertion a user-defined cast/castable must accept
	// 1.1-only lexical forms (e.g. year 0000). suppressDepth keeps the throwaway
	// context from emitting diagnostics; validateValue still returns the error.
	vc := &validationContext{
		schema:        d.schema,
		version:       d.version,
		errorHandler:  helium.NilErrorHandler{},
		suppressDepth: 1,
	}
	return validateValue(ctx, value, nsMap, td, "", "", 0, vc)
}

// ListItemType returns the item type name for a list type.
func (d schemaDecls) ListItemType(typeName string) (string, bool) {
	td, ok := d.lookupAtomizationType(typeName)
	if !ok {
		return "", false
	}
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Variety == TypeVarietyList && cur.ItemType != nil {
			return d.typeName(cur.ItemType), true
		}
	}
	return "", false
}

// UnionMemberTypes returns the member type names for a union type. Variety and the
// member list are resolved via resolveVariety / resolveUnionMembers (which walk the
// base chain), NOT the type's direct fields — so a synthetic facet-only restriction
// over a union base (e.g. a simpleContent restriction with a direct xs:pattern over a
// union content type, whose effective content type carries only BaseType) still
// reports its union members, matching how validation / $value / CTA resolve variety.
func (d schemaDecls) UnionMemberTypes(typeName string) []string {
	td, ok := d.lookupAtomizationType(typeName)
	if !ok || resolveVariety(td) != TypeVarietyUnion {
		return nil
	}
	members := resolveUnionMembers(td)
	out := make([]string, 0, len(members))
	for _, m := range members {
		out = append(out, d.typeName(m))
	}
	return out
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
	return schemaDecls{schema: vc.schema, version: vc.version, anon: vc.assertAnonTypes, anonNames: vc.assertAnonNames}
}
