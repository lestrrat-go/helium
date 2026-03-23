package xslt3

import (
	"context"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
			// The element declaration's type may be nil when the XSD compiler
			// could not resolve a type attribute that matches the element's own
			// name (e.g., <xs:element name="foo" type="foo"/>). In this case,
			// check for a type definition with the same QName.
			if td, tdFound := s.LookupType(local, ns); tdFound {
				return xsdTypeNameFromDef(td), true
			}
			return "xs:untyped", true
		}
	}
	return "", false
}

// HasNamespace returns true if any schema in the registry has the given
// target namespace. Used to detect whether an element's namespace is
// governed by any imported schema.
func (r *schemaRegistry) HasNamespace(ns string) bool {
	for _, s := range r.schemas {
		if s.TargetNamespace() == ns {
			return true
		}
	}
	return false
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

// LookupTypeDef returns the raw TypeDef for a type name in annotation format.
func (r *schemaRegistry) LookupTypeDef(typeName string) (*xsd.TypeDef, *xsd.Schema, bool) {
	local, ns := splitAnnotationName(typeName)
	for _, s := range r.schemas {
		td, found := s.LookupType(local, ns)
		if found {
			return td, s, true
		}
	}
	return nil, nil, false
}

// LookupAllTypeDefs returns all matching TypeDef/Schema pairs for a type name.
// Used when multiple schemas define the same type in the same namespace (e.g.,
// different schema versions imported via different xsl:import-schema declarations).
func (r *schemaRegistry) LookupAllTypeDefs(typeName string) []struct {
	TD     *xsd.TypeDef
	Schema *xsd.Schema
} {
	local, ns := splitAnnotationName(typeName)
	var results []struct {
		TD     *xsd.TypeDef
		Schema *xsd.Schema
	}
	for _, s := range r.schemas {
		td, found := s.LookupType(local, ns)
		if found {
			results = append(results, struct {
				TD     *xsd.TypeDef
				Schema *xsd.Schema
			}{td, s})
		}
	}
	return results
}

// CastToSchemaType validates and normalizes a string value against a
// user-defined schema type. Returns the normalized value and nil on success,
// or an error if the value is invalid. The typeName must be in annotation
// format: "Q{ns}local" for user-defined types or "xs:local" for built-ins.
func (r *schemaRegistry) CastToSchemaType(value, typeName string) (string, error) {
	local, ns := splitAnnotationName(typeName)
	for _, s := range r.schemas {
		td, ok := s.LookupType(local, ns)
		if !ok {
			continue
		}
		if td.ContentType != xsd.ContentTypeSimple {
			// Complex types can't be cast from string.
			return value, nil
		}
		if err := xsd.ValidateSimpleValue(value, td); err != nil {
			return "", fmt.Errorf("value %q is not valid for type %s: %w", value, typeName, err)
		}
		return schemaNormalizeLexical(value, td), nil
	}
	// If the type name is still a prefix:local form (no Q{} wrapping yet),
	// try resolving by local name against all schemas' target namespaces.
	if !strings.HasPrefix(typeName, "Q{") && !strings.HasPrefix(typeName, "xs:") {
		idx := strings.IndexByte(typeName, ':')
		if idx >= 0 {
			localOnly := typeName[idx+1:]
			for _, s := range r.schemas {
				td, ok := s.LookupType(localOnly, s.TargetNamespace())
				if !ok {
					continue
				}
				if td.ContentType != xsd.ContentTypeSimple {
					return value, nil
				}
				if err := xsd.ValidateSimpleValue(value, td); err != nil {
					return "", fmt.Errorf("value %q is not valid for type %s: %w", value, typeName, err)
				}
				return schemaNormalizeLexical(value, td), nil
			}
		}
	}
	return "", fmt.Errorf("unknown schema type %s", typeName)
}

// LookupSchemaElement implements xpath3.SchemaDeclarations.
func (r *schemaRegistry) LookupSchemaElement(local, ns string) (typeName string, ok bool) {
	return r.LookupElement(local, ns)
}

// LookupSchemaAttribute implements xpath3.SchemaDeclarations.
func (r *schemaRegistry) LookupSchemaAttribute(local, ns string) (typeName string, ok bool) {
	return r.LookupAttribute(local, ns)
}

// LookupSchemaAttributeType returns the type name of an attribute declared
// within an element's type definition. This looks up the element declaration,
// finds its complex type, and returns the attribute's type.
func (r *schemaRegistry) LookupSchemaAttributeType(elemLocal, elemNS, attrLocal, attrNS string) string {
	for _, s := range r.schemas {
		elemDecl, found := s.LookupElement(elemLocal, elemNS)
		if !found {
			continue
		}
		if elemDecl.Type == nil {
			continue
		}
		for _, attrUse := range elemDecl.Type.Attributes {
			if attrUse.Name.Local == attrLocal && attrUse.Name.NS == attrNS {
				if attrUse.TypeName != (xsd.QName{}) {
					if attrUse.TypeName.NS == lexicon.NamespaceXSD {
						return "xs:" + attrUse.TypeName.Local
					}
					return xpath3.QAnnotation(attrUse.TypeName.NS, attrUse.TypeName.Local)
				}
			}
		}
	}
	return ""
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
	// xs:anyType is the universal supertype of all types.
	if baseTypeName == "xs:anyType" {
		return true
	}
	// Delegate built-in XSD types to the static hierarchy.
	if isXSBuiltin(typeName) {
		if isBuiltinSubtypeOf(typeName, baseTypeName) {
			return true
		}
		// Also check if baseTypeName is a union type with typeName as a member.
		if !isXSBuiltin(baseTypeName) {
			baseLocal, baseNS := splitAnnotationName(baseTypeName)
			for _, s := range r.schemas {
				baseTD, found := s.LookupType(baseLocal, baseNS)
				if !found {
					continue
				}
				if baseTD.Variety == xsd.TypeVarietyUnion {
					for _, member := range baseTD.MemberTypes {
						memberName := xsdTypeNameFromDef(member)
						if typeName == memberName || isBuiltinSubtypeOf(typeName, memberName) {
							return true
						}
					}
				}
			}
		}
		return false
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
	// Check if baseTypeName is a union type and typeName is a member type.
	baseLocal, baseNS := splitAnnotationName(baseTypeName)
	for _, s := range r.schemas {
		baseTD, found := s.LookupType(baseLocal, baseNS)
		if !found {
			continue
		}
		if baseTD.Variety == xsd.TypeVarietyUnion {
			for _, member := range baseTD.MemberTypes {
				memberName := xsdTypeNameFromDef(member)
				if typeName == memberName || r.IsSubtypeOf(typeName, memberName) {
					return true
				}
			}
		}
	}
	return false
}

// ValidateCast implements xpath3.SchemaDeclarations.
// It validates a string value against a user-defined schema type's facets.
func (r *schemaRegistry) ValidateCast(value, typeName string) error {
	td, _, found := r.LookupTypeDef(typeName)
	if !found {
		return nil // type not found — no facet check possible
	}
	if td.ContentType != xsd.ContentTypeSimple {
		return nil // complex types don't constrain string values
	}
	return xsd.ValidateSimpleValue(value, td)
}

// ValidateCastWithNS validates a value against a schema type using namespace
// context for QName/NOTATION resolution. Implements xpath3.SchemaDeclarations.
func (r *schemaRegistry) ValidateCastWithNS(value, typeName string, nsMap map[string]string) error {
	td, _, found := r.LookupTypeDef(typeName)
	if !found {
		return nil
	}
	if td.ContentType != xsd.ContentTypeSimple {
		return nil
	}
	return xsd.ValidateSimpleValueWithNS(value, nsMap, td)
}

// ListItemType implements xpath3.SchemaDeclarations.
// For list types, returns the item type name in annotation format.
func (r *schemaRegistry) ListItemType(typeName string) (string, bool) {
	td, _, found := r.LookupTypeDef(typeName)
	if !found {
		return "", false
	}
	// Walk up the type chain to find the list variety.
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Variety == xsd.TypeVarietyList && cur.ItemType != nil {
			return xsdTypeNameFromDef(cur.ItemType), true
		}
	}
	return "", false
}

// UnionMemberTypes implements xpath3.SchemaDeclarations.
func (r *schemaRegistry) UnionMemberTypes(typeName string) []string {
	td, _, found := r.LookupTypeDef(typeName)
	if !found || td == nil || td.Variety != xsd.TypeVarietyUnion {
		return nil
	}
	members := make([]string, 0, len(td.MemberTypes))
	for _, member := range td.MemberTypes {
		members = append(members, xsdTypeNameFromDef(member))
	}
	return members
}

// IsSubstitutionGroupMember checks if an element (memberLocal, memberNS) is in
// the substitution group of the head element (headLocal, headNS).
func (r *schemaRegistry) IsSubstitutionGroupMember(memberLocal, memberNS, headLocal, headNS string) bool {
	headQN := xsd.QName{Local: headLocal, NS: headNS}
	memberQN := xsd.QName{Local: memberLocal, NS: memberNS}
	for _, s := range r.schemas {
		for _, member := range s.SubstGroupMembers(headQN) {
			if member.Name == memberQN {
				return true
			}
		}
	}
	return false
}

// isXSBuiltin returns true if the annotation name is an xs: built-in type.
func isXSBuiltin(name string) bool {
	return len(name) > 3 && name[:3] == "xs:"
}

// builtinSimpleTypes is the set of xs: built-in simple (atomic/list/union) types
// that can be validated via xpath3.CastFromString. Complex types (xs:anyType,
// xs:untyped) and the abstract anySimpleType/anyAtomicType are excluded because
// they have no concrete lexical space to validate against.
var builtinSimpleTypes = map[string]struct{}{
	"xs:string":             {},
	"xs:boolean":            {},
	"xs:decimal":            {},
	"xs:float":              {},
	"xs:double":             {},
	"xs:duration":           {},
	"xs:dateTime":           {},
	"xs:time":               {},
	"xs:date":               {},
	"xs:gYearMonth":         {},
	"xs:gYear":              {},
	"xs:gMonthDay":          {},
	"xs:gDay":               {},
	"xs:gMonth":             {},
	"xs:hexBinary":          {},
	"xs:base64Binary":       {},
	"xs:anyURI":             {},
	"xs:QName":              {},
	"xs:NOTATION":           {},
	"xs:normalizedString":   {},
	"xs:token":              {},
	"xs:language":           {},
	"xs:NMTOKEN":            {},
	"xs:NMTOKENS":           {},
	"xs:Name":               {},
	"xs:NCName":             {},
	"xs:ID":                 {},
	"xs:IDREF":              {},
	"xs:IDREFS":             {},
	"xs:ENTITY":             {},
	"xs:ENTITIES":           {},
	"xs:integer":            {},
	"xs:nonPositiveInteger": {},
	"xs:negativeInteger":    {},
	"xs:long":               {},
	"xs:int":                {},
	"xs:short":              {},
	"xs:byte":               {},
	"xs:nonNegativeInteger": {},
	"xs:unsignedLong":       {},
	"xs:unsignedInt":        {},
	"xs:unsignedShort":      {},
	"xs:unsignedByte":       {},
	"xs:positiveInteger":    {},
	"xs:yearMonthDuration":  {},
	"xs:dayTimeDuration":    {},
	"xs:dateTimeStamp":      {},
}

// isBuiltinSimpleType returns true if typeName is a concrete xs: built-in simple
// type that can be validated via xpath3.CastFromString.
func isBuiltinSimpleType(typeName string) bool {
	_, ok := builtinSimpleTypes[typeName]
	return ok
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
		au, found := s.LookupAttribute(local, ns)
		if !found {
			continue
		}
		if au.TypeName.Local == "" {
			return "xs:untypedAtomic", true
		}
		td, tdOk := s.LookupType(au.TypeName.Local, au.TypeName.NS)
		if tdOk {
			return xsdTypeNameFromDef(td), true
		}
		return "xs:untypedAtomic", true
	}
	return "", false
}

