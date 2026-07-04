package xsd

import "github.com/lestrrat-go/helium/internal/lexicon"

// builtinSimpleBase maps each XSD built-in SIMPLE type's local name (XSD
// namespace) to the local name of the type it is derived from — its {base type
// definition} — forming the built-in datatype hierarchy of XSD §3.2/§3.4. The
// chain bottoms out at "anySimpleType"; the complex xs:anyType is intentionally
// excluded, so this encodes only the SIMPLE-type hierarchy.
//
// The built-in types are registered (registerBuiltinTypes) WITHOUT BaseType
// pointer links, so isDerivedFrom cannot chain e.g. xs:nonNegativeInteger →
// xs:integer → xs:decimal. This table is the authoritative encoding used wherever
// that chaining is needed. Keep it accurate if built-in types are added/changed.
// Keys/values use lexicon.Type* identifiers (not string literals) so the table is
// the single source of truth and adds no duplicate-literal lint noise.
var builtinSimpleBase = map[string]string{
	// anyAtomicType is the base of every primitive (XSD 1.1).
	lexicon.TypeAnyAtomicType: lexicon.TypeAnySimpleType,

	// Primitives ⊂ anyAtomicType.
	lexicon.TypeString:       lexicon.TypeAnyAtomicType,
	lexicon.TypeBoolean:      lexicon.TypeAnyAtomicType,
	lexicon.TypeDecimal:      lexicon.TypeAnyAtomicType,
	lexicon.TypeFloat:        lexicon.TypeAnyAtomicType,
	lexicon.TypeDouble:       lexicon.TypeAnyAtomicType,
	lexicon.TypeDuration:     lexicon.TypeAnyAtomicType,
	lexicon.TypeDateTime:     lexicon.TypeAnyAtomicType,
	lexicon.TypeTime:         lexicon.TypeAnyAtomicType,
	lexicon.TypeDate:         lexicon.TypeAnyAtomicType,
	lexicon.TypeGYearMonth:   lexicon.TypeAnyAtomicType,
	lexicon.TypeGYear:        lexicon.TypeAnyAtomicType,
	lexicon.TypeGMonthDay:    lexicon.TypeAnyAtomicType,
	lexicon.TypeGDay:         lexicon.TypeAnyAtomicType,
	lexicon.TypeGMonth:       lexicon.TypeAnyAtomicType,
	lexicon.TypeHexBinary:    lexicon.TypeAnyAtomicType,
	lexicon.TypeBase64Binary: lexicon.TypeAnyAtomicType,
	lexicon.TypeAnyURI:       lexicon.TypeAnyAtomicType,
	lexicon.TypeQName:        lexicon.TypeAnyAtomicType,
	lexicon.TypeNotation:     lexicon.TypeAnyAtomicType,

	// String family.
	lexicon.TypeNormalizedString: lexicon.TypeString,
	lexicon.TypeToken:            lexicon.TypeNormalizedString,
	lexicon.TypeLanguage:         lexicon.TypeToken,
	lexicon.TypeNMToken:          lexicon.TypeToken,
	lexicon.TypeName:             lexicon.TypeToken,
	lexicon.TypeNCName:           lexicon.TypeName,
	lexicon.TypeID:               lexicon.TypeNCName,
	lexicon.TypeIDREF:            lexicon.TypeNCName,
	lexicon.TypeENTITY:           lexicon.TypeNCName,
	// The built-in LIST types derive (by list construction) from xs:anySimpleType,
	// NOT from their atomic item type — a list type is never ·validly derived· from
	// its item type. Chaining them to NMTOKEN/IDREF/ENTITY would falsely accept e.g.
	// an xs:NMTOKENS alternative for a declared xs:NMTOKEN.
	lexicon.TypeNMTokens: lexicon.TypeAnySimpleType,
	lexicon.TypeIDREFS:   lexicon.TypeAnySimpleType,
	lexicon.TypeENTITIES: lexicon.TypeAnySimpleType,

	// Decimal / integer family.
	lexicon.TypeInteger:            lexicon.TypeDecimal,
	lexicon.TypeNonPositiveInteger: lexicon.TypeInteger,
	lexicon.TypeNegativeInteger:    lexicon.TypeNonPositiveInteger,
	lexicon.TypeLong:               lexicon.TypeInteger,
	lexicon.TypeInt:                lexicon.TypeLong,
	lexicon.TypeShort:              lexicon.TypeInt,
	lexicon.TypeByte:               lexicon.TypeShort,
	lexicon.TypeNonNegativeInteger: lexicon.TypeInteger,
	lexicon.TypeUnsignedLong:       lexicon.TypeNonNegativeInteger,
	lexicon.TypeUnsignedInt:        lexicon.TypeUnsignedLong,
	lexicon.TypeUnsignedShort:      lexicon.TypeUnsignedInt,
	lexicon.TypeUnsignedByte:       lexicon.TypeUnsignedShort,
	lexicon.TypePositiveInteger:    lexicon.TypeNonNegativeInteger,

	// Duration / dateTime family (XSD 1.1).
	lexicon.TypeYearMonthDuration: lexicon.TypeDuration,
	lexicon.TypeDayTimeDuration:   lexicon.TypeDuration,
	lexicon.TypeDateTimeStamp:     lexicon.TypeDateTime,
}

