package xsd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// msgAbstractType is the validity-error message reported when an element's
// effective type definition is abstract (cvc-elt / cvc-type).
const msgAbstractType = "The type definition is abstract."

// fixedValueMatches reports whether an instance value satisfies a fixed value
// constraint whose declared simple type is td. The comparison is performed in
// the type's value space (XSD 1.1 §3.16, cvc-au/cvc-elt fixed-value rules):
//
//   - The comparison branches on the type's variety *before* applying any
//     whiteSpace facet. A union has no whiteSpace facet of its own, so its raw
//     values are forwarded to fixedUnionMatches, which resolves each value's
//     active member (ordered union semantics) and compares in that member's
//     value space — preserving significant whitespace for an xs:string member
//     that a "collapse" at the union level would have stripped.
//   - For atomic and list types, both values are first whitespace-normalized
//     using the type's *effective* whiteSpace facet, resolved up the derivation
//     chain via resolveWhiteSpace. This honours a facet derived on a restriction
//     (e.g. xs:string restricted with whiteSpace="collapse"), which a
//     builtin-name-only canonicalization would ignore.
//   - For list types, the normalized values are split into items and compared
//     item-by-item in the item type's value space.
//   - For atomic types, the comparison uses the declared builtin's value space:
//     value-comparable builtins (numeric, boolean, date/time, binary including
//     hexBinary/base64Binary) compare via value.Compare, so "0A" == "0a" and
//     "1" == "+1"; QName/NOTATION resolve each lexical QName against its own
//     in-scope namespaces (instanceNS for the instance, fixedNS for the schema
//     fixed value) and compare the resolved {namespace URI, local name}, so two
//     different prefixes bound to the same URI are equal; non-comparable
//     (string-family/anyURI) types compare their whitespace-normalized lexical
//     forms, so a numeric-looking string fixed value "5" does not accept "5.0".
//
// instanceNS and fixedNS carry the in-scope namespace bindings for the instance
// value and the schema fixed value respectively; they are only consulted for
// QName/NOTATION types. When td is nil the comparison falls back to raw string
// equality.
func fixedValueMatches(ctx context.Context, instance, fixed string, td *TypeDef, instanceNS, fixedNS map[string]string, schema *Schema, version Version) bool {
	if td == nil {
		return instance == fixed
	}

	// XSD 1.1: a simpleContent complex type's fixed value lives in its NARROWED
	// content simple type, not the outer complex type's own base chain — compare in
	// the effective content type so e.g. a content type restricted to xs:QName uses
	// QName VALUE-space equality (a different prefix bound to the same URI matches).
	// effectiveContentSimpleType returns a non-simpleContent type unchanged, so the
	// simple (attribute / element / list-item / union-member) callers are
	// unaffected. Centralized here so EVERY caller — runtime element/attribute fixed
	// checks AND the compile-time content-model restriction check
	// (restriction_particle.go) — is consistent. XSD 1.0 keeps the raw declared type.
	if version == Version11 {
		td = effectiveContentSimpleType(td)
	}

	// Branch on variety *before* normalizing with the type's own whiteSpace
	// facet. A union type has no meaningful whiteSpace of its own — each member
	// applies its own facet — so normalizing here (the union default is
	// "collapse") would strip significant whitespace before an xs:string member
	// ever sees it. The union path therefore receives the raw values and lets
	// each member normalize with its own facet. List and atomic types keep their
	// type-level normalization.
	if resolveVariety(td) == TypeVarietyUnion {
		return fixedUnionMatches(ctx, instance, fixed, td, instanceNS, fixedNS, schema, version)
	}

	ws := resolveWhiteSpace(td)
	ni := normalizeWhiteSpace(instance, ws)
	nf := normalizeWhiteSpace(fixed, ws)

	if resolveVariety(td) == TypeVarietyList {
		return fixedListMatches(ctx, ni, nf, td, instanceNS, fixedNS, schema, version)
	}
	return fixedAtomicMatches(ni, nf, builtinBaseLocal(td), instanceNS, fixedNS)
}

// fixedListMatches compares two whitespace-normalized list values item by item
// in the list's item-type value space. Each item is dispatched through the
// variety-aware comparator on the actual item type, so a list whose item type is
// a union (or itself a list) is compared in the correct value space rather than
// raw lexical text.
func fixedListMatches(ctx context.Context, instance, fixed string, td *TypeDef, instanceNS, fixedNS map[string]string, schema *Schema, version Version) bool {
	ii := value.XSDFields(instance)
	fi := value.XSDFields(fixed)
	if len(ii) != len(fi) {
		return false
	}
	itemType := resolveItemType(td)
	for i := range ii {
		if !fixedValueMatches(ctx, ii[i], fi[i], itemType, instanceNS, fixedNS, schema, version) {
			return false
		}
	}
	return true
}

// fixedUnionMatches compares an instance value against a fixed value whose
// declared type is a union, using XSD's *ordered* union semantics. Union
// membership is not "any member that makes the two lexicals compare equal":
// each lexical value has a single ACTIVE member — the first member type, in
// declaration order, that the value fully validates against (facets, lists, and
// nested unions all enforced). The fixed value and the instance value each
// resolve their own active member (the fixed value uses the schema's in-scope
// namespaces, the instance the document's). The comparison is then:
//
//   - If either value has no valid active member, it is not a valid union value,
//     so for fixed-comparison purposes it is treated as not-equal.
//   - If the active member is the same, the two values are compared in *that*
//     member's value space by recursing through fixedValueMatches with the
//     member type (which applies the member's own whiteSpace facet). Thus
//     memberTypes="xs:string xs:integer" with fixed="1" resolves both sides to
//     the xs:string member (the first member, which accepts any text), so "1"
//     and "01" compare as strings and do NOT match.
//   - If the active members DIFFER, the values may still be equal when both
//     members reduce to the same PRIMITIVE value-space family (XSD 1.1 §2.3:
//     restrictions do not create new values). e.g. memberTypes="xs:integer
//     xs:decimal" with fixed="1.0" → active member xs:decimal, instance "1" →
//     active member xs:integer: both reduce to the decimal value space, and
//     1.0 == 1, so they MUST compare equal. This includes string-derived members:
//     fixed "a b" (active in one xs:string restriction) and instance " a   b "
//     (active in another xs:string restriction with whiteSpace="collapse") both
//     reduce to the string value space and denote "a b". The shared family is
//     determined by primitiveValueSpaceFamily; value-comparable families compare
//     with value.Compare, while the string family compares whitespace-normalized
//     lexical forms. Cross-family pairs (xs:string vs xs:integer, xs:integer vs
//     xs:boolean, …) have no shared value space and remain unequal.
//
// The active member is resolved with fixedUnionActiveMember, which reuses the
// same per-member validateValue path the normal (non-fixed) validation uses, so
// the fixed-comparison and ordinary-validation notions of "active member" stay
// consistent.
func fixedUnionMatches(ctx context.Context, instance, fixed string, td *TypeDef, instanceNS, fixedNS map[string]string, schema *Schema, version Version) bool {
	members := resolveUnionMembers(td)

	fixedMember := fixedUnionActiveMember(ctx, fixed, fixedNS, members, schema, version)
	if fixedMember == nil {
		return false
	}
	instanceMember := fixedUnionActiveMember(ctx, instance, instanceNS, members, schema, version)
	if instanceMember == nil {
		return false
	}

	if fixedMember == instanceMember {
		// Same active member: compare in that member's value space. A union has no
		// whiteSpace facet of its own, so the raw values are forwarded and the
		// member normalizes both with its own facet inside fixedValueMatches.
		return fixedValueMatches(ctx, instance, fixed, fixedMember, instanceNS, fixedNS, schema, version)
	}

	// Different active members. XSD 1.1 §2.3 — restrictions do not create new
	// values, so two values are equal iff they denote the same value in their
	// SHARED value space. This is dispatched on each member's variety: list members
	// compare item-by-item in their item type's value space (so an intList member
	// and a decimalList member both denote the same sequence of decimals), and
	// atomic members reduce to their primitive value-space family. Cross-variety
	// pairs (a list member vs an atomic member) have no shared value space and
	// remain unequal.
	return crossMemberValueEqual(ctx, instance, fixed, instanceMember, fixedMember, instanceNS, fixedNS, schema, version)
}

// crossMemberValueComparisonMaxDepth bounds the recursion of
// crossMemberValueEqual so a pathological cyclic type reference (a union or list
// type whose active member resolves back to itself) cannot loop forever. Real
// simple-type variety lattices are shallow; this ceiling is far above any
// legitimate nesting depth.
const crossMemberValueComparisonMaxDepth = 64

// crossMemberValueEqual reports whether two values active in DIFFERENT union
// members denote the same value across the members' shared value space. It is
// FULLY recursive over the entire simple-type variety lattice — atomic, list,
// and union — so NO nesting level is dropped. A union of lists
// (memberTypes="intList decimalList") compares the instance "1 2" (active in
// intList) and the literal "1.0 2.0" (active in decimalList) item-by-item in
// the decimal value space rather than value-comparing the whole multi-token
// strings as scalars; and an item or member type that is itself a union (a
// list-of-union, or a union-of-list-of-union) is resolved to its per-value
// active member and recursed into, so arbitrary nesting bottoms out at atomic
// comparison.
//
// Dispatch, per side's effective variety:
//
//   - UNION (either side): resolve THIS value's active member within the union
//     (via fixedUnionActiveMember) and recurse on the resolved member type. A
//     value with no valid active member has no comparable value, so unequal.
//   - Both LIST: split each value (in its own whiteSpace value space) and compare
//     items pairwise by recursing on the two item types (which may themselves be
//     atomic, list, or union).
//   - Both ATOMIC: if both members are QName-derived (or both NOTATION-derived),
//     compare resolved expanded names so different prefixes bound to the same URI
//     are equal (QName-vs-NOTATION stays unequal). Otherwise reduce each to its
//     primitive value-space family (XSD 1.1 §2.3); equal iff the families match and
//     the values compare equal there (value.Compare for comparable families,
//     normalized-lexical for the string family).
//   - Any other variety mismatch that cannot be reconciled (e.g. list vs atomic):
//     no shared value space → unequal.
func crossMemberValueEqual(ctx context.Context, instance, fixed string, instanceMember, fixedMember *TypeDef, instanceNS, fixedNS map[string]string, schema *Schema, version Version) bool {
	return crossMemberValueEqualDepth(ctx, instance, fixed, instanceMember, fixedMember, instanceNS, fixedNS, 0, schema, version)
}

func crossMemberValueEqualDepth(ctx context.Context, instance, fixed string, instanceMember, fixedMember *TypeDef, instanceNS, fixedNS map[string]string, depth int, schema *Schema, version Version) bool {
	if depth > crossMemberValueComparisonMaxDepth {
		return false
	}
	if instanceMember == nil || fixedMember == nil {
		return false
	}

	instanceVariety := resolveVariety(instanceMember)
	fixedVariety := resolveVariety(fixedMember)

	// UNION on either side: resolve the active member for THAT value and recurse
	// on the resolved member type. This handles a list whose item type is a union,
	// a union nested directly inside another union, and any deeper combination, so
	// the recursion always descends to a non-union variety before comparing.
	if instanceVariety == TypeVarietyUnion {
		active := fixedUnionActiveMember(ctx, instance, instanceNS, resolveUnionMembers(instanceMember), schema, version)
		if active == nil {
			return false
		}
		return crossMemberValueEqualDepth(ctx, instance, fixed, active, fixedMember, instanceNS, fixedNS, depth+1, schema, version)
	}
	if fixedVariety == TypeVarietyUnion {
		active := fixedUnionActiveMember(ctx, fixed, fixedNS, resolveUnionMembers(fixedMember), schema, version)
		if active == nil {
			return false
		}
		return crossMemberValueEqualDepth(ctx, instance, fixed, instanceMember, active, instanceNS, fixedNS, depth+1, schema, version)
	}

	if instanceVariety == TypeVarietyList && fixedVariety == TypeVarietyList {
		ni := normalizeWhiteSpace(instance, resolveWhiteSpace(instanceMember))
		nf := normalizeWhiteSpace(fixed, resolveWhiteSpace(fixedMember))
		ii := value.XSDFields(ni)
		fi := value.XSDFields(nf)
		if len(ii) != len(fi) {
			return false
		}
		instanceItem := resolveItemType(instanceMember)
		fixedItem := resolveItemType(fixedMember)
		if instanceItem == nil || fixedItem == nil {
			return false
		}
		for i := range ii {
			if !crossMemberValueEqualDepth(ctx, ii[i], fi[i], instanceItem, fixedItem, instanceNS, fixedNS, depth+1, schema, version) {
				return false
			}
		}
		return true
	}

	if instanceVariety != TypeVarietyAtomic || fixedVariety != TypeVarietyAtomic {
		return false
	}

	// Both atomic. When the two item/member types are the SAME (e.g. both list
	// items are xs:integer), compare in that one type's value space directly so a
	// QName/NOTATION item pair resolves namespaces rather than being dropped by the
	// no-shared-family rule.
	if instanceMember == fixedMember {
		return fixedValueMatches(ctx, instance, fixed, fixedMember, instanceNS, fixedNS, schema, version)
	}

	fixedLocal := builtinBaseLocal(fixedMember)
	instanceLocal := builtinBaseLocal(instanceMember)

	// QName/NOTATION have no shared primitive family in primitiveValueSpaceFamily
	// (their equality is namespace-context dependent, not a value/lexical compare),
	// so handle them here before that fallback. When BOTH members are QName-derived
	// (or BOTH NOTATION-derived), normalize each side with its member's effective
	// whiteSpace facet and compare the resolved expanded names: cross-member equality
	// holds iff both resolve to the same {namespace, local}, even when the two
	// members bind different prefixes to the same URI. QName-vs-NOTATION stays
	// unequal (no shared value space).
	instanceIsQName := instanceLocal == lexicon.TypeQName
	fixedIsQName := fixedLocal == lexicon.TypeQName
	instanceIsNotation := instanceLocal == lexicon.TypeNotation
	fixedIsNotation := fixedLocal == lexicon.TypeNotation
	if (instanceIsQName && fixedIsQName) || (instanceIsNotation && fixedIsNotation) {
		ni := normalizeWhiteSpace(instance, resolveWhiteSpace(instanceMember))
		nf := normalizeWhiteSpace(fixed, resolveWhiteSpace(fixedMember))
		iqn, ierr := resolveLexicalQName(ni, instanceNS)
		if ierr != nil {
			return false
		}
		fqn, ferr := resolveLexicalQName(nf, fixedNS)
		if ferr != nil {
			return false
		}
		return iqn == fqn
	}

	fixedFamily, fComparable, fok := primitiveValueSpaceFamily(fixedLocal)
	instanceFamily, _, iok := primitiveValueSpaceFamily(instanceLocal)
	if !fok || !iok || fixedFamily != instanceFamily {
		return false
	}
	// Normalize each operand with ITS active member's effective whiteSpace facet
	// before comparing, so an instance " 1 " (whose member collapses the spaces)
	// or " a   b " (whose member collapses to "a b") is reduced to its value-space
	// form first.
	ni := normalizeWhiteSpace(instance, resolveWhiteSpace(instanceMember))
	nf := normalizeWhiteSpace(fixed, resolveWhiteSpace(fixedMember))
	if !fComparable {
		// String-family: the value space equals the whitespace-processed lexical
		// space, so compare the normalized lexical forms directly.
		return ni == nf
	}
	cmp, ok := value.Compare(ni, nf, fixedFamily)
	return ok && cmp == 0
}