// xmlNSAttrTypes maps xml:* attribute local names to their XSD type names.
// These are the built-in declarations for the http://www.w3.org/XML/1998/namespace
// namespace, as specified by the XML namespace schema.
var xmlNSAttrTypes = map[string]string{
	"lang":  "xs:language",
	"space": "xs:NMTOKEN", // enumeration "preserve"|"default"
	"id":    "xs:ID",
	"base":  "xs:anyURI",
}

// xmlNSAttrEnums maps xml:space to its allowed enumeration values.
var xmlNSAttrEnums = map[string]map[string]struct{}{
	"space": {"preserve": {}, "default": {}},
}

// validateXMLNSAttr validates a value for the xml:* built-in attributes.
// Returns ("", true, nil) when no built-in declaration exists for localName.
// Returns (typeName, true, nil) on success, (typeName, false, err) on failure.
//
// Special cases per the XML 1.0 spec and the W3C XML namespace schema:
//   - xml:lang: empty string is valid (means "no language"); non-empty must
//     match a BCP 47 language tag pattern (simplified check)
//   - xml:space: only "preserve" or "default" are allowed
//   - xml:id: must be a valid XML NCName (xs:ID)
//   - xml:base: must be a valid URI (xs:anyURI)
func validateXMLNSAttr(localName, value string) (string, bool, error) {
	typeName, ok := xmlNSAttrTypes[localName]
	if !ok {
		return "", true, nil // unknown xml:* attribute — no built-in validation
	}

	switch localName {
	case "lang":
		// Per XML 1.0 §2.12, xml:lang="" is valid (means "no language").
		// Non-empty values must be valid BCP 47 language tags.
		if value == "" {
			return typeName, true, nil
		}
		_, castErr := xpath3.CastFromString(value, typeName)
		if castErr != nil {
			return typeName, false, castErr
		}
		return typeName, true, nil

	case "space":
		// xml:space is restricted to the enumeration "preserve" | "default".
		if enums := xmlNSAttrEnums["space"]; enums != nil {
			if _, valid := enums[value]; !valid {
				return typeName, false, fmt.Errorf("value %q is not one of the allowed values (preserve, default)", value)
			}
		}
		return typeName, true, nil

	default:
		// Generic validation via CastFromString.
		_, castErr := xpath3.CastFromString(value, typeName)
		if castErr != nil {
			return typeName, false, castErr
		}
		return typeName, true, nil
	}
}

