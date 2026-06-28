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
	// String-family list types.
	lexicon.TypeNMTokens: lexicon.TypeNMToken,
	lexicon.TypeIDREFS:   lexicon.TypeIDREF,
	lexicon.TypeENTITIES: lexicon.TypeENTITY,

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