// primitiveValueSpaceFamily maps a builtin's local name to the local name of the
// PRIMITIVE built-in whose value space it shares, for cross-member fixed-value
// comparison (XSD 1.1 §2.3: restrictions do not create new values). It returns
// (family, comparable, true) for every type with a recognized primitive ancestor,
// where:
//
//   - family is a stable key identifying the shared primitive value space. All
//     xs:decimal-derived integer types collapse to "decimal"; all xs:string-derived
//     types (string, normalizedString, token, language, Name, NCName, NMTOKEN,
//     IDREF, ENTITY, …) and anyURI collapse to "string"; every other primitive
//     (boolean, float, double, each date/time-family type, hexBinary,
//     base64Binary) is its own family.
//   - comparable is true when value.Compare implements value-space equality for
//     that family (the enumValueSpaceTypes allowlist); for the "string" family it
//     is false, so callers compare whitespace-normalized lexical forms instead
//     (the string value space equals the whitespace-processed lexical space).
//
// QName/NOTATION return ("", false, false): their equality is namespace-context
// dependent, not a cross-member value/lexical comparison, so they have no shared
// primitive family for this path.
func primitiveValueSpaceFamily(builtinLocal string) (string, bool, bool) {
	switch builtinLocal {
	case lexicon.TypeQName, lexicon.TypeNotation, "":
		return "", false, false
	case lexicon.TypeDecimal, lexicon.TypeInteger,
		lexicon.TypeNonPositiveInteger, lexicon.TypeNegativeInteger, lexicon.TypeLong, lexicon.TypeInt, lexicon.TypeShort, lexicon.TypeByte,
		lexicon.TypeNonNegativeInteger, lexicon.TypeUnsignedLong, lexicon.TypeUnsignedInt, lexicon.TypeUnsignedShort,
		lexicon.TypeUnsignedByte, lexicon.TypePositiveInteger:
		return lexicon.TypeDecimal, true, true
	case lexicon.TypeString, lexicon.TypeNormalizedString, "token", "language",
		"Name", "NCName", "ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES",
		"NMTOKEN", "NMTOKENS", "anyURI":
		// String value space equals the whitespace-processed lexical space; not
		// value-comparable via value.Compare, so the caller compares lexically.
		return lexicon.TypeString, false, true
	case lexicon.TypeDateTimeStamp:
		// XSD 1.1 subtype of xs:dateTime; compares in the dateTime value space.
		return "dateTime", true, true
	case lexicon.TypeDayTimeDuration, lexicon.TypeYearMonthDuration:
		// XSD 1.1 subtypes of xs:duration; compare in the duration value space.
		return "duration", true, true
	default:
		// Remaining comparable primitives (boolean, float, double, date/time
		// family, binary) are gated on the same allowlist the enumeration path uses.
		if _, ok := enumValueSpaceTypes[builtinLocal]; !ok {
			return "", false, false
		}
		return builtinLocal, true, true
	}
}

// fixedUnionActiveMember returns the active BASIC (atomic) member type for a
// value within a union: the first member (in declaration order) the value fully
// validates against, descending through nested unions to the basic member that
// actually accepts the value. It reuses the validateValue path so the validity
// criteria match the main validation engine exactly (facets, list items, nested
// unions, and QName/NOTATION namespace resolution). Errors are discarded via a
// suppressing validation context with a NilErrorHandler. Returns nil when no
// member accepts the value.
//
// Descending into nested unions matters for cross-member value-space comparison:
// an outer member that is itself a union must contribute its active basic member
// (e.g. xs:integer), not the union TypeDef, so valueSpaceFamily can reduce it to
// the comparable family (decimal) and compare it against a sibling decimal
// member's value.
//
// version is the schema's effective XSD version, threaded from the caller so the
// throwaway validation context applies the same version-sensitive lexical rules
// the main validation path uses — e.g. a 1.1-only lexical form ("+INF" for
// xs:double) appearing INSIDE a union fixed-value or enumeration literal is
// accepted in 1.1 mode rather than rejected under a defaulted Version10.
// fixedUnionActiveMember returns the union member that accepts value. The schema
// argument (nil when none is available) is threaded onto the throwaway
// validation context so a member whose own xs:assertion needs schema-aware
// resolution (e.g. `castable as t:T`) validates the same way as the real path;
// the per-validation cast guard flows through ctx. Most callers pass nil
// (schema-awareness is immaterial for plain fixed/enumeration comparisons); the
// assertion $value path passes the real schema so union member typing is
// consistent with validation.
func fixedUnionActiveMember(ctx context.Context, value string, valueNS map[string]string, members []*TypeDef, schema *Schema, version Version) *TypeDef {
	for _, member := range members {
		vc := &validationContext{
			schema:        schema,
			errorHandler:  helium.NilErrorHandler{},
			suppressDepth: 1,
			version:       version,
		}
		if validateValue(ctx, value, valueNS, member, "", "", 0, vc) != nil {
			continue
		}
		// The member accepts the value. If it is itself a union, recurse to find
		// the active basic member within it; the validateValue success above
		// guarantees at least one nested member accepts the value.
		if resolveVariety(member) == TypeVarietyUnion {
			if basic := fixedUnionActiveMember(ctx, value, valueNS, resolveUnionMembers(member), schema, version); basic != nil {
				return basic
			}
		}
		return member
	}
	return nil
}

// fixedAtomicMatches compares two already whitespace-normalized atomic values in
// the builtin type's value space. QName/NOTATION resolve each side's prefix
// against its own in-scope namespaces and compare the resolved {URI, local}.
// Other value-comparable builtins use value.Compare (covering numeric, boolean,
// date/time, and binary value spaces); everything else falls back to exact
// equality of the normalized lexical forms.
func fixedAtomicMatches(instance, fixed, builtinLocal string, instanceNS, fixedNS map[string]string) bool {
	if builtinLocal == lexicon.TypeQName || builtinLocal == lexicon.TypeNotation {
		iqn, ierr := resolveLexicalQName(instance, instanceNS)
		fqn, ferr := resolveLexicalQName(fixed, fixedNS)
		// A prefix that cannot be resolved makes the QName/NOTATION itself invalid;
		// the fixed comparison must NOT fall back to raw lexical equality (which
		// would wrongly accept a fixed "s:name" against an instance "s:name" that
		// has no binding for s). Reject instead.
		if ierr != nil || ferr != nil {
			return false
		}
		return iqn == fqn
	}
	if _, ok := enumValueSpaceTypes[builtinLocal]; ok {
		if (builtinLocal == lexicon.TypeFloat || builtinLocal == lexicon.TypeDouble) &&
			value.IsFloatNaN(instance) && value.IsFloatNaN(fixed) {
			return true
		}
		if cmp, ok := value.Compare(instance, fixed, builtinLocal); ok {
			return cmp == 0
		}
	}
	return instance == fixed
}

type validationContext struct {
	schema        *Schema
	version       Version // XSD spec version governing version-sensitive lexical rules
	cfg           *validateConfig
	filename      string
	errorHandler  helium.ErrorHandler
	suppressDepth int
	// edcType is the complex type whose content model is currently being matched.
	// It is set (and restored) by validateContentByType, so the wildcard
	// Element-Declarations-Consistent check (validateWildcardElementConsistent) can
	// consult the type's BASE chain for a same-named local element declaration: a
	// derived type's wildcard may match an element the base type declared locally
	// with a different type, which the dynamic EDC check must reject even when the
	// derived content model itself no longer declares it. XSD 1.1 only.
	edcType *TypeDef
	// skipContentNodes records every element node inside a processContents="skip"
	// wildcard-matched subtree (and the matched elements themselves). Such content
	// is NOT schema-assessed, so a pass-2 identity-constraint selector must not
	// pick it: an xs:key/xs:unique selecting an unassessed skip-content element
	// would impose key/uniqueness on a node that carries no PSVI contribution.
	// Populated by annotateSkipChildren; consulted by evaluateIDC. XSD 1.1 only.
	skipContentNodes map[helium.Node]struct{}
	// actualElemType records the ACTUAL *TypeDef determined for each element
	// during pass-1 content validation, including any xsi:type override. Pass-2
	// identity-constraint field resolution consults this before falling back to
	// descending the declared content model, so an IDC field whose type is
	// contributed by xsi:type is canonicalized in the correct value space.
	actualElemType map[*helium.Element]*TypeDef
	// actualElemDecl records the resolved *ElementDecl matched for each element
	// instance during pass-1, including LOCAL declarations buried inside content
	// models (which lookupElemDecl, finding only GLOBAL declarations, cannot
	// recover). It is written AS SOON AS a child MATCHES a particle (recordElemDecl)
	// — BEFORE the child's content is validated/assessed — so a partially-satisfied
	// occurrence (e.g. an unsatisfied minOccurs) still records the matched decl.
	// Pass-2 identity-constraint evaluation consults this map (for the host decl and
	// its IDCs / default / fixed / nillable metadata) BEFORE falling back to
	// lookupElemDecl, so xs:key/xs:unique/xs:keyref declared on a local element are
	// evaluated rather than silently skipped. Because it is written pre-assessment,
	// it must NOT be used as a "was assessed" signal — the ID/IDREF pass uses
	// assessedElemType for that.
	actualElemDecl map[*helium.Element]*ElementDecl
	// assertAnnotations maps assessed element and attribute nodes to their XSD
	// type name (the xpath3 annotation form, e.g. "xs:integer"). It is populated
	// during validation in XSD 1.1 mode (nil otherwise) so xs:assert tests
	// evaluate against a PSVI-typed tree: a typed attribute like @length atomizes
	// to xs:nonNegativeInteger rather than xs:untypedAtomic (which a value
	// comparison would cast to xs:string), and "instance of" tests see the
	// declared type. Unassessed skip/lax-no-declaration content is deliberately
	// excluded even when actualElemType records an xsi:type for IDC canonicalization.
	assertAnnotations TypeAnnotations
	// assertAnonTypes / assertAnonNames register INLINE ANONYMOUS list/union simple
	// types under stable synthetic annotation names (Q{assertAnonNS}N). An anonymous
	// type has no schema-table name, so xsdTypeName collapses it to a named ancestor
	// (or xs:anyType) and its list-item / union-member metadata would be lost to
	// xs:assert node atomization. Recording the actual *TypeDef lets schemaDecls
	// recover that metadata. assertAnonNames dedups by *TypeDef so one inline type
	// gets one stable name across all nodes that use it.
	assertAnonTypes map[string]*TypeDef
	assertAnonNames map[*TypeDef]string
	// assertEffectiveValues records, per EMPTY element node that has a schema
	// default/fixed value, the effective (schema-normalized) value and the namespace
	// context to resolve a QName/NOTATION default's prefix (the DECLARATION's
	// context). isolatedAssertTree materializes these onto the isolated copy so
	// data(c) on a DEFAULTED descendant atomizes the default rather than "" —
	// matching the asserted element's own $value (which already substitutes it).
	assertEffectiveValues map[helium.Node]assertEffectiveValue
	// attrInheritable records, for XSD 1.1, the instance attribute nodes matched to
	// an AttrUse whose {inheritable} is true. The top-down validation walk populates
	// it for every ancestor before a descendant's conditional type assignment runs,
	// so inheritedAttributes can resolve a CTA/assertion @test against inherited
	// ancestor attributes.
	attrInheritable map[*helium.Attribute]struct{}
	// assessedElemType records the ACTUAL *TypeDef of each element that was truly
	// SCHEMA-ASSESSED during pass-1 — the validation root, a content-model particle
	// match whose content was actually validated, or an xs:anyType/lax child WITH a
	// matching global declaration (all post-xsi:type). It is the element-side
	// counterpart of actualAttrType. It is deliberately NOT populated by
	// annotateSkipChildren or the lax-no-declaration branch (which write
	// actualElemType purely for pass-2 IDC canonicalization), NOR at the
	// recordElemDecl match site (which fires before assessment), so an element
	// admitted through a processContents="skip" wildcard — even one carrying
	// xsi:type="xs:ID" — and a matched-but-unassessed child (e.g. an unsatisfied
	// minOccurs) are both absent here. The XSD 1.1 ID/IDREF pass uses ONLY this map
	// for element typing (never actualElemType and never actualElemDecl), so neither
	// skip content nor matched-but-failed children are mistaken for an xs:ID/xs:IDREF.
	assessedElemType map[*helium.Element]*TypeDef
	// actualAttrType records the declared *TypeDef of each attribute that was
	// actually SCHEMA-ASSESSED during pass-1 — matched by an explicit attribute
	// use, or admitted by a strict/lax xs:anyAttribute wildcard with a matching
	// global declaration, or inserted as a default/fixed value. An attribute
	// admitted by a processContents="skip" wildcard is NOT assessed and is
	// therefore absent here. The XSD 1.1 ID/IDREF pass consults ONLY this map so a
	// skip-admitted attribute is never mistaken for an xs:ID/xs:IDREF via a global
	// fallback (which would false-reject duplicate skipped IDs).
	actualAttrType map[*helium.Attribute]*TypeDef
}