// ValidateAttribute validates a single attribute value against a globally
// declared attribute in the imported schemas. It returns (typeName, true, nil)
// when the attribute value is valid, (typeName, false, err) when invalid, and
// ("", false, nil) when no matching global attribute declaration is found.
func (r *schemaRegistry) ValidateAttribute(localName, nsURI, value string) (string, bool, error) {
	// Check built-in XML namespace declarations.
	if nsURI == lexicon.NamespaceXML {
		if r.hasXMLNSImport() {
			return validateXMLNSAttr(localName, value)
		}
	}

	for _, s := range r.schemas {
		au, found := s.LookupAttribute(localName, nsURI)
		if !found {
			continue
		}
		if au.TypeName.Local == "" {
			// No type constraint — any value is valid.
			return "xs:untypedAtomic", true, nil
		}
		// Resolve the type definition and validate the value string.
		td, tdOk := s.LookupType(au.TypeName.Local, au.TypeName.NS)
		if !tdOk {
			return "xs:untypedAtomic", true, nil
		}
		typeName := xsdTypeNameFromDef(td)
		// Validate using the XSD type definition (supports pattern facets,
		// enumerations, and other constraints that CastFromString doesn't handle).
		if valErr := xsd.ValidateSimpleValue(value, td); valErr != nil {
			return typeName, false, valErr
		}
		return typeName, true, nil
	}
	// No matching global attribute declaration found.
	return "", false, nil
}