// builtinSimpleDerivedFrom reports whether the XSD built-in simple type whose
// local name is sub is the same as, or transitively derived from, the built-in
// simple type whose local name is super (both XSD-namespace local names). It is
// used where TypeDef.BaseType pointers cannot chain because the built-in types are
// registered without links. Empty names never match.
func builtinSimpleDerivedFrom(sub, super string) bool {
	if sub == "" || super == "" {
		return false
	}
	for cur := sub; cur != ""; cur = builtinSimpleBase[cur] {
		if cur == super {
			return true
		}
	}
	return false
}

// isAnyType reports whether td is the ur-type xs:anyType (the complex root of the
// whole type hierarchy). Its BaseType chain is not pointer-linked from the simple
// types, so callers that need to recognize it compare by expanded name.
func isAnyType(td *TypeDef) bool {
	return td != nil && td.Name.NS == lexicon.NamespaceXSD && td.Name.Local == typeAnyType
}

// isBuiltinSimpleType reports whether td is a built-in (XSD-namespace) SIMPLE type
// definition. The built-in simple types are registered WITHOUT BaseType pointer
// links, so their derivation must be resolved through the builtinSimpleBase table
// rather than by walking BaseType.
func isBuiltinSimpleType(td *TypeDef) bool {
	return td != nil && td.Name.NS == lexicon.NamespaceXSD && !td.IsComplex
}

// strictBuiltinAwareDerivedFrom reports whether sub is validly derived from super
// (Type Derivation OK), additionally consulting the built-in simple-type hierarchy
// when super is a built-in simple type (whose BaseType chain is not linked, so
// isDerivedFrom alone cannot confirm e.g. xs:int ⊂ xs:integer). Unlike the CTA
// substitutability fallback it has NO permissive simple-vs-simple acceptance: it is
// the STRICT derivation predicate used both for the xsi:type-must-derive-from-the-
// declared-type check and as the decisive built-in branch of isValidlySubstitutable.
func strictBuiltinAwareDerivedFrom(sub, super *TypeDef) bool {
	if isDerivedFrom(sub, super) {
		return true
	}
	if !isBuiltinSimpleType(super) {
		return false
	}
	// xs:anySimpleType is the root of the simple-type hierarchy: EVERY simple type —
	// and every complex type with simple content — is validly derived from it,
	// including USER list/union types whose BaseType is not pointer-linked to it (so
	// builtinBaseLocal cannot see them). A type "has simple content" iff its
	// {content type} is a simple type, i.e. ContentType == ContentTypeSimple (true
	// for atomic/list/union simple types AND complex types with <xs:simpleContent>).
	if super.Name.Local == lexicon.TypeAnySimpleType {
		return sub != nil && sub.ContentType == ContentTypeSimple
	}
	return builtinSimpleDerivedFrom(builtinBaseLocal(sub), super.Name.Local)
}