// assertEffectiveValue is a recorded element default/fixed effective value plus the
// namespace context used to resolve a QName/NOTATION value's prefix.
type assertEffectiveValue struct {
	value string
	ns    map[string]string
	td    *TypeDef
	// qname is true when the effective value's active type carries xs:QName or
	// xs:NOTATION values, including list item types and active union members. The
	// isolated assert tree then materializes every QName/NOTATION token's prefix
	// against ns; a non-QName value that happens to contain a colon is appended
	// verbatim with no prefix rewrite.
	qname bool
}

// assertAnonNS is the synthetic namespace for inline anonymous list/union type
// annotation names recorded for xs:assert node atomization (see assertAnonTypes).
const assertAnonNS = "urn:x-helium:assert-anon"

// pendingKeyRef is an evaluated keyref table awaiting resolution against the
// key/unique tables built for the SAME host-element occurrence (XSD
// identity-constraint scope), once every key/unique on that occurrence has been
// evaluated.
type pendingKeyRef struct {
	idc   *IDConstraint
	table *idcTable
}

func newValidationContext(schema *Schema, cfg *validateConfig, filename string, handler helium.ErrorHandler) *validationContext {
	var version Version
	if schema != nil {
		version = schema.version
	}
	vc := &validationContext{
		schema:           schema,
		version:          version,
		cfg:              cfg,
		filename:         filename,
		errorHandler:     handler,
		actualElemType:   make(map[*helium.Element]*TypeDef),
		actualElemDecl:   make(map[*helium.Element]*ElementDecl),
		attrInheritable:  make(map[*helium.Attribute]struct{}),
		assessedElemType: make(map[*helium.Element]*TypeDef),
		actualAttrType:   make(map[*helium.Attribute]*TypeDef),
	}
	if version == Version11 {
		vc.assertAnnotations = make(TypeAnnotations)
		vc.assertAnonTypes = make(map[string]*TypeDef)
		vc.assertAnonNames = make(map[*TypeDef]string)
		vc.assertEffectiveValues = make(map[helium.Node]assertEffectiveValue)
		vc.skipContentNodes = make(map[helium.Node]struct{})
	}
	return vc
}

// validationErrors is a synchronous ErrorHandler that accumulates error
// strings in order. Used internally by ValidateElement and tests.
type validationErrors struct {
	errors []string
}

func (ve *validationErrors) Handle(_ context.Context, err error) {
	ve.errors = append(ve.errors, err.Error())
}

// reportValidityError formats a validation error and sends it to the ErrorHandler.
func (vc *validationContext) reportValidityError(ctx context.Context, file string, line int, elemName, msg string) {
	if vc.suppressDepth > 0 {
		return
	}
	ve := &ValidationError{
		Filename: file,
		Line:     line,
		Element:  elemName,
		Message:  msg,
	}
	vc.errorHandler.Handle(ctx, newLeveledValidationError(ve, helium.ErrorLevelError))
}

// reportValidityErrorAttr formats an attribute validation error and sends it to the ErrorHandler.
func (vc *validationContext) reportValidityErrorAttr(ctx context.Context, file string, line int, elemName, attrName, msg string) {
	if vc.suppressDepth > 0 {
		return
	}
	ve := &ValidationError{
		Filename:      file,
		Line:          line,
		Element:       elemName,
		AttributeName: attrName,
		Message:       msg,
	}
	vc.errorHandler.Handle(ctx, newLeveledValidationError(ve, helium.ErrorLevelError))
}

// Validate validates a lexical value against this simple type definition.
// nsMap provides prefix-to-URI mappings for QName/NOTATION resolution and may be nil.
func (td *TypeDef) Validate(ctx context.Context, value string, nsMap map[string]string) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	if td.ContentType != ContentTypeSimple {
		return fmt.Errorf("type %q is not a simple type", typeQualifiedName(td))
	}
	// Standalone simple-type validation has no schema context, so it applies the
	// default (Version10) lexical rules.
	vc := &validationContext{
		errorHandler: helium.NilErrorHandler{},
	}
	return validateValue(ctx, value, nsMap, td, "", "", 0, vc)
}

// ValidateElement validates an element's content against this type definition.
// This is used by XSLT xsl:type validation where the element is constructed
// in the result tree and must conform to the given type.
func (td *TypeDef) ValidateElement(ctx context.Context, elem *helium.Element, schema *Schema) error {
	return td.ValidateElementAnnotated(ctx, elem, schema, nil)
}

// ValidateElementAnnotated validates an element's content against this type
// definition, and — when ann is non-nil — records PSVI type annotations for the
// element's descendant elements and attributes into *ann, keyed on the LIVE
// nodes of elem's subtree. The root element's own annotation is the caller's
// responsibility (it is the type named by the xsl:type/type attribute, not a
// content-model match). This is used by XSLT xsl:type validation so that later
// element(*, T) / schema-element() type tests see the schema types of the
// constructed subtree, not just the root.
func (td *TypeDef) ValidateElementAnnotated(ctx context.Context, elem *helium.Element, schema *Schema, ann *TypeAnnotations) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	collector := &validationErrors{}
	cfg := &validateConfig{}
	if ann != nil {
		if *ann == nil {
			*ann = make(TypeAnnotations)
		}
		cfg.annotations = ann
	}
	vc := newValidationContext(schema, cfg, "", collector)
	err := vc.validateElementContent(ctx, elem, nil, td)
	if err == nil {
		return nil
	}
	if len(collector.errors) > 0 {
		var b strings.Builder
		for _, e := range collector.errors {
			b.WriteString(e)
		}
		return fmt.Errorf("%s", strings.TrimSpace(b.String()))
	}
	return err
}

func validateDocument(ctx context.Context, doc *helium.Document, schema *Schema, cfg *validateConfig, handler helium.ErrorHandler) bool {
	filename := cfg.label
	if filename == "" {
		filename = doc.URL()
	}
	if filename == "" {
		filename = "(string)"
	}
	valid := true
	vc := newValidationContext(schema, cfg, filename, handler)

	// Initialize annotations map if requested.
	if cfg.annotations != nil && *cfg.annotations == nil {
		*cfg.annotations = make(TypeAnnotations)
	}
	// Initialize nilled elements map if requested.
	if cfg.nilledElements != nil && *cfg.nilledElements == nil {
		*cfg.nilledElements = make(NilledElements)
	}

	root := findDocumentElement(doc)
	if root == nil {
		return false
	}

	// Walk the document tree for content model validation.
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		if err := vc.validateElement(ctx, elem); err != nil {
			valid = false
		}
		return nil
	}))

	// Second walk: evaluate identity constraints (xs:key, xs:keyref, xs:unique).
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		// Choose the declaration whose identity constraints apply to this element
		// instance. idcHostDecl uses the non-ref declaration recorded during pass-1
		// if one is present — even when it carries zero IDCs — because a local
		// element that merely shadows a same-named global must NOT inherit the
		// global's IDCs. lookupElemDecl finds only GLOBAL declarations, so IDCs on a
		// local element would otherwise be silently skipped. It falls back to the
		// global lookup only when no declaration was recorded OR the recorded one is
		// a ref: an <xs:element ref="g"> matches a ref declaration (IsRef) that does
		// NOT copy the global's IDCs (IDCs are a property of the referenced global
		// declaration), so for a ref the global lookup is the one that carries the
		// constraints.
		edecl := vc.idcHostDecl(elem)
		if edecl != nil && len(edecl.IDCs) > 0 {
			if err := vc.validateIDConstraints(ctx, elem, edecl); err != nil {
				valid = false
			}
		}
		return nil
	}))

	// Third walk: XSD 1.1 document-wide xs:ID / xs:IDREF / xs:IDREFS validation.
	// Gated to 1.1 so XSD 1.0 stays byte-identical (helium does not enforce these
	// datatype constraints in 1.0, and the libxml2-compat goldens depend on that).
	if vc.version == Version11 {
		if !vc.validateIDIDREF(ctx, doc) {
			valid = false
		}
	}

	// Fourth walk: XSD 1.1 document-wide xs:ENTITY / xs:ENTITIES value-space
	// validation (cvc-id / §3.3.11). Gated to 1.1 so XSD 1.0 stays byte-identical
	// (helium validates these datatypes only lexically in 1.0).
	if vc.version == Version11 {
		if !vc.validateEntities(ctx, doc) {
			valid = false
		}
	}

	return valid
}

func (vc *validationContext) validateElement(ctx context.Context, elem *helium.Element) error {
	parent := elem.Parent()
	if parent == nil || parent.Type() == helium.DocumentNode {
		// Root element — must match a global element declaration.
		return vc.validateRootElement(ctx, elem)
	}
	// Non-root elements are validated by their parent's content model.
	return nil
}

func (vc *validationContext) validateRootElement(ctx context.Context, elem *helium.Element) error {
	local := elem.LocalName()
	ns := elem.URI()
	// Match on the element's full expanded name. An element with a non-empty
	// namespace must NOT fall back to an unqualified declaration that merely
	// shares the local name: cvc-elt requires the instance and declaration
	// expanded names to be identical (libxml2 rejects {urn:wrong}foo against a
	// no-namespace schema declaring {}foo).
	edecl, ok := vc.schema.LookupElement(local, ns)
	if !ok {
		msg := "No matching global declaration available for the validation root."
		vc.reportValidityError(ctx, vc.filename, elem.Line(), local, msg)
		return fmt.Errorf("no matching global declaration")
	}

	// Keep edecl as the ACTUAL root declaration so its own Nillable flag is
	// honored by the nilled-element check. For a no-type substitution-group
	// member, the effective TYPE is inherited from the head (effectiveDeclType
	// walks the substitutionGroup chain), but the declaration — and thus the
	// nillable flag — stays the member's. This mirrors the particle paths.
	declType := effectiveDeclType(edecl, vc.schema)
	if declType == nil {
		return nil
	}

	// XSD 1.1 conditional type assignment: the alternatives may select a
	// different governing type. xsi:type (resolved next) still takes precedence.
	declType = vc.applyTypeAlternatives(ctx, elem, edecl, declType)

	td, err := vc.resolveXsiType(ctx, elem, declType, vc.hasTypeTable(edecl))
	if err != nil {
		return err
	}
	// Check block flags against xsi:type derivation. A blocked xsi:type is a
	// validity error (cvc-elt.4.3): fail rather than silently fall back to the
	// declared type — otherwise a blocked narrowing whose value is also valid under
	// the declared type (e.g. xsi:type="xs:int" / declared xs:integer with
	// block="restriction") would be wrongly accepted. This mirrors the per-child
	// match sites, which already treat a blocked xsi:type as a content error.
	if td != declType && isDerivationBlocked(td, declType, edecl.Block) {
		msg := "The xsi:type definition is blocked by the element declaration."
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return fmt.Errorf("blocked xsi:type")
	}
	if td != nil && td.Abstract {
		msg := msgAbstractType
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return fmt.Errorf("abstract type")
	}

	// Annotate root element with its type and record its declaration.
	vc.annotateElement(ctx, elem, td, true)
	vc.recordElemDecl(elem, edecl)

	nilled, err := vc.checkXsiNil(ctx, elem)
	if err != nil {
		return err
	}
	if nilled {
		return vc.validateNilledElement(ctx, elem, edecl, td)
	}

	return vc.validateElementContent(ctx, elem, edecl, td)
}

func (vc *validationContext) validateElementContent(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) error {
	// XSD 1.1: a governing type of xs:error (selected by conditional type
	// assignment, or referenced directly) has an empty value space, so any element
	// it governs is invalid. This is the single choke point for every type-selection
	// site (root and the per-child content-model matches).
	if vc.version == Version11 && isErrorType(td) {
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem),
			"The element is not valid: the conditional type assignment selected the type xs:error.")
		return fmt.Errorf("xs:error type selected")
	}

	// Validate attributes and annotate them.
	if err := vc.validateAttributes(ctx, elem, td); err != nil {
		return err
	}

	if err := vc.validateContentByType(ctx, elem, edecl, td); err != nil {
		return err
	}

	// XSD 1.1: xs:assert constraints are evaluated against the element once its
	// attributes and content have been validated. edecl carries the element's
	// default/fixed value so $value reflects the effective simple value for an
	// empty element.
	if vc.version == Version11 {
		return vc.checkAssertions(ctx, elem, edecl, td)
	}
	return nil
}