// hasXMLNSImport returns true if the registry knows about the XML namespace
// (i.e., xsl:import-schema namespace="http://www.w3.org/XML/1998/namespace"
// was used in the stylesheet).
func (r *schemaRegistry) hasXMLNSImport() bool {
	// The XML namespace is handled as a built-in, so a namespace-only import
	// doesn't add a schema entry. We signal its presence via a sentinel nil
	// entry in the schemas slice, OR we track it explicitly. For simplicity,
	// always treat the XML namespace as known when the registry exists and
	// was initialized by a stylesheet that is schema-aware.
	return true // registry is only created for schema-aware stylesheets
}

// IsComplexType returns true if the given annotation-format type name refers
// to a complex type in the imported schemas. Built-in xs:* types are always
// simple, so they return false.
func (r *schemaRegistry) IsComplexType(typeName string) bool {
	td, _, found := r.LookupTypeDef(typeName)
	if !found {
		return false
	}
	// A complex type has attributes, a content model, or element-only/mixed content.
	if len(td.Attributes) > 0 || td.AnyAttribute != nil || td.ContentModel != nil {
		return true
	}
	if td.ContentType == xsd.ContentTypeElementOnly || td.ContentType == xsd.ContentTypeMixed {
		return true
	}
	return false
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

	// No namespace match — try schemas with empty target namespace, but only
	// when the document root is also in no namespace. A no-namespace schema
	// should not validate elements in a different namespace.
	if rootNS == "" {
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
	}

	// No matching schema — return empty annotations.
	return nil, nil
}

