package xsd

import (
	"context"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
func fixedValueMatches(ctx context.Context, instance, fixed string, td *TypeDef, instanceNS, fixedNS map[string]string) bool {
	if td == nil {
		return instance == fixed
	}

	// Branch on variety *before* normalizing with the type's own whiteSpace
	// facet. A union type has no meaningful whiteSpace of its own — each member
	// applies its own facet — so normalizing here (the union default is
	// "collapse") would strip significant whitespace before an xs:string member
	// ever sees it. The union path therefore receives the raw values and lets
	// each member normalize with its own facet. List and atomic types keep their
	// type-level normalization.
	if resolveVariety(td) == TypeVarietyUnion {
		return fixedUnionMatches(ctx, instance, fixed, td, instanceNS, fixedNS)
	}

	ws := resolveWhiteSpace(td)
	ni := normalizeWhiteSpace(instance, ws)
	nf := normalizeWhiteSpace(fixed, ws)

	if resolveVariety(td) == TypeVarietyList {
		return fixedListMatches(ctx, ni, nf, td, instanceNS, fixedNS)
	}
	return fixedAtomicMatches(ni, nf, builtinBaseLocal(td), instanceNS, fixedNS)
}

// fixedListMatches compares two whitespace-normalized list values item by item
// in the list's item-type value space. Each item is dispatched through the
// variety-aware comparator on the actual item type, so a list whose item type is
// a union (or itself a list) is compared in the correct value space rather than
// raw lexical text.
func fixedListMatches(ctx context.Context, instance, fixed string, td *TypeDef, instanceNS, fixedNS map[string]string) bool {
	ii := value.XSDFields(instance)
	fi := value.XSDFields(fixed)
	if len(ii) != len(fi) {
		return false
	}
	itemType := resolveItemType(td)
	for i := range ii {
		if !fixedValueMatches(ctx, ii[i], fi[i], itemType, instanceNS, fixedNS) {
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
func fixedUnionMatches(ctx context.Context, instance, fixed string, td *TypeDef, instanceNS, fixedNS map[string]string) bool {
	members := resolveUnionMembers(td)

	fixedMember := fixedUnionActiveMember(ctx, fixed, fixedNS, members)
	if fixedMember == nil {
		return false
	}
	instanceMember := fixedUnionActiveMember(ctx, instance, instanceNS, members)
	if instanceMember == nil {
		return false
	}

	if fixedMember == instanceMember {
		// Same active member: compare in that member's value space. A union has no
		// whiteSpace facet of its own, so the raw values are forwarded and the
		// member normalizes both with its own facet inside fixedValueMatches.
		return fixedValueMatches(ctx, instance, fixed, fixedMember, instanceNS, fixedNS)
	}

	// Different active members. XSD 1.1 §2.3 — restrictions do not create new
	// values, so two values are equal iff they denote the same value in their
	// SHARED value space. This is dispatched on each member's variety: list members
	// compare item-by-item in their item type's value space (so an intList member
	// and a decimalList member both denote the same sequence of decimals), and
	// atomic members reduce to their primitive value-space family. Cross-variety
	// pairs (a list member vs an atomic member) have no shared value space and
	// remain unequal.
	return crossMemberValueEqual(ctx, instance, fixed, instanceMember, fixedMember, instanceNS, fixedNS)
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
func crossMemberValueEqual(ctx context.Context, instance, fixed string, instanceMember, fixedMember *TypeDef, instanceNS, fixedNS map[string]string) bool {
	return crossMemberValueEqualDepth(ctx, instance, fixed, instanceMember, fixedMember, instanceNS, fixedNS, 0)
}

func crossMemberValueEqualDepth(ctx context.Context, instance, fixed string, instanceMember, fixedMember *TypeDef, instanceNS, fixedNS map[string]string, depth int) bool {
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
		active := fixedUnionActiveMember(ctx, instance, instanceNS, resolveUnionMembers(instanceMember))
		if active == nil {
			return false
		}
		return crossMemberValueEqualDepth(ctx, instance, fixed, active, fixedMember, instanceNS, fixedNS, depth+1)
	}
	if fixedVariety == TypeVarietyUnion {
		active := fixedUnionActiveMember(ctx, fixed, fixedNS, resolveUnionMembers(fixedMember))
		if active == nil {
			return false
		}
		return crossMemberValueEqualDepth(ctx, instance, fixed, instanceMember, active, instanceNS, fixedNS, depth+1)
	}

	if instanceVariety == TypeVarietyList && fixedVariety == TypeVarietyList {
		ni := normalizeWhiteSpace(instance, resolveWhiteSpace(instanceMember))
		nf := normalizeWhiteSpace(fixed, resolveWhiteSpace(fixedMember))
		ii := strings.Fields(ni)
		fi := strings.Fields(nf)
		if len(ii) != len(fi) {
			return false
		}
		instanceItem := resolveItemType(instanceMember)
		fixedItem := resolveItemType(fixedMember)
		if instanceItem == nil || fixedItem == nil {
			return false
		}
		for i := range ii {
			if !crossMemberValueEqualDepth(ctx, ii[i], fi[i], instanceItem, fixedItem, instanceNS, fixedNS, depth+1) {
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
		return fixedValueMatches(ctx, instance, fixed, fixedMember, instanceNS, fixedNS)
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
	case "decimal", "integer",
		"nonPositiveInteger", "negativeInteger", "long", "int", "short", "byte",
		"nonNegativeInteger", "unsignedLong", "unsignedInt", "unsignedShort",
		"unsignedByte", "positiveInteger":
		return "decimal", true, true
	case "string", "normalizedString", "token", "language",
		"Name", "NCName", "ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES",
		"NMTOKEN", "NMTOKENS", "anyURI":
		// String value space equals the whitespace-processed lexical space; not
		// value-comparable via value.Compare, so the caller compares lexically.
		return lexicon.TypeString, false, true
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
func fixedUnionActiveMember(ctx context.Context, value string, valueNS map[string]string, members []*TypeDef) *TypeDef {
	for _, member := range members {
		vc := &validationContext{
			errorHandler:  helium.NilErrorHandler{},
			suppressDepth: 1,
		}
		if validateValue(ctx, value, valueNS, member, "", "", 0, vc) != nil {
			continue
		}
		// The member accepts the value. If it is itself a union, recurse to find
		// the active basic member within it; the validateValue success above
		// guarantees at least one nested member accepts the value.
		if resolveVariety(member) == TypeVarietyUnion {
			if basic := fixedUnionActiveMember(ctx, value, valueNS, resolveUnionMembers(member)); basic != nil {
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
		if (builtinLocal == "float" || builtinLocal == "double") &&
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
	cfg           *validateConfig
	filename      string
	errorHandler  helium.ErrorHandler
	suppressDepth int
	// actualElemType records the ACTUAL *TypeDef determined for each element
	// during pass-1 content validation, including any xsi:type override. Pass-2
	// identity-constraint field resolution consults this before falling back to
	// descending the declared content model, so an IDC field whose type is
	// contributed by xsi:type is canonicalized in the correct value space.
	actualElemType map[*helium.Element]*TypeDef
}

func newValidationContext(schema *Schema, cfg *validateConfig, filename string, handler helium.ErrorHandler) *validationContext {
	return &validationContext{
		schema:         schema,
		cfg:            cfg,
		filename:       filename,
		errorHandler:   handler,
		actualElemType: make(map[*helium.Element]*TypeDef),
	}
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
	vc := &validationContext{
		errorHandler: helium.NilErrorHandler{},
	}
	return validateValue(ctx, value, nsMap, td, "", "", 0, vc)
}

// ValidateElement validates an element's content against this type definition.
// This is used by XSLT xsl:type validation where the element is constructed
// in the result tree and must conform to the given type.
func (td *TypeDef) ValidateElement(ctx context.Context, elem *helium.Element, schema *Schema) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	collector := &validationErrors{}
	vc := newValidationContext(schema, &validateConfig{}, "", collector)
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
		edecl := lookupElemDecl(elem, vc.schema)
		if edecl != nil && len(edecl.IDCs) > 0 {
			if err := vc.validateIDConstraints(ctx, elem, edecl); err != nil {
				valid = false
			}
		}
		return nil
	}))

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

	td, err := vc.resolveXsiType(ctx, elem, declType)
	if err != nil {
		return err
	}
	// Check block flags against xsi:type derivation.
	if td != declType && isDerivationBlocked(td, declType, edecl.Block) {
		msg := "The xsi:type definition is blocked by the element declaration."
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		td = declType // fall back to declared type
	}
	if td != nil && td.Abstract {
		msg := msgAbstractType
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return fmt.Errorf("abstract type")
	}

	// Annotate root element with its type.
	vc.annotateElement(ctx, elem, td)

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
	// Validate attributes and annotate them.
	if err := vc.validateAttributes(ctx, elem, td); err != nil {
		return err
	}

	switch td.ContentType {
	case ContentTypeEmpty:
		return vc.validateEmptyContent(ctx, elem)
	case ContentTypeSimple:
		return vc.validateSimpleContent(ctx, elem, edecl, td)
	case ContentTypeElementOnly, ContentTypeMixed:
		// For element-only content, non-whitespace text children are not allowed.
		if td.ContentType == ContentTypeElementOnly {
			for child := range helium.Children(elem) {
				if child.Type() == helium.TextNode || child.Type() == helium.CDATASectionNode {
					if strings.TrimSpace(string(child.Content())) != "" {
						msg := "Character content other than whitespace is not allowed because the content type is 'element-only'."
						vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
						return fmt.Errorf("text content in element-only type")
					}
				}
			}
		}
		if td.ContentModel == nil {
			// No content model means anything goes (for mixed) or empty (for element-only).
			if td.ContentType == ContentTypeElementOnly {
				return vc.validateEmptyContent(ctx, elem)
			}
			// Mixed content with no model group (xs:anyType and similar lax/open
			// content) admits arbitrary child elements. Pass 2 IDC evaluation can
			// still reach descendants of this subtree, so each child must be
			// lax-annotated with its ACTUAL type (honoring xsi:type) and recursed
			// into — otherwise resolveFieldType falls back to declared types and
			// misses xsi:type overrides on descendants.
			return vc.annotateAnyTypeChildren(ctx, elem)
		}
		return vc.validateContentModel(ctx, elem, td.ContentModel)
	}
	return nil
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
			// Lax: no global declaration, so the child (and its subtree) is not
			// schema-assessed. Still record its own xsi:type ACTUAL type — a parent
			// IDC selecting this element directly must canonicalize an
			// xsi:type-introduced field in that type's value space — then recurse so
			// any deeper anyType descendant with a resolvable global declaration gets
			// annotated.
			if actual, ok := vc.resolveXsiTypeQuiet(ce); ok {
				vc.annotateElement(ctx, ce, actual)
			}
			if err := vc.annotateAnyTypeChildren(ctx, ce); err != nil {
				contentErr = err
			}
			continue
		}
		td, xsiErr := vc.resolveXsiType(ctx, ce, effectiveDeclType(edecl, vc.schema))
		if xsiErr != nil {
			contentErr = xsiErr
			continue
		}
		if td != nil && td.Abstract {
			msg := msgAbstractType
			vc.reportValidityError(ctx, vc.filename, ce.Line(), elemDisplayName(ce), msg)
			contentErr = fmt.Errorf("abstract type")
			continue
		}
		vc.annotateElement(ctx, ce, td)
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
	if actual, ok := vc.resolveXsiTypeQuiet(elem); ok {
		vc.annotateElement(ctx, elem, actual)
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
		// pass-2 can already derive from the content model, so record only that.
		if actual, ok := vc.resolveXsiTypeQuiet(ce); ok {
			vc.annotateElement(ctx, ce, actual)
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
		if !fixedValueMatches(ctx, value, *edecl.Fixed, fixedType, collectNSContext(elem), edecl.FixedNS) {
			msg := fmt.Sprintf("The element content '%s' does not match the fixed value constraint '%s'.", value, *edecl.Fixed)
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
			return fmt.Errorf("fixed value constraint")
		}
	}

	// Validate the text value against the type.
	if td != nil && (td.Facets != nil || resolveVariety(td) == TypeVarietyList || resolveVariety(td) == TypeVarietyUnion || builtinBaseLocal(td) != "" && builtinBaseLocal(td) != "string" && builtinBaseLocal(td) != "anySimpleType") {
		return validateValue(ctx, effectiveValue, collectNSContext(elem), td, elemDisplayName(elem), vc.filename, elem.Line(), vc)
	}

	return nil
}

func (vc *validationContext) validateEmptyContent(ctx context.Context, elem *helium.Element) error {
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
			if !isBlank(child.Content()) {
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

func isSpecialAttr(a *helium.Attribute) bool {
	p := a.Prefix()
	if p == "xmlns" || (p == "" && a.LocalName() == "xmlns") {
		return true
	}
	uri := a.URI()
	if uri == lexicon.NamespaceXSI {
		return true
	}
	if uri == lexicon.NamespaceXML {
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

func (vc *validationContext) validateAttributes(ctx context.Context, elem *helium.Element, td *TypeDef) error {
	var hasErr bool

	if len(td.Attributes) == 0 && td.AnyAttribute == nil {
		// No attribute declarations — check that instance has no attributes
		// (except xsi: namespace attributes and xmlns which are always allowed).
		for _, a := range elem.Attributes() {
			if isSpecialAttr(a) {
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

	// Build set of allowed attributes.
	allowed := make(map[QName]*AttrUse, len(td.Attributes))
	for _, au := range td.Attributes {
		allowed[au.Name] = au
	}

	// Build set of present instance attributes (excluding special attrs)
	// for O(1) lookups in the required-check and default-insertion loops.
	present := make(map[QName]struct{}, len(elem.Attributes()))

	// Check for unknown attributes and fixed value constraints.
	for _, a := range elem.Attributes() {
		if isSpecialAttr(a) {
			continue
		}
		aqn := QName{Local: a.LocalName(), NS: a.URI()}
		present[aqn] = struct{}{}
		if au, ok := allowed[aqn]; ok {
			// Resolve the declared type up front so the fixed-value check can
			// compare in the type's value space (applying its whitespace
			// facet) rather than by raw string equality.
			attrTD, tdOK := vc.attrUseType(au)
			if au.Fixed != nil && !fixedValueMatches(ctx, a.Value(), *au.Fixed, attrTD, collectNSContext(elem), au.FixedNS) {
				ad := attrDisplayName(a)
				msg := fmt.Sprintf("The value '%s' does not match the fixed value constraint '%s'.", a.Value(), *au.Fixed)
				vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
				hasErr = true
			}
			// Validate the attribute value against its declared type
			// (inline anonymous simpleType takes precedence over a named type).
			if tdOK && attrTD.ContentType == ContentTypeSimple {
				if err := attrTD.Validate(ctx, a.Value(), collectNSContext(elem)); err != nil {
					ad := attrDisplayName(a)
					msg := fmt.Sprintf("The value '%s' is not valid for the type of attribute '%s'.", a.Value(), ad)
					vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
					hasErr = true
				}
			}
			// Annotate the attribute with its declared type.
			vc.annotateAttrUse(ctx, a, au)
			continue
		}
		// Not in explicit declarations — check anyAttribute wildcard.
		if td.AnyAttribute != nil && wildcardMatchesAttr(td.AnyAttribute, a.URI()) {
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
		// Annotate the newly inserted attribute.
		for _, a := range elem.Attributes() {
			if a.LocalName() == au.Name.Local && a.URI() == au.Name.NS {
				vc.annotateAttrUse(ctx, a, au)
				break
			}
		}
	}

	if hasErr {
		return fmt.Errorf("attribute validation failed")
	}
	return nil
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

	// Enforce the global attribute's fixed-value constraint. A wildcard-matched
	// global fixed attribute must still satisfy its fixed value, in the declared
	// type's value space (mirroring the non-wildcard attribute path).
	if globalAttr.Fixed != nil && !fixedValueMatches(ctx, a.Value(), *globalAttr.Fixed, attrTD, collectNSContext(elem), globalAttr.FixedNS) {
		ad := attrDisplayName(a)
		msg := fmt.Sprintf("The value '%s' does not match the fixed value constraint '%s'.", a.Value(), *globalAttr.Fixed)
		vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
		return fmt.Errorf("fixed value constraint")
	}

	if ok && attrTD.ContentType == ContentTypeSimple {
		value := a.Value()
		if err := attrTD.Validate(ctx, value, collectNSContext(elem)); err != nil {
			ad := attrDisplayName(a)
			typeName := typeDisplayName(attrTD)
			msg := fmt.Sprintf("'%s' is not a valid value of the atomic type '%s'.", strings.TrimSpace(value), typeName)
			vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
			return err
		}
	}

	return nil
}

// wildcardMatchesAttr checks if an attribute namespace matches an anyAttribute wildcard.
func wildcardMatchesAttr(wc *Wildcard, attrNS string) bool {
	return wildcardMatches(wc, attrNS)
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

func isBlank(b []byte) bool {
	for _, c := range b {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
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
			if !isBlank(child.Content()) {
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
	if base.Name.Local == "anyType" && base.Name.NS == lexicon.NamespaceXSD {
		return true
	}
	for cur := derived.BaseType; cur != nil; cur = cur.BaseType {
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
func (vc *validationContext) resolveXsiType(ctx context.Context, elem *helium.Element, declaredType *TypeDef) (*TypeDef, error) {
	var xsiTypeVal string
	for _, a := range elem.Attributes() {
		if a.URI() == lexicon.NamespaceXSI && a.LocalName() == attrType {
			xsiTypeVal = a.Value()
			break
		}
	}
	if xsiTypeVal == "" {
		return declaredType, nil
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

	// Check derivation: xsi:type must be the same as or derived from the declared type.
	if declaredType != nil && !isDerivedFrom(td, declaredType) {
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
	for cur := td.BaseType; cur != nil; cur = cur.BaseType {
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

// annotateElement records a type annotation for an element node.
func (vc *validationContext) annotateElement(_ context.Context, elem *helium.Element, td *TypeDef) {
	// Always record the actual *TypeDef (post-xsi:type) for pass-2 IDC field
	// type resolution, independent of the optional user-facing annotations map.
	if vc.actualElemType != nil && td != nil {
		vc.actualElemType[elem] = td
	}
	if vc.cfg == nil || vc.cfg.annotations == nil {
		return
	}
	(*vc.cfg.annotations)[elem] = xsdTypeName(td)
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
	if vc.cfg == nil || vc.cfg.annotations == nil {
		return
	}
	td, ok := vc.attrUseType(au)
	if !ok {
		return
	}
	(*vc.cfg.annotations)[a] = xsdTypeName(td)
}