// rejectNonWhitespaceText reports a validity error and returns a non-nil error
// if elem has any non-whitespace text or CDATA child. It is used for element-only
// content types (including the XSD 1.1 synthesized empty+openContent type), where
// character content other than whitespace is not allowed.
func (vc *validationContext) rejectNonWhitespaceText(ctx context.Context, elem *helium.Element) error {
	for child := range helium.Children(elem) {
		if child.Type() != helium.TextNode && child.Type() != helium.CDATASectionNode {
			continue
		}
		// Use XSD/XML whitespace (space, tab, CR, LF) only: characters like
		// NBSP (U+00A0) are NOT ignorable in element-only content, so
		// strings.TrimSpace (which strips all Unicode space) must not be used.
		if !xmlchar.IsAllSpace(child.Content()) {
			msg := "Character content other than whitespace is not allowed because the content type is 'element-only'."
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
			return fmt.Errorf("text content in element-only type")
		}
	}
	return nil
}

// validateContentByType validates an element's content against its type's
// content-type (empty/simple/element-only/mixed). Attribute validation and the
// XSD 1.1 assertion check are handled by the caller (validateElementContent).
func (vc *validationContext) validateContentByType(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) error {
	// Record the type whose content model is about to be matched so a wildcard
	// match inside it can consult this type's base chain for a same-named local
	// element declaration (dynamic EDC). Restored on exit so a nested element's
	// own content validation does not leak its type to the parent's match.
	prevEDC := vc.edcType
	vc.edcType = td
	defer func() { vc.edcType = prevEDC }()

	switch td.ContentType {
	case ContentTypeEmpty:
		// XSD 1.1 §3.4.2.3.3: an empty explicit content type plus effective open
		// content (mode != none) becomes element-only with an empty particle plus
		// the open content, so extra wildcard-matched children are admitted.
		if vc.version == Version11 && td.OpenContent != nil {
			// The synthesized type is element-only, so non-whitespace character
			// content is not allowed even though extra wildcard-matched children are.
			if err := vc.rejectNonWhitespaceText(ctx, elem); err != nil {
				return err
			}
			mg := td.ContentModel
			if mg == nil {
				mg = &ModelGroup{Compositor: CompositorSequence, MinOccurs: 1, MaxOccurs: 1}
			}
			return vc.validateContentModelOpen(ctx, elem, mg, td.OpenContent)
		}
		return vc.validateEmptyContent(ctx, elem, false)
	case ContentTypeSimple:
		return vc.validateSimpleContent(ctx, elem, edecl, td)
	case ContentTypeElementOnly, ContentTypeMixed:
		// XSD 1.1: a genuinely-EMPTY element-only content type — an empty
		// <xs:sequence/>, an empty model group, or no model group at all — with no
		// effective open content has an "empty" content type per §3.4.2, so it must
		// reject ALL character content INCLUDING whitespace (cvc-complex-type.2.1),
		// not merely non-whitespace text. read_types.go classifies an empty
		// compositor as element-only with a non-nil empty model group, so the
		// emptiness is detected via modelGroupHasContent here rather than at
		// classification time. XSD 1.0 keeps the historical whitespace tolerance
		// (rejectNonWhitespaceText below), byte-identical.
		if td.ContentType == ContentTypeElementOnly && vc.version == Version11 && td.OpenContent == nil &&
			(td.ContentModel == nil || !modelGroupHasContent(td.ContentModel)) {
			return vc.validateEmptyContent(ctx, elem, true)
		}
		// For element-only content, non-whitespace text children are not allowed.
		if td.ContentType == ContentTypeElementOnly {
			if err := vc.rejectNonWhitespaceText(ctx, elem); err != nil {
				return err
			}
		}
		if td.ContentModel == nil {
			// XSD 1.1: effective open content on a type with NO declared model group
			// (e.g. a mixed or appliesToEmpty type that picked up a
			// <xs:defaultOpenContent>) turns the empty declared content into an empty
			// model plus the open content, so every child element must match the open
			// wildcard — it does NOT admit arbitrary children. Element-only text was
			// already rejected above; mixed text stays allowed (the open-content
			// matcher inspects child elements only).
			if vc.version == Version11 && td.OpenContent != nil {
				mg := &ModelGroup{Compositor: CompositorSequence, MinOccurs: 1, MaxOccurs: 1}
				return vc.validateContentModelOpen(ctx, elem, mg, td.OpenContent)
			}
			// No content model and no open content means anything goes (for mixed) or
			// empty (for element-only).
			if td.ContentType == ContentTypeElementOnly {
				return vc.validateEmptyContent(ctx, elem, false)
			}
			// Mixed content with no model group (xs:anyType and similar lax/open
			// content) admits arbitrary child elements. Pass 2 IDC evaluation can
			// still reach descendants of this subtree, so each child must be
			// lax-annotated with its ACTUAL type (honoring xsi:type) and recursed
			// into — otherwise resolveFieldType falls back to declared types and
			// misses xsi:type overrides on descendants.
			return vc.annotateAnyTypeChildren(ctx, elem)
		}
		// XSD 1.1 open content admits extra wildcard-matched children beyond the
		// declared model (interleaved or as a suffix).
		if vc.version == Version11 && td.OpenContent != nil {
			return vc.validateContentModelOpen(ctx, elem, td.ContentModel, td.OpenContent)
		}
		return vc.validateContentModel(ctx, elem, td.ContentModel)
	}
	return nil
}

// assessLaxElement performs XSD lax assessment of an element that matched a
// processContents="lax" wildcard (or is a child of an xs:anyType element) and has
// NO element declaration. Per XSD lax: if a governing type can be found — here via
// xsi:type — the element must be ·valid· against it and IS schema-assessed, so it
// is validated against that type and recorded with assessed=true (its
// xs:ID/xs:IDREF content then participates in the document-wide ID/IDREF pass).
// With no resolvable xsi:type the element is not assessed; only its subtree is
// walked to annotate deeper descendants for pass-2 IDC canonicalization.
//
// An undeclared element has no nillable declaration, so xsi:nil cannot make it
// nil: its content is ALWAYS validated against the governing type (a nilled
// element with non-empty content, or empty content the type forbids such as
// xs:int, is rejected; empty content a type permits stays valid). checkXsiNil
// still runs to surface a malformed xsi:nil boolean.
func (vc *validationContext) assessLaxElement(ctx context.Context, ce *helium.Element) error {
	actual, hasType := vc.resolveXsiTypeQuiet(ce)
	if !hasType {
		return vc.annotateAnyTypeChildren(ctx, ce)
	}
	if actual != nil && actual.Abstract {
		vc.reportValidityError(ctx, vc.filename, ce.Line(), elemDisplayName(ce), msgAbstractType)
		return fmt.Errorf("abstract type")
	}
	vc.annotateElement(ctx, ce, actual, true)
	if actual == nil {
		return nil
	}
	if _, nilErr := vc.checkXsiNil(ctx, ce); nilErr != nil {
		return nilErr
	}
	return vc.validateElementContent(ctx, ce, nil, actual)
}

// annotateAnyTypeChildren lax-validates the child elements of an xs:anyType (or
// other mixed, model-group-less) element. There is no content model to walk, so
// children are validated like elements matched by a lax wildcard: each child's
// global element declaration is consulted (skipped when absent), its xsi:type
// override is resolved, the resulting ACTUAL type is recorded via annotateElement,
// and validation recurses into the child. This populates actualElemType for every
// descendant that pass-2 IDC resolution can inspect, so xsi:type on descendants is
// honored during key canonicalization.
func (vc *validationContext) annotateAnyTypeChildren(ctx context.Context, elem *helium.Element) error {
	var contentErr error
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		edecl := lookupElemDecl(ce, vc.schema)
		if edecl == nil {
			// Lax with no global declaration: assess the child against its xsi:type
			// (if resolvable), else recurse to annotate deeper descendants.
			if err := vc.assessLaxElement(ctx, ce); err != nil {
				contentErr = err
			}
			continue
		}
		// XSD 1.1 conditional type assignment applies to a global element reached
		// through xs:anyType too: select the alternative type BEFORE resolving
		// xsi:type (xsi:type still wins), mirroring the established order at the
		// explicit-particle/wildcard match sites.
		declType := vc.applyTypeAlternatives(ctx, ce, edecl, effectiveDeclType(edecl, vc.schema))
		td, xsiErr := vc.resolveXsiType(ctx, ce, declType, vc.hasTypeTable(edecl))
		if xsiErr != nil {
			contentErr = xsiErr
			continue
		}
		// A blocked xsi:type derivation is a validity error (cvc-elt.4.3), enforced
		// for a global element assessed through xs:anyType too.
		if td != declType && declType != nil && isDerivationBlocked(td, declType, edecl.Block) {
			vc.reportValidityError(ctx, vc.filename, ce.Line(), elemDisplayName(ce),
				"The xsi:type definition is blocked by the element declaration.")
			contentErr = fmt.Errorf("blocked xsi:type")
			continue
		}
		if td != nil && td.Abstract {
			msg := msgAbstractType
			vc.reportValidityError(ctx, vc.filename, ce.Line(), elemDisplayName(ce), msg)
			contentErr = fmt.Errorf("abstract type")
			continue
		}
		vc.annotateElement(ctx, ce, td, true)
		if td == nil {
			continue
		}
		nilled, nilErr := vc.checkXsiNil(ctx, ce)
		if nilErr != nil {
			contentErr = nilErr
			continue
		}
		if nilled {
			if err := vc.validateNilledElement(ctx, ce, edecl, td); err != nil {
				contentErr = err
			}
			continue
		}
		if err := vc.validateElementContent(ctx, ce, edecl, td); err != nil {
			contentErr = err
		}
	}
	return contentErr
}

// annotateSkipChildren walks the subtree of an element matched by an
// `xs:any processContents="skip"` wildcard purely to RECORD actual types for
// pass-2 IDC field canonicalization. Skipped content is NOT schema-assessed, so
// this MUST NOT impose any validation errors and MUST NOT run any content-model
// validation: it only records, for every descendant that carries a resolvable
// xsi:type, the ACTUAL type that override denotes (via annotateElement), then
// recurses. A nested global IDC host's fields would otherwise be canonicalized
// with declared (or raw) types, missing xsi:type overrides on descendants — even
// LOCAL descendants with no global declaration — under the skipped wrapper.
//
// The matched element ITSELF is annotated too: a PARENT IDC that selects this
// skip-wildcard-matched element directly must see its xsi:type ACTUAL type, so an
// xsi:type-introduced field (e.g. an inline xs:integer attribute) is canonicalized
// in the actual type's value space rather than compared lexically.
func (vc *validationContext) annotateSkipChildren(ctx context.Context, elem *helium.Element) {
	// Record this element as un-assessed skip content so a pass-2 IDC selector
	// does not pick it (an xs:key/xs:unique must not constrain unassessed nodes).
	if vc.skipContentNodes != nil {
		vc.skipContentNodes[elem] = struct{}{}
	}
	// Skipped content is NOT schema-assessed: annotate for pass-2 IDC
	// canonicalization only (assessed=false), so a skipped element carrying
	// xsi:type="xs:ID"/"xs:IDREF" is never picked up by the ID/IDREF pass.
	if actual, ok := vc.resolveXsiTypeQuiet(elem); ok {
		vc.annotateElement(ctx, elem, actual, false)
	}
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Resolve xsi:type WITHOUT reporting: skipped content is not assessed, so
		// an unresolvable or non-derived xsi:type must not raise a validity error.
		// Only an xsi:type override contributes an actual type distinct from what
		// pass-2 can already derive from the content model, so record only that
		// (assessed=false — skipped content is not schema-assessed).
		if actual, ok := vc.resolveXsiTypeQuiet(ce); ok {
			vc.annotateElement(ctx, ce, actual, false)
		}
		vc.annotateSkipChildren(ctx, ce)
	}
}