// findDocumentElement returns the root element of a document, or nil.
func findDocumentElement(doc *helium.Document) *helium.Element {
	for child := range helium.Children(doc) {
		if child.Type() == helium.ElementNode {
			return child.(*helium.Element)
		}
	}
	return nil
}

// xsdTypeNameFromDef converts a xsd.TypeDef to a type name string.
// For anonymous types (no name), it walks the BaseType chain to find
// the nearest named ancestor type, so schema-element() matching can
// compare against the declared type hierarchy.
func xsdTypeNameFromDef(td *xsd.TypeDef) string {
	if td == nil {
		return "xs:untyped"
	}
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Name.NS == lexicon.NamespaceXSD && cur.Name.Local != "" {
			return "xs:" + cur.Name.Local
		}
		if cur.Name.NS != "" && cur.Name.Local != "" {
			return xpath3.QAnnotation(cur.Name.NS, cur.Name.Local)
		}
		if cur.Name.Local != "" {
			return "Q{}" + cur.Name.Local
		}
	}
	// Anonymous type with no named ancestor in the base chain: the type
	// was validated, so it implicitly derives from xs:anyType.
	return "xs:anyType"
}

// ValidateDocIDConstraints checks document-level xs:ID uniqueness and xs:IDREF
// resolution using the type annotations from a prior ValidateDoc call. Returns
// an error wrapping XTTE1555 if any constraint is violated.
func ValidateDocIDConstraints(doc *helium.Document, ann xsd.TypeAnnotations) error {
	// Collect all xs:ID and xs:IDREF values by walking the annotation map.
	ids := make(map[string]struct{})
	var idrefs []string

	for node, typeName := range ann {
		attr, ok := node.(*helium.Attribute)
		if !ok {
			continue
		}
		val := attr.Value()
		switch typeName {
		case "xs:ID":
			if _, dup := ids[val]; dup {
				return dynamicError(errCodeXTTE1555,
					"duplicate xs:ID value %q in validated document", val)
			}
			ids[val] = struct{}{}
		case "xs:IDREF":
			idrefs = append(idrefs, val)
		case "xs:IDREFS":
			// Space-separated list of IDREFs.
			idrefs = append(idrefs, splitSpaceSeparated(val)...)
		}
	}

	// Also walk the document tree for xml:id attributes, which are always xs:ID
	// (part of the XML namespace built-in schema), and may not appear in ann when
	// only a namespace-only import is present.
	if err := collectXMLIDsFromDoc(doc, ids); err != nil {
		return err
	}

	// Collect IDREFs from xsi:type-annotated elements (e.g., xsi:type="xs:IDREFS").
	idrefs = append(idrefs, collectXSITypeIDREFs(doc)...)

	// Check that all IDREFs resolve to an ID.
	for _, ref := range idrefs {
		if _, ok := ids[ref]; !ok {
			return dynamicError(errCodeXTTE1555,
				"xs:IDREF value %q does not resolve to any xs:ID in the validated document", ref)
		}
	}

	return nil
}