func (vc *validationContext) validateSimpleContent(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) error {
	// Simple content types must not have child elements.
	for child := range helium.Children(elem) {
		if child.Type() == helium.ElementNode {
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elem.LocalName(),
				"Element content is not allowed, because the content type is a simple type definition.")
			return fmt.Errorf("element content not allowed")
		}
	}

	value := elemTextContent(elem)
	isEmpty := value == ""

	// Effective value: substitute default/fixed for empty elements.
	effectiveValue := value
	if isEmpty && edecl != nil {
		if edecl.Fixed != nil {
			effectiveValue = *edecl.Fixed
		} else if edecl.Default != nil {
			effectiveValue = *edecl.Default
		}
	}

	// Record the effective default/fixed value of an EMPTY element so an xs:assert on
	// an ANCESTOR atomizes data(thisElement) as the schema-normalized default rather
	// than "" (isolatedAssertTree materializes it onto the copy). A QName/NOTATION
	// default's prefix resolves in the DECLARATION's namespace context (effectiveValueNS).
	if isEmpty && effectiveValue != "" && vc.assertEffectiveValues != nil {
		ns := effectiveValueNS(elem, edecl, true)
		contentTD := effectiveContentSimpleType(td)
		vc.assertEffectiveValues[elem] = assertEffectiveValue{
			value: effectiveValue,
			ns:    ns,
			td:    contentTD,
			qname: vc.valueHasQNameNotationCarrier(ctx, contentTD, effectiveValue, ns),
		}
	}

	// Fixed value mismatch check (only when element has actual content).
	// Compare in the *declared* type's value space (applying its whitespace
	// facet) rather than an unconditional TrimSpace, so value-equal lexical
	// variants are accepted and significant whitespace stays significant. The
	// fixed-value constraint is defined by the element declaration's own type,
	// not by an xsi:type actual type that may derive a different whiteSpace
	// facet — content is still validated against the actual td below, but the
	// fixed comparison must use edecl.Type so e.g. a declared xs:string
	// fixed="abc " keeps its trailing space even when xsi:type collapses.
	if !isEmpty && edecl != nil && edecl.Fixed != nil {
		fixedType := edecl.Type
		if fixedType == nil {
			fixedType = td
		}
		// In XSD 1.1 fixedValueMatches itself narrows a simpleContent type to its
		// effective content simple type, so the raw declared type is passed here.
		if !fixedValueMatches(ctx, value, *edecl.Fixed, fixedType, collectNSContext(elem), edecl.FixedNS, vc.schema, vc.version) {
			msg := fmt.Sprintf("The element content '%s' does not match the fixed value constraint '%s'.", value, *edecl.Fixed)
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
			return fmt.Errorf("fixed value constraint")
		}
	}

	// XSD 1.1: validate the text against the EFFECTIVE content simple type, composed
	// across the whole simpleContent derivation chain (effectiveContentSimpleType):
	// a base type's facets/assertions are enforced (e.g. an extension of a named
	// faceted type), and a narrowed content type is inherited through derived
	// simpleContent types (e.g. a restriction's enumeration survives a further
	// restriction or extension). A QName/NOTATION value substituted from the
	// declaration's fixed/default (empty element) resolves its prefix against the
	// DECLARATION's namespace context, not the instance's. XSD 1.0 keeps the original
	// gating and instance-context resolution, byte-identical.
	if vc.version == Version11 {
		valueNS := effectiveValueNS(elem, edecl, isEmpty)
		return vc.validateSimpleContentValue(ctx, effectiveValue, valueNS, td, elemDisplayName(elem), elem.Line())
	}

	// XSD 1.0: validate the text value against the type with the historical gating.
	if td != nil && (td.Facets != nil || resolveVariety(td) == TypeVarietyList || resolveVariety(td) == TypeVarietyUnion || builtinBaseLocal(td) != "" && builtinBaseLocal(td) != "string" && builtinBaseLocal(td) != lexicon.TypeAnySimpleType) {
		return validateValue(ctx, effectiveValue, collectNSContext(elem), td, elemDisplayName(elem), vc.filename, elem.Line(), vc)
	}

	return nil
}

// validateSimpleContentValue validates a value against a simpleContent (or plain
// simple) type's FULL effective constraint set: the composed effective content
// simple type (effectiveContentSimpleType) PLUS every inherited base content type
// reached through a nested <xs:simpleType> restriction (validateNestedSimpleContentBases).
// It is the single source of truth shared by the runtime simpleContent check
// (validateSimpleContent) and the compile-time element default/fixed check
// (checkElementDeclConstraints), so a default validated at compile time enforces
// exactly the same constraints the instance value does. displayName/line are used
// only for diagnostics (the schema error is emitted by the caller, so a suppressed
// validationContext drops the inner reports).
func (vc *validationContext) validateSimpleContentValue(ctx context.Context, value string, ns map[string]string, td *TypeDef, displayName string, line int) error {
	effTD := effectiveContentSimpleType(td)
	if simpleContentNeedsValidation(effTD) {
		if err := validateValue(ctx, value, ns, effTD, displayName, vc.filename, line, vc); err != nil {
			return err
		}
	}
	return vc.validateNestedSimpleContentBases(ctx, value, ns, td, effTD, displayName, line)
}

func (vc *validationContext) validateNestedSimpleContentBases(ctx context.Context, value string, ns map[string]string, td, effTD *TypeDef, displayName string, line int) error {
	visited := make(map[*TypeDef]struct{})
	for cur := td; cur != nil && cur.IsSimpleContent; cur = cur.BaseType {
		if _, seen := visited[cur]; seen {
			return nil
		}
		visited[cur] = struct{}{}
		// A nested <xs:simpleType> restriction (or a nested type narrowed further by
		// sibling facets) restricts the base content type per XSD §3.4.2.2; it does
		// not REPLACE it. effectiveContentSimpleType returns such an inline type with
		// its own declared base chain, so any ancestor complex type's inherited content
		// facets would otherwise be bypassed after a further derivation hop. Validate
		// each such ancestor base content type as well so both sets apply. The
		// facet-only synthetic case (ContentSimpleType.BaseType == cur) is already
		// re-based onto the effective base content by effectiveContentSimpleType, so
		// it is excluded here to avoid redundant work.
		if cur.ContentSimpleType == nil || cur.ContentSimpleType.BaseType == cur {
			continue
		}
		baseContent := effectiveContentSimpleType(cur.BaseType)
		if baseContent == effTD || !simpleContentNeedsValidation(baseContent) {
			continue
		}
		if err := validateValue(ctx, value, ns, baseContent, displayName, vc.filename, line, vc); err != nil {
			return err
		}
	}
	return nil
}

// effectiveValueNS returns the namespace context for resolving a QName/NOTATION
// in a simpleContent element's effective value. A non-empty element's content is
// the instance's own text, resolved against the instance's in-scope namespaces.
// An EMPTY element's value is substituted from the declaration's fixed/default,
// which was authored in the schema, so its prefixes resolve against the
// DECLARATION's namespace context (FixedNS/DefaultNS).
func effectiveValueNS(elem *helium.Element, edecl *ElementDecl, isEmpty bool) map[string]string {
	if isEmpty && edecl != nil {
		if edecl.Fixed != nil && edecl.FixedNS != nil {
			return edecl.FixedNS
		}
		if edecl.Default != nil && edecl.DefaultNS != nil {
			return edecl.DefaultNS
		}
	}
	return collectNSContext(elem)
}

// effectiveContentSimpleType returns the simple type that constrains the text
// content of a simpleContent complex type, composed across the WHOLE simpleContent
// derivation chain. It recurses through simpleContent complex types (IsSimpleContent)
// and stops at the underlying simpleType/builtin, which IS its own content type:
//
//   - extension / restriction with no own narrowing: content = base content;
//   - restriction with a nested <xs:simpleType>: that simpleType (carries its base
//     and facets);
//   - restriction with direct facets (a synthetic type whose BaseType is the owning
//     complex type): a fresh restriction of the base's EFFECTIVE content type with
//     those facets, so an ancestor's facets compose with the derived ones.
//
// A visited set guards against a cyclic base chain (an invalid schema reported by
// the circular-type check) without bounding the depth, so EVERY finite acyclic
// chain — however deep — is walked fully and a narrowing facet far down the chain
// is never silently skipped.
func effectiveContentSimpleType(td *TypeDef) *TypeDef {
	return effectiveContentSimpleTypeRec(td, make(map[*TypeDef]struct{}))
}

func effectiveContentSimpleTypeRec(td *TypeDef, visited map[*TypeDef]struct{}) *TypeDef {
	if td == nil || !td.IsSimpleContent {
		return td
	}
	if _, seen := visited[td]; seen {
		return td // cyclic base chain; the circular-type check reports the error
	}
	visited[td] = struct{}{}
	base := effectiveContentSimpleTypeRec(td.BaseType, visited)
	cst := td.ContentSimpleType
	if cst == nil {
		return base
	}
	if cst.BaseType == td {
		// Synthetic facet-only restriction: re-base on the effective base content
		// type so the ancestor's facets are still applied alongside these.
		return &TypeDef{
			ContentType: ContentTypeSimple,
			Derivation:  DerivationRestriction,
			Facets:      cst.Facets,
			BaseType:    base,
		}
	}
	// Nested <xs:simpleType>: it already carries its own base and facets.
	return cst
}

// simpleContentNeedsValidation reports whether the effective content simple type
// constrains its value (so validateValue must run): a list/union variety, a
// non-string builtin base, or any facet/assertion anywhere along its base chain.
func simpleContentNeedsValidation(td *TypeDef) bool {
	if td == nil {
		return false
	}
	switch resolveVariety(td) {
	case TypeVarietyList, TypeVarietyUnion:
		return true
	}
	if bl := builtinBaseLocal(td); bl != "" && bl != "string" && bl != lexicon.TypeAnySimpleType {
		return true
	}
	for cur := range baseChain(td) {
		if cur.Facets != nil {
			return true
		}
	}
	return false
}

// validateEmptyContent validates an element whose content type permits no child
// elements. When strict is true (an XSD 1.1 "empty" content type, e.g. a
// genuinely-empty <xs:sequence/> with no effective open content), ALL character
// content is rejected including whitespace (cvc-complex-type.2.1). When strict
// is false (a simple/empty content type, or the historical XSD 1.0 behavior),
// whitespace-only character content is tolerated.
func (vc *validationContext) validateEmptyContent(ctx context.Context, elem *helium.Element, strict bool) error {
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.ElementNode:
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			vc.reportValidityError(ctx, vc.filename, ce.Line(), ce.LocalName(), "This element is not expected.")
			return fmt.Errorf("not expected")
		case helium.TextNode, helium.CDATASectionNode:
			if strict {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elem.LocalName(), "Character content is not allowed, because the content type is empty.")
				return fmt.Errorf("not expected")
			}
			if !xmlchar.IsAllSpace(child.Content()) {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elem.LocalName(), "Character content is not allowed, because the type definition is simple.")
				return fmt.Errorf("not expected")
			}
		}
	}
	return nil
}

func (vc *validationContext) validateContentModel(ctx context.Context, elem *helium.Element, mg *ModelGroup) error {
	children := collectChildElements(elem)
	return vc.validateContentModelTop(ctx, elem, mg, children)
}

type childElem struct {
	elem        *helium.Element
	name        string // local name (for matching)
	ns          string // namespace URI (for matching)
	displayName string // namespace-qualified name (for error messages)
}

func collectChildElements(elem *helium.Element) []childElem {
	var children []childElem
	for child := range helium.Children(elem) {
		if child.Type() == helium.ElementNode {
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			children = append(children, childElem{elem: ce, name: ce.LocalName(), ns: ce.URI(), displayName: elemDisplayName(ce)})
		}
	}
	return children
}

// isSpecialAttr reports whether an attribute is always permitted regardless of
// the type's attribute declarations. In XSD 1.1 the XML-namespace
// attributes (xml:lang/space/base/id) are NOT implicitly allowed: they are
// subject to ordinary attribute-use and wildcard matching, so a wildcard's
// @notQName can legitimately exclude e.g. xml:space. Only xmlns and the xsi:
// processor attributes remain unconditionally special. In 1.0 the historical
// lenient behavior (XML namespace always allowed) is preserved.
func (vc *validationContext) isSpecialAttr(a *helium.Attribute) bool {
	p := a.Prefix()
	if p == "xmlns" || (p == "" && a.LocalName() == "xmlns") {
		return true
	}
	if a.URI() == lexicon.NamespaceXSI {
		return true
	}
	if vc.version != Version11 && a.URI() == lexicon.NamespaceXML {
		return true
	}
	return false
}

func elemDisplayName(elem *helium.Element) string {
	if elem.URI() != "" {
		return helium.ClarkName(elem.URI(), elem.LocalName())
	}
	return elem.LocalName()
}

func attrDisplayName(a *helium.Attribute) string {
	uri := a.URI()
	if uri != "" {
		return helium.ClarkName(uri, a.LocalName())
	}
	return a.LocalName()
}

// isKnownXsiProcessorAttr reports whether local is one of the FOUR processor
// attributes that actually exist in the XSI namespace (xsi:type, xsi:nil,
// xsi:schemaLocation, xsi:noNamespaceSchemaLocation). The XSI namespace is
// reserved to exactly these; any other xsi:-namespace local name (e.g.
// xsi:foo) is not a real attribute and must NOT receive the declared-xsi
// special handling.
func isKnownXsiProcessorAttr(local string) bool {
	switch local {
	case attrType, attrNil, attrSchemaLocation, attrNoNSSchemaLocation:
		return true
	}
	return false
}

// xsiProcessorAttrBuiltinType returns the built-in SCALAR type a declared xsi:
// processor-attribute reference is associated with, so its fixed/default value
// is validated against (and compared in) that type's value space (xsi:type→
// xs:QName, xsi:nil→xs:boolean, xsi:noNamespaceSchemaLocation→xs:anyURI). It
// returns ("", false) for xsi:schemaLocation, whose type is a LIST of xs:anyURI
// with no scalar built-in equivalent — its value/even-pair validity is handled
// directly by validateDeclaredXsiAttrValue instead.
func xsiProcessorAttrBuiltinType(local string) (QName, bool) {
	switch local {
	case attrType:
		return QName{Local: lexicon.TypeQName, NS: lexicon.NamespaceXSD}, true
	case attrNil:
		return QName{Local: lexicon.TypeBoolean, NS: lexicon.NamespaceXSD}, true
	case attrNoNSSchemaLocation:
		return QName{Local: lexicon.TypeAnyURI, NS: lexicon.NamespaceXSD}, true
	}
	return QName{}, false
}

// xsiSchemaLocationTokens collapses a value in XSD whitespace and splits it on
// XSD list whitespace ONLY (space/tab/CR/LF via value.XSDFields — NBSP and
// other Unicode whitespace are NOT separators), yielding the
// xsi:schemaLocation token list (its value space is a list of xs:anyURI).
func xsiSchemaLocationTokens(val string) []string {
	return value.XSDFields(normalizeWhiteSpace(val, "collapse"))
}

// validateXsiSchemaLocationValue validates a value against xsi:schemaLocation's
// value space: a NON-empty, EVEN-length list of (namespace, location)
// xs:anyURI pairs. An odd count, an empty value, or a non-anyURI token is an
// error. Used for both a present instance value and a fixed/default LITERAL.
func validateXsiSchemaLocationValue(val string, version Version) error {
	toks := xsiSchemaLocationTokens(val)
	if len(toks) == 0 || len(toks)%2 != 0 {
		return fmt.Errorf("xsi:schemaLocation must be a non-empty list of anyURI pairs")
	}
	for _, t := range toks {
		if err := validateBuiltinValue(t, lexicon.TypeAnyURI, version); err != nil {
			return err
		}
	}
	return nil
}

// xsiSchemaLocationValueEqual reports whether two xsi:schemaLocation values are
// equal in VALUE space — equal token lists after XSD whitespace collapse and
// XSD-whitespace tokenization — so leading/trailing/internal run-length
// whitespace differences (e.g. " urn:a   loc.xsd " vs "urn:a loc.xsd") do not
// matter, while raw string equality would false-reject.
func xsiSchemaLocationValueEqual(a, b string) bool {
	return slices.Equal(xsiSchemaLocationTokens(a), xsiSchemaLocationTokens(b))
}

// isDeclaredXsiSchemaLocationUse reports whether au is a declared xsi:schemaLocation
// attribute use (the only one of the four xsi: processor attributes whose
// value space is a list with no scalar built-in type, so its fixed/default is
// handled by the list helpers above rather than a built-in TypeDef).
func isDeclaredXsiSchemaLocationUse(au *AttrUse, version Version) bool {
	return version == Version11 && au.Name.NS == lexicon.NamespaceXSI && au.Name.Local == attrSchemaLocation
}

// validateDeclaredXsiAttrValue validates a PRESENT xsi: processor attribute
// whose use is EXPLICITLY declared (XSD 1.1 `ref="xsi:*"`) against the fixed
// built-in type that attribute is defined with. `ref="xsi:type"` does not
// resolve to a typed global attribute use, so the ordinary attribute-use value
// check sees no type; this supplies each xsi attribute's built-in type so an
// empty/malformed value — xsi:type="" or a malformed/unbound QName, a
// non-boolean xsi:nil, a non-anyURI location — is a validation error rather
// than a silently-satisfied (present) use. Only the four real xsi: processor
// attributes are recognized (the caller gates on isKnownXsiProcessorAttr).
// Returns nil when the value is valid.
func (vc *validationContext) validateDeclaredXsiAttrValue(a *helium.Attribute, elem *helium.Element) error {
	val := a.Value()
	switch a.LocalName() {
	case attrType:
		// xs:QName (whiteSpace=collapse): lexically valid and, when prefixed, the
		// prefix must be bound in scope (a non-empty, namespace-resolvable QName).
		q := normalizeWhiteSpace(val, "collapse")
		if err := validateQName(q); err != nil {
			return err
		}
		if prefix, _, ok := strings.Cut(q, ":"); ok && lookupNS(elem, prefix) == "" {
			return fmt.Errorf("unbound xsi:type QName prefix %q", prefix)
		}
		return nil
	case attrNil:
		return validateBuiltinValue(normalizeWhiteSpace(val, "collapse"), lexicon.TypeBoolean, vc.version)
	case attrSchemaLocation:
		return validateXsiSchemaLocationValue(val, vc.version)
	case attrNoNSSchemaLocation:
		return validateBuiltinValue(normalizeWhiteSpace(val, "collapse"), lexicon.TypeAnyURI, vc.version)
	}
	return nil
}

func (vc *validationContext) validateAttributes(ctx context.Context, elem *helium.Element, td *TypeDef) error {
	var hasErr bool

	if len(td.Attributes) == 0 && td.AnyAttribute == nil {
		// No attribute declarations — check that instance has no attributes
		// (except xsi: namespace attributes and xmlns which are always allowed).
		for _, a := range elem.Attributes() {
			if vc.isSpecialAttr(a) {
				continue
			}
			ad := attrDisplayName(a)
			msg := fmt.Sprintf("The attribute '%s' is not allowed.", ad)
			vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
			hasErr = true
		}
		if hasErr {
			return fmt.Errorf("attribute not allowed")
		}
		return nil
	}

	// Build set of allowed attributes. A prohibited attribute use does not
	// contribute an allowed attribute; instead its QName is recorded so an
	// instance attribute carrying it is rejected before wildcard matching. A
	// non-prohibited use of the same QName always wins (it removes any
	// prohibition recorded for that QName).
	allowed := make(map[QName]*AttrUse, len(td.Attributes))
	var prohibited map[QName]struct{}
	for _, au := range td.Attributes {
		if au.Prohibited {
			if _, ok := allowed[au.Name]; ok {
				continue
			}
			if prohibited == nil {
				prohibited = make(map[QName]struct{})
			}
			prohibited[au.Name] = struct{}{}
			continue
		}
		allowed[au.Name] = au
		delete(prohibited, au.Name)
	}

	// Build set of present instance attributes (excluding special attrs)
	// for O(1) lookups in the required-check and default-insertion loops.
	present := make(map[QName]struct{}, len(elem.Attributes()))

	// Check for unknown attributes and fixed value constraints.
	for _, a := range elem.Attributes() {
		aqn := QName{Local: a.LocalName(), NS: a.URI()}
		// True for a present, non-prohibited DECLARED xsi: processor-attribute use
		// whose value was already validated by validateDeclaredXsiAttrValue below —
		// so the generic type-based value check is skipped (no double validation);
		// the fixed-value comparison still runs (against the built-in type).
		declaredXsiValueChecked := false
		if vc.isSpecialAttr(a) {
			// XSD 1.1: the xsi: processor attributes (xsi:type, xsi:nil,
			// xsi:schemaLocation, xsi:noNamespaceSchemaLocation) may be
			// referenced explicitly as attribute uses (e.g. to make xsi:type
			// mandatory, or to PROHIBIT it). When a type's declaration explicitly
			// declares such an xsi: attribute — with ANY use (optional/required/
			// prohibited) — the instance attribute participates in ordinary
			// attribute-use validation (so a required xsi: use is satisfied and a
			// prohibited one rejects a present xsi: attribute); otherwise xsi:
			// (and xmlns) attributes remain unconditionally special. A prohibited
			// use is excluded from `allowed`, so the declaration is detected via
			// both maps. 1.0 keeps the historical skip (byte-identical).
			_, allowedXSI := allowed[aqn]
			_, prohibitedXSI := prohibited[aqn]
			declaredXSI := allowedXSI || prohibitedXSI
			// Only the four real xsi: processor attributes participate; a declared
			// ref to any other xsi: local name (e.g. xsi:foo) is not specially
			// accepted — it stays skipped as special, so a required use of it is
			// never satisfied (the instance is rejected as missing).
			if vc.version != Version11 || a.URI() != lexicon.NamespaceXSI || !declaredXSI || !isKnownXsiProcessorAttr(a.LocalName()) {
				continue
			}
			// A NON-prohibited declared xsi: use must validate its value against the
			// attribute's built-in type (ref="xsi:*" carries no resolvable type). An
			// invalid value (e.g. xsi:type="") is a validity error and does NOT
			// satisfy the use — it is not recorded as present, so a required use also
			// reports missing. A prohibited use needs no value check: its mere
			// presence is rejected below.
			if allowedXSI {
				if err := vc.validateDeclaredXsiAttrValue(a, elem); err != nil {
					ad := attrDisplayName(a)
					msg := fmt.Sprintf("The value '%s' is not valid for the type of attribute '%s'.", a.Value(), ad)
					vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
					hasErr = true
					continue
				}
				declaredXsiValueChecked = true
			}
		}
		present[aqn] = struct{}{}
		if au, ok := allowed[aqn]; ok {
			// Resolve the declared type up front so the fixed-value check can
			// compare in the type's value space (applying its whitespace
			// facet) rather than by raw string equality.
			attrTD, tdOK := vc.attrUseType(au)
			if au.Fixed != nil {
				// xsi:schemaLocation has no scalar built-in type, so its fixed value
				// is compared in VALUE space as a list of xs:anyURI (collapse + XSD
				// tokenize + token-list equality); every other use compares via
				// fixedValueMatches against its (built-in or declared) type.
				fixedMatches := false
				switch {
				case isDeclaredXsiSchemaLocationUse(au, vc.version):
					fixedMatches = xsiSchemaLocationValueEqual(a.Value(), *au.Fixed)
				default:
					fixedMatches = fixedValueMatches(ctx, a.Value(), *au.Fixed, attrTD, collectNSContext(elem), au.FixedNS, vc.schema, vc.version)
				}
				if !fixedMatches {
					ad := attrDisplayName(a)
					msg := fmt.Sprintf("The value '%s' does not match the fixed value constraint '%s'.", a.Value(), *au.Fixed)
					vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
					hasErr = true
				}
			}
			// Validate the attribute value against its declared type
			// (inline anonymous simpleType takes precedence over a named type).
			// A declared xsi: use was already value-validated above (its built-in
			// type is associated only for the fixed-value comparison just done), so
			// skip the generic check to avoid validating the same value twice.
			if tdOK && attrTD.ContentType == ContentTypeSimple && !declaredXsiValueChecked {
				if err := validateValue(ctx, a.Value(), collectNSContext(elem), attrTD, elemDisplayName(elem), vc.filename, elem.Line(), &validationContext{schema: vc.schema, version: vc.version, errorHandler: helium.NilErrorHandler{}}); err != nil {
					ad := attrDisplayName(a)
					msg := fmt.Sprintf("The value '%s' is not valid for the type of attribute '%s'.", a.Value(), ad)
					vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
					hasErr = true
				}
			}
			// Annotate the attribute with its declared type.
			vc.annotateAttrUse(ctx, a, au)
			// XSD 1.1: record an inheritable attribute so descendants' conditional
			// type assignment / assertions can see it as an inherited attribute.
			if vc.version == Version11 && au.Inheritable {
				vc.attrInheritable[a] = struct{}{}
			}
			continue
		}
		// An explicitly prohibited attribute use is rejected outright and must
		// not be allowed in by an attribute wildcard.
		if _, prohib := prohibited[aqn]; prohib {
			ad := attrDisplayName(a)
			msg := fmt.Sprintf("The attribute '%s' is not allowed.", ad)
			vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
			hasErr = true
			continue
		}
		// Not in explicit declarations — check anyAttribute wildcard.
		if td.AnyAttribute != nil && wildcardAllowsExpandedName(td.AnyAttribute, a.LocalName(), a.URI(), vc.schema, true) {
			if err := vc.validateWildcardAttr(ctx, a, elem, td.AnyAttribute); err != nil {
				hasErr = true
			}
			continue
		}
		ad := attrDisplayName(a)
		msg := fmt.Sprintf("The attribute '%s' is not allowed.", ad)
		vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
		hasErr = true
	}

	// Check for required attributes.
	for _, au := range td.Attributes {
		if !au.Required {
			continue
		}
		if _, ok := present[au.Name]; !ok {
			msg := fmt.Sprintf("The attribute '%s' is required but missing.", au.Name.Local)
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
			hasErr = true
		}
	}

	// Insert default/fixed attribute values for absent optional attributes.
	for _, au := range td.Attributes {
		if au.Required {
			continue
		}
		// A prohibited attribute use must never materialize a default/fixed
		// value: the absent attribute is accepted as-is, and supplying one
		// would itself be rejected.
		if au.Prohibited {
			continue
		}
		defVal := ""
		if au.Default != nil {
			defVal = *au.Default
		} else if au.Fixed != nil {
			defVal = *au.Fixed
		} else {
			continue
		}
		if _, ok := present[au.Name]; ok {
			continue
		}
		// XSD 1.1: a QName/NOTATION default/fixed value was authored in the schema, so
		// its lexical prefix denotes the schema's namespace. Once materialized on the
		// instance, an xs:assert / IDC that atomizes the attribute resolves the
		// prefix against the INSTANCE's in-scope namespaces — so ensure that prefix
		// is bound to the declaration's URI on the element (rewriting to a fresh
		// prefix if the instance already binds it to a different URI). Gated to 1.1
		// so XSD 1.0 inserts the value exactly as authored (byte-identical
		// serialization — no namespace-declaration rewrite).
		if vc.version == Version11 {
			declNS := au.FixedNS
			if au.Default != nil {
				declNS = au.DefaultNS
			}
			defVal = vc.materializeQNameAttrValue(ctx, elem, au, defVal, declNS)
		}
		// Insert the default/fixed value as an attribute on the element. A
		// qualified attribute (non-empty NS, e.g. under attributeFormDefault=
		// "qualified") must be inserted with its namespace so later consumers
		// such as an xs:key field "@t:a" can match it.
		if au.Name.NS != "" {
			ns := inScopeNamespace(elem, au.Name.NS)
			if ns == nil {
				ns = helium.NewNamespace("", au.Name.NS)
			}
			_, _ = elem.SetAttributeNS(au.Name.Local, defVal, ns)
		} else {
			_, _ = elem.SetAttribute(au.Name.Local, defVal)
		}
		// Annotate the newly inserted attribute and, for XSD 1.1, record it as
		// inheritable when its use is — a defaulted/fixed attribute is part of the
		// inherited-attribute set just like an explicitly-present one.
		for _, a := range elem.Attributes() {
			if a.LocalName() == au.Name.Local && a.URI() == au.Name.NS {
				vc.annotateAttrUse(ctx, a, au)
				if vc.version == Version11 && au.Inheritable {
					vc.attrInheritable[a] = struct{}{}
				}
				break
			}
		}
	}

	if hasErr {
		return fmt.Errorf("attribute validation failed")
	}
	return nil
}