// collectXMLIDsFromDoc walks the document tree collecting xml:id attribute values
// (which are always xs:ID typed per the XML namespace spec), checking for duplicates,
// and also collecting xs:IDREFS element content (from xsi:type overrides).
func collectXMLIDsFromDoc(doc *helium.Document, ids map[string]struct{}) error {
	return walkElementsForID(doc.FirstChild(), ids)
}


// walkElementsForID recursively walks sibling node chains collecting xml:id values.
func walkElementsForID(node helium.Node, ids map[string]struct{}) error {
	for ; node != nil; node = node.NextSibling() {
		elem, ok := node.(*helium.Element)
		if !ok {
			continue
		}
		for _, attr := range elem.Attributes() {
			if attr.URI() == lexicon.NamespaceXML && attr.LocalName() == "id" {
				val := attr.Value()
				if _, dup := ids[val]; dup {
					return dynamicError(errCodeXTTE1555,
						"duplicate xml:id value %q in validated document", val)
				}
				ids[val] = struct{}{}
			}
		}
		if err := walkElementsForID(elem.FirstChild(), ids); err != nil {
			return err
		}
	}
	return nil
}

// collectXSITypeIDREFs walks the document tree collecting IDREF values from elements
// with xsi:type="xs:IDREFS" or xsi:type="xs:IDREF". These need to be checked against
// the collected xs:ID values for resolution.
func collectXSITypeIDREFs(doc *helium.Document) []string {
	var idrefs []string
	collectXSITypeIDREFsFromNode(doc.FirstChild(), &idrefs)
	return idrefs
}

func collectXSITypeIDREFsFromNode(node helium.Node, idrefs *[]string) {
	for ; node != nil; node = node.NextSibling() {
		elem, ok := node.(*helium.Element)
		if !ok {
			continue
		}
		// Check for xsi:type="xs:IDREF" or xsi:type="xs:IDREFS".
		for _, attr := range elem.Attributes() {
			if attr.URI() == lexicon.NamespaceXSI && attr.LocalName() == "type" {
				xsiType := attr.Value()
				// Strip prefix to get local type name (e.g., "xs:IDREFS" → "IDREFS").
				if idx := len(xsiType) - len("IDREFS"); idx >= 0 && xsiType[idx:] == "IDREFS" {
					*idrefs = append(*idrefs, splitSpaceSeparated(string(elem.Content()))...)
				} else if idx := len(xsiType) - len("IDREF"); idx >= 0 && xsiType[idx:] == "IDREF" {
					val := string(elem.Content())
					if val != "" {
						*idrefs = append(*idrefs, val)
					}
				}
			}
		}
		collectXSITypeIDREFsFromNode(elem.FirstChild(), idrefs)
	}
}

// splitSpaceSeparated splits a string on whitespace into individual tokens.
func splitSpaceSeparated(s string) []string {
	var parts []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			if start >= 0 {
				parts = append(parts, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, s[start:])
	}
	return parts
}