// materializeQNameAttrValue prepares a default/fixed attribute value for insertion
// so QName/NOTATION lexical values keep their SCHEMA-intended namespace once on the
// instance. The value is authored with the declaration's prefix bindings (declNS);
// an instance consumer (xs:assert, IDC) resolves prefixes against the element's
// in-scope namespaces instead, so this binds each QName/NOTATION token's prefix to
// the declaration URI on the element. If the instance already binds that prefix to
// a DIFFERENT URI, the token is rewritten to a fresh, non-colliding prefix. The
// type walk is value-dependent for unions and recursive through list item types.
func (vc *validationContext) materializeQNameAttrValue(ctx context.Context, elem *helium.Element, au *AttrUse, value string, declNS map[string]string) string {
	if declNS == nil {
		return value
	}
	td, ok := vc.attrUseType(au)
	if !ok {
		return value
	}
	materialized, _ := vc.materializeQNameValue(ctx, elem, td, value, declNS)
	return materialized
}

func (vc *validationContext) materializeQNameValue(ctx context.Context, elem *helium.Element, td *TypeDef, value string, declNS map[string]string) (string, bool) {
	if td == nil || declNS == nil {
		return value, false
	}
	return vc.materializeQNameValueVisit(ctx, elem, td, value, declNS, map[*TypeDef]struct{}{}, make(map[string]string))
}

func (vc *validationContext) materializeQNameValueVisit(ctx context.Context, elem *helium.Element, td *TypeDef, raw string, declNS map[string]string, seen map[*TypeDef]struct{}, rewrites map[string]string) (string, bool) {
	if td == nil {
		return raw, false
	}
	if _, ok := seen[td]; ok {
		return raw, false
	}
	seen[td] = struct{}{}
	defer delete(seen, td)

	switch resolveVariety(td) {
	case TypeVarietyUnion:
		collapsed := normalizeWhiteSpace(raw, "collapse")
		activeTD := fixedUnionActiveMember(ctx, collapsed, declNS, resolveUnionMembers(td), vc.schema, vc.version)
		if activeTD == nil {
			return raw, false
		}
		return vc.materializeQNameValueVisit(ctx, elem, activeTD, raw, declNS, seen, rewrites)
	case TypeVarietyList:
		itemType := resolveItemType(td)
		if itemType == nil {
			return raw, false
		}
		collapsed := normalizeWhiteSpace(raw, "collapse")
		tokens := value.XSDFields(collapsed)
		if len(tokens) == 0 {
			return raw, false
		}
		out := make([]string, len(tokens))
		hasCarrier := false
		rewrote := false
		for i, token := range tokens {
			next, carrier := vc.materializeQNameValueVisit(ctx, elem, itemType, token, declNS, seen, rewrites)
			out[i] = next
			if carrier {
				hasCarrier = true
			}
			if next != token {
				rewrote = true
			}
		}
		if !hasCarrier {
			return raw, false
		}
		if rewrote {
			return strings.Join(out, " "), true
		}
		return raw, true
	default:
		switch builtinBaseLocal(td) {
		case lexicon.TypeQName, lexicon.TypeNotation:
			collapsed := normalizeWhiteSpace(raw, resolveWhiteSpace(td))
			if rewritten, ok := materializeQNameToken(elem, collapsed, declNS, rewrites); ok {
				return rewritten, true
			}
			return raw, true
		default:
			return raw, false
		}
	}
}

func (vc *validationContext) valueHasQNameNotationCarrier(ctx context.Context, td *TypeDef, value string, declNS map[string]string) bool {
	return vc.valueHasQNameNotationCarrierVisit(ctx, td, value, declNS, map[*TypeDef]struct{}{})
}

func (vc *validationContext) valueHasQNameNotationCarrierVisit(ctx context.Context, td *TypeDef, raw string, declNS map[string]string, seen map[*TypeDef]struct{}) bool {
	if td == nil {
		return false
	}
	if _, ok := seen[td]; ok {
		return false
	}
	seen[td] = struct{}{}
	defer delete(seen, td)

	switch resolveVariety(td) {
	case TypeVarietyUnion:
		collapsed := normalizeWhiteSpace(raw, "collapse")
		activeTD := fixedUnionActiveMember(ctx, collapsed, declNS, resolveUnionMembers(td), vc.schema, vc.version)
		return vc.valueHasQNameNotationCarrierVisit(ctx, activeTD, collapsed, declNS, seen)
	case TypeVarietyList:
		itemType := resolveItemType(td)
		if itemType == nil {
			return false
		}
		collapsed := normalizeWhiteSpace(raw, "collapse")
		for _, token := range value.XSDFields(collapsed) {
			if vc.valueHasQNameNotationCarrierVisit(ctx, itemType, token, declNS, seen) {
				return true
			}
		}
		return false
	default:
		switch builtinBaseLocal(td) {
		case lexicon.TypeQName, lexicon.TypeNotation:
			return true
		default:
			return false
		}
	}
}

func materializeQNameToken(elem *helium.Element, token string, declNS map[string]string, rewrites map[string]string) (string, bool) {
	prefix, local, found := strings.Cut(token, ":")
	if !found || prefix == "" {
		return token, false // no prefix → no-namespace QName; nothing to bind
	}
	uri, ok := declNS[prefix]
	if !ok || uri == "" {
		return token, false // prefix not bound in the declaration; leave as authored
	}
	if elem == nil {
		return token, false
	}
	inScope := collectNSContext(elem)
	cur, bound := inScope[prefix]
	if bound && cur == uri {
		return token, false // already resolves to the intended URI
	}
	if !bound {
		elem.AddNamespaceDecl(helium.NewNamespace(prefix, uri))
		return token, false
	}
	key := prefix + "\x00" + uri
	if np, ok := rewrites[key]; ok {
		return np + ":" + local, true
	}
	np := freshNSPrefix(inScope, prefix)
	rewrites[key] = np
	elem.AddNamespaceDecl(helium.NewNamespace(np, uri))
	return np + ":" + local, true
}

// freshNSPrefix returns a prefix not present in inScope, derived from base.
func freshNSPrefix(inScope map[string]string, base string) string {
	for i := 0; ; i++ {
		candidate := fmt.Sprintf("%s_gen%d", base, i)
		if _, taken := inScope[candidate]; !taken {
			return candidate
		}
	}
}

// validateWildcardAttr validates an attribute matched by a wildcard according
// to its processContents setting (strict, lax, or skip).
func (vc *validationContext) validateWildcardAttr(ctx context.Context, a *helium.Attribute, elem *helium.Element, wc *Wildcard) error {
	if wc.ProcessContents == ProcessSkip {
		return nil
	}

	// Look up global attribute declaration.
	aqn := QName{Local: a.LocalName(), NS: a.URI()}
	globalAttr, found := vc.schema.globalAttrs[aqn]

	if !found {
		if wc.ProcessContents == ProcessStrict {
			ad := attrDisplayName(a)
			msg := "No matching global attribute declaration available, but demanded by the strict wildcard."
			vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
			return fmt.Errorf("strict wildcard: no global attr")
		}
		// Lax: no global declaration found — skip validation.
		return nil
	}

	// Global attribute found — validate value against its effective type if
	// known (an inline anonymous simpleType takes precedence over a named type).
	// TypeDef.Validate handles facets, lists, and unions, not just the builtin
	// base lexical space.
	attrTD, ok := vc.attrUseType(globalAttr)

	// This wildcard-admitted attribute IS schema-assessed (strict/lax with a
	// matching global declaration — skip already returned above), so record its
	// type for the XSD 1.1 ID/IDREF pass.
	if ok && vc.actualAttrType != nil {
		vc.actualAttrType[a] = attrTD
	}

	// Enforce the global attribute's fixed-value constraint. A wildcard-matched
	// global fixed attribute must still satisfy its fixed value, in the declared
	// type's value space (mirroring the non-wildcard attribute path).
	if globalAttr.Fixed != nil && !fixedValueMatches(ctx, a.Value(), *globalAttr.Fixed, attrTD, collectNSContext(elem), globalAttr.FixedNS, vc.schema, vc.version) {
		ad := attrDisplayName(a)
		msg := fmt.Sprintf("The value '%s' does not match the fixed value constraint '%s'.", a.Value(), *globalAttr.Fixed)
		vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
		return fmt.Errorf("fixed value constraint")
	}

	if ok && attrTD.ContentType == ContentTypeSimple {
		value := a.Value()
		if err := validateValue(ctx, value, collectNSContext(elem), attrTD, elemDisplayName(elem), vc.filename, elem.Line(), &validationContext{schema: vc.schema, version: vc.version, errorHandler: helium.NilErrorHandler{}}); err != nil {
			ad := attrDisplayName(a)
			typeName := typeDisplayName(attrTD)
			msg := fmt.Sprintf("'%s' is not a valid value of the atomic type '%s'.", strings.TrimSpace(value), typeName)
			vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
			return err
		}
	}

	// A wildcard-matched global attribute participates in type annotation and (XSD
	// 1.1) inheritance exactly like an explicitly-declared attribute use, so a
	// descendant's conditional type assignment can see an inheritable ancestor
	// attribute admitted through xs:anyAttribute.
	vc.annotateAttrUse(ctx, a, globalAttr)
	if vc.version == Version11 && globalAttr.Inheritable {
		vc.attrInheritable[a] = struct{}{}
	}
	return nil
}

// lookupElemDecl finds the global element declaration for an instance element.
// Matching is on the element's full expanded name: a namespaced element does
// not fall back to an unqualified declaration sharing the local name.
func lookupElemDecl(elem *helium.Element, schema *Schema) *ElementDecl {
	edecl, ok := schema.LookupElement(elem.LocalName(), elem.URI())
	if ok {
		return edecl
	}
	return nil
}

// elemTextContent returns the concatenated text content of an element,
// including both text nodes and CDATA sections.
func elemTextContent(elem *helium.Element) string {
	var buf []byte
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			buf = append(buf, child.Content()...)
		}
	}
	return string(buf)
}

// checkXsiNil parses the element's xsi:nil attribute as an xs:boolean (after
// whitespace collapse). It returns whether the element is nilled ("true"/"1").
// "false"/"0" and an absent attribute mean not-nilled. Any other lexical form
// is an invalid xs:boolean value: a validity error is reported and a non-nil
// error is returned so the element is not silently validated as ordinary
// content.
func (vc *validationContext) checkXsiNil(ctx context.Context, elem *helium.Element) (bool, error) {
	for _, a := range elem.Attributes() {
		if a.URI() != lexicon.NamespaceXSI || a.LocalName() != attrNil {
			continue
		}
		v := normalizeWhiteSpace(a.Value(), "collapse")
		switch v {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		}
		msg := fmt.Sprintf("'%s' is not a valid value of the atomic type 'xs:boolean'.", v)
		vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), attrDisplayName(a), msg)
		return false, fmt.Errorf("invalid xsi:nil value %q", a.Value())
	}
	return false, nil
}

// validateNilledElement handles an element with xsi:nil="true".
// If the declaration is nillable, validates that the element has no character
// or element content (attributes are still checked).  If not nillable,
// reports a validity error.
func (vc *validationContext) validateNilledElement(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) error {
	dn := elemDisplayName(elem)

	// XSD 1.1: a governing type of xs:error has an empty value space, so the element
	// is invalid regardless of xsi:nil — the nilled path must NOT let it through.
	if vc.version == Version11 && isErrorType(td) {
		vc.reportValidityError(ctx, vc.filename, elem.Line(), dn,
			"The element is not valid: the conditional type assignment selected the type xs:error.")
		return fmt.Errorf("xs:error type selected")
	}

	if !edecl.Nillable {
		vc.reportValidityError(ctx, vc.filename, elem.Line(), dn,
			"Element is not nillable.")
		return fmt.Errorf("element not nillable")
	}

	// Record the element as nilled for PSVI consumers (e.g. fn:nilled()).
	if vc.cfg != nil && vc.cfg.nilledElements != nil {
		(*vc.cfg.nilledElements)[elem] = struct{}{}
	}

	// Validate attributes even for nilled elements.
	if td != nil {
		if err := vc.validateAttributes(ctx, elem, td); err != nil {
			return err
		}
	}

	// xsi:nil="true" — the element must have no character or element children.
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.ElementNode:
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			vc.reportValidityError(ctx, vc.filename, ce.Line(), elemDisplayName(ce),
				"This element is not expected, because the element '"+dn+"' is nilled.")
			return fmt.Errorf("content in nilled element")
		case helium.TextNode, helium.CDATASectionNode:
			// cvc-elt.3.2.1: a nilled element must have NO character content. XSD 1.0
			// tolerates insignificant whitespace (matching libxml2); XSD 1.1 rejects
			// any character content, including whitespace-only.
			if vc.version == Version11 || !xmlchar.IsAllSpace(child.Content()) {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), dn,
					"Character content is not allowed, because the element is nilled.")
				return fmt.Errorf("content in nilled element")
			}
		}
	}

	return nil
}

// isDerivedFrom returns true if derived is the same type as base, or if any
// ancestor in derived's BaseType chain is base. Also returns true if base is
// xs:anyType (the ur-type from which everything derives).
func isDerivedFrom(derived, base *TypeDef) bool {
	if derived == base {
		return true
	}
	if base.Name.Local == typeAnyType && base.Name.NS == lexicon.NamespaceXSD {
		return true
	}
	for cur := range baseChain(derived.BaseType) {
		if cur == base {
			return true
		}
	}
	return false
}

// resolveXsiType checks if the element has an xsi:type attribute and, if so,
// resolves it to a type definition in the schema. Returns the resolved type
// or the original declaredType if no xsi:type is present. Returns an error
// if the xsi:type value doesn't resolve or is not derived from the declared type.
// ctaActive must be true when conditional type assignment is in effect for elem
// (Version11 and the element declaration has a {type table}). It scopes the
// empty-xsi:type handling: only then does a present-but-empty xsi:type hard-error,
// so it cannot suppress a CTA-selected type (e.g. xs:error). Everywhere else an
// empty xsi:type falls back to the declared type, byte-identical to pre-CTA
// behavior (XSD 1.0 and no-alternative 1.1).
func (vc *validationContext) resolveXsiType(ctx context.Context, elem *helium.Element, declaredType *TypeDef, ctaActive bool) (*TypeDef, error) {
	var xsiTypeVal string
	var present bool
	for _, a := range elem.Attributes() {
		if a.URI() == lexicon.NamespaceXSI && a.LocalName() == attrType {
			xsiTypeVal = a.Value()
			present = true
			break
		}
	}
	// An ABSENT xsi:type always falls back to the declared type (under CTA this is
	// the already-selected type, so it must not error).
	if !present {
		return declaredType, nil
	}
	// A present-but-empty xsi:type historically also falls back to the declared
	// type. Preserve that EXCEPT when CTA is active, where it must instead report the
	// invalid-QName validity error so it cannot bypass a CTA-selected type (e.g.
	// xs:error). A non-empty value always proceeds to QName resolution below.
	if xsiTypeVal == "" && !ctaActive {
		return declaredType, nil
	}

	// xsi:type is an xs:QName, whose whiteSpace facet is "collapse": normalize
	// the raw attribute value before parsing so leading/trailing/internal
	// whitespace (e.g. " t:foo ") does not defeat QName resolution.
	xsiTypeVal = normalizeWhiteSpace(xsiTypeVal, "collapse")
	if err := validateQName(xsiTypeVal); err != nil {
		msg := fmt.Sprintf("The value '%s' of the xsi:type attribute does not resolve to a type definition.", xsiTypeVal)
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return nil, fmt.Errorf("xsi:type not a valid QName")
	}

	// Parse QName value: may be "prefix:local" or just "local".
	local := xsiTypeVal
	var ns string
	if prefix, rest, ok := strings.Cut(xsiTypeVal, ":"); ok {
		local = rest
		ns = lookupNS(elem, prefix)
	} else {
		// No prefix — use the default namespace (empty prefix) or schema target namespace.
		ns = lookupNS(elem, "")
	}

	td, ok := vc.schema.LookupType(local, ns)
	if !ok {
		// Try with schema's target namespace.
		td, ok = vc.schema.LookupType(local, vc.schema.TargetNamespace())
	}
	if !ok {
		msg := fmt.Sprintf("The value '%s' of the xsi:type attribute does not resolve to a type definition.", xsiTypeVal)
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return nil, fmt.Errorf("xsi:type not found")
	}

	// Check derivation: xsi:type must be the same as or validly substitutable for
	// the declared type. Use the built-in-aware predicate plus the union-member
	// rule so a narrowing to a built-in subtype (e.g. xsi:type="xs:int" over a
	// declared xs:integer) and a member of a declared union are accepted. The
	// predicate is STRICT (no permissive simple-vs-simple fallback), so an unrelated
	// xsi:type (e.g. xs:string over xs:integer) is still rejected.
	if declaredType != nil && !isXsiTypeDerivedFromDeclared(td, declaredType) {
		msg := fmt.Sprintf("The type definition '%s' is not validly derived from the type definition '%s'.",
			typeDisplayName(td), typeDisplayName(declaredType))
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return nil, fmt.Errorf("xsi:type not derived")
	}

	return td, nil
}

// resolveXsiTypeQuiet resolves an element's xsi:type to a schema type WITHOUT
// reporting any validity error. It is used for skipped (`processContents="skip"`)
// content, which is not schema-assessed: a missing or non-derived xsi:type must
// not raise an error, it just means no actual type override is available. Returns
// (type, true) only when the xsi:type value resolves to a known type.
func (vc *validationContext) resolveXsiTypeQuiet(elem *helium.Element) (*TypeDef, bool) {
	var xsiTypeVal string
	for _, a := range elem.Attributes() {
		if a.URI() == lexicon.NamespaceXSI && a.LocalName() == attrType {
			xsiTypeVal = a.Value()
			break
		}
	}
	if xsiTypeVal == "" {
		return nil, false
	}

	// xsi:type is an xs:QName (whiteSpace=collapse): normalize before parsing so
	// surrounding whitespace does not defeat resolution. Errors are not reported
	// here (skipped content is not assessed); an invalid QName simply yields no
	// actual type override.
	xsiTypeVal = normalizeWhiteSpace(xsiTypeVal, "collapse")
	if err := validateQName(xsiTypeVal); err != nil {
		return nil, false
	}

	local := xsiTypeVal
	var ns string
	if prefix, rest, ok := strings.Cut(xsiTypeVal, ":"); ok {
		local = rest
		ns = lookupNS(elem, prefix)
	} else {
		ns = lookupNS(elem, "")
	}

	td, ok := vc.schema.LookupType(local, ns)
	if !ok {
		td, ok = vc.schema.LookupType(local, vc.schema.TargetNamespace())
	}
	if !ok {
		return nil, false
	}
	return td, true
}

// xsdTypeName converts a TypeDef to a type name string suitable for annotations.
// For anonymous types (no name), it walks up the base type chain to find the
// nearest named ancestor type, since XPath type checks need a concrete type name.
func xsdTypeName(td *TypeDef) string {
	if td == nil {
		return "xs:untyped"
	}
	if td.Name.NS == lexicon.NamespaceXSD {
		return "xs:" + td.Name.Local
	}
	if td.Name.NS != "" {
		return "Q{" + td.Name.NS + "}" + td.Name.Local
	}
	if td.Name.Local != "" {
		return "Q{}" + td.Name.Local
	}
	// Anonymous type: walk up the base type chain to find a named type.
	for cur := range baseChain(td.BaseType) {
		if cur.Name.NS == lexicon.NamespaceXSD {
			return "xs:" + cur.Name.Local
		}
		if cur.Name.NS != "" {
			return "Q{" + cur.Name.NS + "}" + cur.Name.Local
		}
		if cur.Name.Local != "" {
			return cur.Name.Local
		}
	}
	// Anonymous type with no named ancestor in the base chain: the type
	// was successfully validated, so it implicitly derives from xs:anyType.
	// Returning xs:untyped here would be wrong — xs:untyped means the
	// element was never validated, while xs:anyType means "validated but
	// the type is anonymous."
	return "xs:anyType"
}

// assertAnnotationName returns the PSVI annotation name recorded for a NODE in the
// xs:assert evaluation tree. For an INLINE ANONYMOUS list/union simple type (whose
// list-item / union-member metadata would be lost once xsdTypeName collapses it to a
// named ancestor) it mints a stable synthetic name and registers the actual
// *TypeDef so schemaDecls can recover the metadata; otherwise it is xsdTypeName. In
// either case the anonymous list-item / union-member types reachable from a list/
// union are registered too (assertRegisterAnonChildren) — INCLUDING anonymous ATOMIC
// (faceted) members — so active-member selection validates the actual faceted member
// rather than a collapsed builtin ancestor. Used ONLY for the assert annotations map;
// the user-facing annotations map keeps xsdTypeName, byte-identical. A standalone
// anonymous ATOMIC node type keeps xsdTypeName so node annotations are not broadly
// changed.
func (vc *validationContext) assertAnnotationName(td *TypeDef) string {
	if td == nil || vc.assertAnonNames == nil {
		return xsdTypeName(td)
	}
	if name, ok := vc.assertAnonNames[td]; ok {
		return name
	}
	switch resolveVariety(td) {
	case TypeVarietyList, TypeVarietyUnion:
		// Register anonymous members/items (recursively) regardless of whether the
		// list/union itself is named, so a NAMED union with inline anonymous members
		// still resolves each member's facets.
		vc.assertRegisterAnonChildren(td)
		if td.Name.Local == "" && td.Name.NS == "" {
			return vc.assertRegisterAnon(td)
		}
	}
	return xsdTypeName(td)
}

// assertRegisterAnon registers an ANONYMOUS TypeDef of ANY variety (atomic
// restriction, list, or union) under a stable synthetic annotation name and returns
// it, recursing into its anonymous item/member types; a NAMED type is left unchanged
// (returns xsdTypeName) but its anonymous descendants are still registered. This lets
// an inline anonymous union member — even a plain faceted atomic restriction (e.g.
// xs:int with maxInclusive) — round-trip to its actual *TypeDef so ValidateCastWithNS
// validates its facets during active-member selection.
func (vc *validationContext) assertRegisterAnon(td *TypeDef) string {
	if td == nil || vc.assertAnonNames == nil {
		return xsdTypeName(td)
	}
	if name, ok := vc.assertAnonNames[td]; ok {
		return name
	}
	name := xsdTypeName(td)
	if td.Name.Local == "" && td.Name.NS == "" {
		name = fmt.Sprintf("Q{%s}%d", assertAnonNS, len(vc.assertAnonTypes)+1)
		vc.assertAnonTypes[name] = td
		vc.assertAnonNames[td] = name
	}
	vc.assertRegisterAnonChildren(td)
	return name
}

// assertRegisterAnonChildren registers the anonymous list-item / union-member
// types reachable from td and its bases. A restriction wrapper can inherit its
// effective list/union variety from an anonymous base type, so the child
// metadata must be registered across the full base chain.
func (vc *validationContext) assertRegisterAnonChildren(td *TypeDef) {
	for cur := range baseChain(td) {
		if cur.ItemType != nil {
			vc.assertRegisterAnon(cur.ItemType)
		}
		for _, m := range cur.MemberTypes {
			vc.assertRegisterAnon(m)
		}
	}
}

// annotateElement records a type annotation for an element node. assessed reports
// whether the element was actually SCHEMA-ASSESSED (vs annotated purely for pass-2
// IDC canonicalization, as for skip/lax-no-declaration content): only an assessed
// element is recorded in assessedElemType, which the XSD 1.1 ID/IDREF pass
// consults. actualElemType is always recorded (post-xsi:type) for pass-2 IDC field
// canonicalization, independent of assessment. The assert annotations map (1.1)
// is populated only for assessed elements so xs:assert/xs:assertion tests atomize
// the PSVI-typed tree without typing processContents="skip" subtrees.
func (vc *validationContext) annotateElement(_ context.Context, elem *helium.Element, td *TypeDef, assessed bool) {
	if td != nil {
		if vc.actualElemType != nil {
			vc.actualElemType[elem] = td
		}
		if assessed && vc.assessedElemType != nil {
			vc.assessedElemType[elem] = td
		}
	}
	if assessed && vc.assertAnnotations != nil {
		vc.assertAnnotations[elem] = vc.assertAnnotationName(td)
	}
	if vc.cfg == nil || vc.cfg.annotations == nil {
		return
	}
	(*vc.cfg.annotations)[elem] = xsdTypeName(td)
}

// recordElemDecl records the resolved *ElementDecl matched for an element
// instance during pass-1 content validation, so pass-2 identity-constraint
// evaluation can recover declarations — including LOCAL ones — that
// lookupElemDecl (global-only) cannot. Called at the content-model match sites
// where the matched declaration is known.
func (vc *validationContext) recordElemDecl(elem *helium.Element, decl *ElementDecl) {
	if vc.actualElemDecl != nil && decl != nil {
		vc.actualElemDecl[elem] = decl
	}
}

// attrUseType resolves the effective simple type for an attribute use. An inline
// anonymous <xs:simpleType> (au.Type) takes precedence over a named type
// reference (au.TypeName).
func (vc *validationContext) attrUseType(au *AttrUse) (*TypeDef, bool) {
	if au.Type != nil {
		return au.Type, true
	}
	if au.TypeName.Local == "" {
		return nil, false
	}
	return vc.schema.LookupType(au.TypeName.Local, au.TypeName.NS)
}

// annotateAttrUse records a type annotation for an attribute node based on its AttrUse declaration.
func (vc *validationContext) annotateAttrUse(_ context.Context, a *helium.Attribute, au *AttrUse) {
	td, ok := vc.attrUseType(au)
	if !ok {
		return
	}
	// Record the assessed attribute's type for the XSD 1.1 ID/IDREF pass,
	// independent of the optional user-facing annotations map.
	if vc.actualAttrType != nil {
		vc.actualAttrType[a] = td
	}
	// Record the assert annotation (1.1) so an xs:assert/xs:assertion atomizes this
	// attribute in its schema value space.
	if vc.assertAnnotations != nil {
		vc.assertAnnotations[a] = vc.assertAnnotationName(td)
	}
	if vc.cfg == nil || vc.cfg.annotations == nil {
		return
	}
	(*vc.cfg.annotations)[a] = xsdTypeName(td)
}
