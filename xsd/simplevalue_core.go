package xsd

import (
	"context"
	"fmt"
	"iter"
	"regexp"
	"strings"

	"github.com/lestrrat-go/helium/internal/lexicon"
	valuepkg "github.com/lestrrat-go/helium/internal/xsd/value"
)

// baseChain yields td and then each type up its BaseType chain. The walk is
// cycle-safe (Floyd's tortoise/hare): a cyclic BaseType chain — an invalid
// schema that checkCircularSimpleTypes reports as ErrCompilationFailed —
// terminates the iteration instead of looping forever. Every base-chain walker
// in the package goes through this helper so none can hang on a cyclic type
// regardless of when it runs relative to the circular-type check. The generator
// does not escape its range statement, so it allocates nothing on the heap.
func baseChain(td *TypeDef) iter.Seq[*TypeDef] {
	return func(yield func(*TypeDef) bool) {
		slow := td
		for fast := td; fast != nil; {
			if !yield(fast) {
				return
			}
			fast = fast.BaseType
			if fast == nil {
				return
			}
			if !yield(fast) {
				return
			}
			fast = fast.BaseType
			slow = slow.BaseType
			if fast == slow {
				return
			}
		}
	}
}

// valueVersion maps the xsd package's Version to the internal value package's
// equivalent, so the lexical validators apply the matching 1.0-vs-1.1 rules.
func valueVersion(v Version) valuepkg.Version {
	if v == Version11 {
		return valuepkg.Version11
	}
	return valuepkg.Version10
}

// validateBuiltinValue validates a value against a builtin XSD type's lexical space.
func validateBuiltinValue(v, builtinLocal string, version Version) error {
	return valuepkg.ValidateBuiltin(v, builtinLocal, valueVersion(version))
}

// validateQName validates a QName value. xs:QName has no version-sensitive
// lexical rules, so the version is immaterial here.
func validateQName(v string) error {
	return valuepkg.ValidateBuiltin(v, "QName", valuepkg.Version10)
}

// languageRegex matches the lexical space of xs:language (RFC 3066).
var languageRegex = regexp.MustCompile(`^[a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*$`)

// resolveWhiteSpace returns the effective whiteSpace facet value for a type,
// walking the base type chain. Returns "collapse" as default per XSD spec
// (most derived types default to collapse).
func resolveWhiteSpace(td *TypeDef) string {
	for cur := range baseChain(td) {
		if cur.Facets != nil && cur.Facets.WhiteSpace != nil {
			return *cur.Facets.WhiteSpace
		}
		// Check if we've reached a built-in type with known whitespace behavior.
		if cur.Name.NS == lexicon.NamespaceXSD {
			switch cur.Name.Local {
			case "string":
				return "preserve"
			case lexicon.TypeNormalizedString:
				return "replace"
			}
			// All other built-in types default to collapse.
			return "collapse"
		}
	}
	return "collapse"
}

// whiteSpaceRestrictiveness ranks a whiteSpace facet value by how much it
// normalizes: preserve (0) < replace (1) < collapse (2). A derived simple type
// may only keep or increase the rank of its base's whiteSpace (XSD Part 2
// §4.3.6); it may never decrease it.
func whiteSpaceRestrictiveness(ws string) int {
	switch ws {
	case "preserve":
		return 0
	case "replace":
		return 1
	default: // collapse
		return 2
	}
}

// normalizeWhiteSpace applies XSD whitespace normalization to a value.
//   - "preserve": no change
//   - "replace": replace \t, \n, \r with space
//   - "collapse": replace + collapse consecutive spaces + trim
func normalizeWhiteSpace(value, mode string) string {
	switch mode {
	case "preserve":
		return value
	case "replace":
		return strings.Map(func(r rune) rune {
			if r == '\t' || r == '\n' || r == '\r' {
				return ' '
			}
			return r
		}, value)
	default: // "collapse"
		replaced := strings.Map(func(r rune) rune {
			if r == '\t' || r == '\n' || r == '\r' {
				return ' '
			}
			return r
		}, value)
		return collapseSpaces(replaced)
	}
}

// collapseSpaces collapses consecutive spaces and trims leading/trailing spaces.
func collapseSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := true // treat start as space to trim leading
	for i := range len(s) {
		if s[i] == ' ' {
			inSpace = true
		} else {
			if inSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteByte(s[i])
			inSpace = false
		}
	}
	return b.String()
}

// validateValue validates a text value against a simple type definition.
func validateValue(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, elemName, filename string, line int, vc *validationContext) error {
	// Apply whitespace normalization per the type's whiteSpace facet.
	trimmed := normalizeWhiteSpace(value, resolveWhiteSpace(td))

	if err := validateValueByVariety(ctx, value, trimmed, valueNS, td, elemName, filename, line, vc); err != nil {
		return err
	}

	// XSD 1.1 <xs:assertion> simple-type facet: evaluated only after the value is
	// known lexically and facet valid, with $value bound to the typed value.
	if vc.version == Version11 {
		return checkSimpleTypeAssertions(ctx, trimmed, valueNS, td, elemName, filename, line, vc)
	}
	return nil
}

// validateValueByVariety validates a value's lexical space and facets per td's
// variety, excluding the XSD 1.1 assertion facet (handled by validateValue once
// the value is otherwise valid).
func validateValueByVariety(ctx context.Context, value, trimmed string, valueNS map[string]string, td *TypeDef, elemName, filename string, line int, vc *validationContext) error {
	// Check if this is a list type.
	if resolveVariety(td) == TypeVarietyList {
		return validateListValue(ctx, trimmed, valueNS, td, elemName, filename, line, vc)
	}

	// Check if this is a union type.
	if resolveVariety(td) == TypeVarietyUnion {
		return validateUnionValue(ctx, value, valueNS, td, elemName, filename, line, vc)
	}

	// Find the builtin base type by walking the BaseType chain.
	builtinLocal := builtinBaseLocal(td)

	// Validate against the builtin type's lexical space.
	if err := validateBuiltinValue(trimmed, builtinLocal, vc.version); err != nil {
		typeName := typeDisplayName(td)
		msg := fmt.Sprintf("'%s' is not a valid value of the atomic type '%s'.", trimmed, typeName)
		vc.reportValidityError(ctx, filename, line, elemName, msg)
		return err
	}

	// For QName/NOTATION the lexical space only enforces NCName form; the value
	// is invalid unless any prefix is bound in scope. resolveLexicalQName
	// reports an error for an unbound prefix.
	if builtinLocal == lexicon.TypeQName || builtinLocal == lexicon.TypeNotation {
		if _, err := resolveLexicalQName(trimmed, valueNS); err != nil {
			typeName := typeDisplayName(td)
			msg := fmt.Sprintf("'%s' is not a valid value of the atomic type '%s'.", trimmed, typeName)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			return err
		}
	}

	// Validate facets along the type chain.
	return validateFacets(ctx, trimmed, valueNS, td, builtinLocal, elemName, filename, line, vc)
}

// resolveUnionMembers walks up the base type chain to find the union's member
// types. baseChain keeps the walk finite on a cyclic BaseType chain.
func resolveUnionMembers(td *TypeDef) []*TypeDef {
	for cur := range baseChain(td) {
		if len(cur.MemberTypes) > 0 {
			return cur.MemberTypes
		}
	}
	return nil
}

// validateUnionValue validates a value against a union type by trying each member type.
// If all member types fail, a union-level error is reported.
func validateUnionValue(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, elemName, filename string, line int, vc *validationContext) error {
	members := resolveUnionMembers(td)

	// First, check restriction facets on the union type itself (e.g., enumeration).
	// If the type has facets and the value doesn't match them, that's the error.
	// Suppress the per-facet error; report a union-level error instead.
	//
	// Enumeration on a union is defined in the value space of the active member
	// type. The active member is resolved INDEPENDENTLY for the instance value and
	// for each enumeration literal, then compared with ordered-union value-family
	// semantics (the same comparison fixed-value uses). A single instance-member
	// value space would mis-accept cross-member values — e.g. memberTypes=
	// "zeroString xs:int" with enumeration "0": the literal "0" is active in the
	// string member, so the instance "+0" (active in xs:int) must NOT match it
	// even though both look numeric. Non-enumeration facets still compare in the
	// instance's active-member value space via checkFacets.
	trimmed := normalizeWhiteSpace(value, resolveWhiteSpace(td))

	// Walk the restriction base chain and apply each step's facets. A restriction
	// derived from an already-enumerated union (with no new facets of its own)
	// must still honor the base union's facets, so the enumeration and the
	// non-enumeration facets are checked for every step that carries a FacetSet,
	// each resolved in that step's own union member context.
	for cur := range baseChain(td) {
		if cur.Facets == nil {
			continue
		}

		vc.suppressDepth++
		enumErr := checkUnionEnumeration(ctx, value, valueNS, cur, elemName, filename, line, vc)
		vc.suppressDepth--
		if enumErr != nil {
			typeName := unionTypeDisplayName(td)
			msg := fmt.Sprintf("'%s' is not a valid value of the %s.", trimmed, typeName)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			return enumErr
		}

		// Resolve the active member down to its LEAF basic (atomic) member —
		// descending through any nested unions — so the non-enumeration facets
		// (pattern/length/bounds) are evaluated with the leaf's builtin local, not
		// an intermediate union's empty local. A nested-union leaf (e.g.
		// outer=union(inner), inner=union(xs:string)) otherwise reaches checkFacets
		// with an empty builtinLocal and would mis-trigger the decimal range-facet
		// fallback on a string leaf.
		memberLocal := ""
		memberWS := resolveWhiteSpace(cur)
		if active := fixedUnionActiveMember(ctx, value, valueNS, resolveUnionMembers(cur), nil, vc.version); active != nil {
			memberLocal = builtinBaseLocal(active)
			memberWS = resolveWhiteSpace(active)
		}
		// Suppress the enumeration facet here — it was checked above in union value
		// space; the remaining facets (pattern, length, bounds) are evaluated in the
		// instance member's value space.
		nonEnum := *cur.Facets
		nonEnum.Enumeration = nil
		nonEnum.EnumerationNS = nil
		vc.suppressDepth++
		err := checkFacets(ctx, trimmed, valueNS, &nonEnum, memberLocal, memberWS, elemName, filename, line, vc)
		vc.suppressDepth--
		if err != nil {
			typeName := unionTypeDisplayName(td)
			msg := fmt.Sprintf("'%s' is not a valid value of the %s.", trimmed, typeName)
			vc.reportValidityError(ctx, filename, line, elemName, msg)
			return err
		}
	}

	// Try each member type. If any accepts the value, it's valid.
	// Suppress per-member errors; only report union-level on total failure.
	for _, member := range members {
		vc.suppressDepth++
		err := validateValue(ctx, value, valueNS, member, elemName, filename, line, vc)
		vc.suppressDepth--
		if err == nil {
			return nil
		}
	}

	// All member types failed — report union-level error.
	// Use raw value (not trimmed) for the error message to match libxml2 behavior.
	typeName := unionTypeDisplayName(td)
	msg := fmt.Sprintf("'%s' is not a valid value of the %s.", value, typeName)
	vc.reportValidityError(ctx, filename, line, elemName, msg)
	return fmt.Errorf("union validation failed")
}

// checkUnionEnumeration enforces a union type's enumeration facet in union value
// space. The instance value and each enumeration literal each resolve their own
// active member (ordered-union semantics) and are compared with the same
// value-family logic fixed-value comparison uses (fixedUnionMatches), recursing
// through list/nested-union member value spaces. This rejects cross-member
// look-alikes — a literal active in a string member is not value-equal to an
// instance active in a numeric member. Each literal's prefixes resolve against its
// captured EnumerationNS bindings; the instance's against valueNS.
func checkUnionEnumeration(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, elemName, filename string, line int, vc *validationContext) error {
	if td.Facets == nil || len(td.Facets.Enumeration) == 0 {
		return nil
	}
	for i, ev := range td.Facets.Enumeration {
		var enumNS map[string]string
		if i < len(td.Facets.EnumerationNS) {
			enumNS = td.Facets.EnumerationNS[i]
		}
		if fixedUnionMatches(ctx, value, ev, td, valueNS, enumNS, vc.schema, vc.version) {
			return nil
		}
	}
	set := "'" + strings.Join(td.Facets.Enumeration, "', '") + "'"
	msg := fmt.Sprintf("[facet 'enumeration'] The value '%s' is not an element of the set {%s}.", value, set)
	vc.reportValidityError(ctx, filename, line, elemName, msg)
	return fmt.Errorf("enumeration")
}

// unionTypeDisplayName returns the display name for a union type error message.
// Named types: "union type '{ns}name'"
// Anonymous types: "local union type"
func unionTypeDisplayName(td *TypeDef) string {
	if td.Name.Local == "" {
		return "local union type"
	}
	return fmt.Sprintf("union type '%s'", typeQualifiedName(td))
}

// resolveVariety returns the effective variety of a type, walking through
// restriction derivations to find the underlying variety.
func resolveVariety(td *TypeDef) TypeVariety {
	for cur := range baseChain(td) {
		if cur.Variety != TypeVarietyAtomic {
			return cur.Variety
		}
	}
	return TypeVarietyAtomic
}

// validateListValue validates a space-separated list value against a list type.
func validateListValue(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, elemName, filename string, line int, vc *validationContext) error {
	// Split value into items by XSD whitespace only (space, tab, CR, LF). Using
	// strings.Fields would also split on NBSP and other Unicode whitespace,
	// silently turning an invalid item like "1 2" into two valid tokens;
	// value.XSDFields keeps it a single token so per-item lexical validation
	// rejects it.
	var items []string
	if value != "" {
		items = valuepkg.XSDFields(value)
	}
	itemCount := len(items)

	// Check length facets using item count.
	var facetErr error
	for cur := range baseChain(td) {
		if cur.Facets != nil {
			if cur.Facets.Length != nil && itemCount != *cur.Facets.Length {
				msg := fmt.Sprintf("[facet 'length'] The value has a length of '%d'; this differs from the allowed length of '%d'.", itemCount, *cur.Facets.Length)
				vc.reportValidityError(ctx, filename, line, elemName, msg)
				facetErr = fmt.Errorf("length")
			}
			if cur.Facets.MinLength != nil && itemCount < *cur.Facets.MinLength {
				msg := fmt.Sprintf("[facet 'minLength'] The value has a length of '%d'; this underruns the allowed minimum length of '%d'.", itemCount, *cur.Facets.MinLength)
				vc.reportValidityError(ctx, filename, line, elemName, msg)
				facetErr = fmt.Errorf("minLength")
			}
			if cur.Facets.MaxLength != nil && itemCount > *cur.Facets.MaxLength {
				msg := fmt.Sprintf("[facet 'maxLength'] The value has a length of '%d'; this exceeds the allowed maximum length of '%d'.", itemCount, *cur.Facets.MaxLength)
				vc.reportValidityError(ctx, filename, line, elemName, msg)
				facetErr = fmt.Errorf("maxLength")
			}
		}
	}

	// Apply the value-level facets — enumeration and pattern — to the
	// whitespace-collapsed whole-list value, walking the base chain. The list's
	// length facets are interpreted as item counts above; checkFacets would
	// instead measure character length, so enumeration and pattern are applied
	// here on their own rather than via the generic checkFacets path.
	itemType := resolveItemType(td)
	for cur := range baseChain(td) {
		if cur.Facets == nil {
			continue
		}
		if err := checkListEnumeration(ctx, value, valueNS, cur.Facets, itemType, elemName, filename, line, vc); err != nil {
			facetErr = err
		}
		if err := checkListPattern(ctx, value, cur.Facets, elemName, filename, line, vc); err != nil {
			facetErr = err
		}
	}

	if facetErr != nil {
		typeName := typeQualifiedName(td)
		msg := fmt.Sprintf("'%s' is not a valid value of the list type '%s'.", value, typeName)
		vc.reportValidityError(ctx, filename, line, elemName, msg)
		return facetErr
	}

	// Validate each list item against the item type. valueNS threads the
	// in-scope namespace bindings down to each item so a list whose item type is
	// QName/NOTATION resolves item prefixes against the instance's namespaces.
	if itemType != nil {
		for _, item := range items {
			if err := validateValue(ctx, item, valueNS, itemType, elemName, filename, line, vc); err != nil {
				return err
			}
		}
	}

	return nil
}

// resolveItemType walks the type chain to find the item type for a list type.
func resolveItemType(td *TypeDef) *TypeDef {
	for cur := range baseChain(td) {
		if cur.ItemType != nil {
			return cur.ItemType
		}
	}
	return nil
}

// builtinBaseLocal returns the local name of the builtin XSD base type.
func builtinBaseLocal(td *TypeDef) string {
	for cur := range baseChain(td) {
		if cur.Name.NS == lexicon.NamespaceXSD && cur.Name.Local != "" {
			return cur.Name.Local
		}
	}
	return ""
}

// typeQualifiedName returns a namespace-qualified display name like "{ns}local".
func typeQualifiedName(td *TypeDef) string {
	if td.Name.Local == "" {
		if td.BaseType != nil {
			return typeQualifiedName(td.BaseType)
		}
		return ""
	}
	if td.Name.NS != "" {
		return fmt.Sprintf("{%s}%s", td.Name.NS, td.Name.Local)
	}
	return td.Name.Local
}

// typeDisplayName returns the display name for a type in error messages.
// Named user types use their local name; XSD builtins use "xs:" prefix.
func typeDisplayName(td *TypeDef) string {
	if td.Name.Local == "" {
		// Anonymous type — use the base type's display name.
		if td.BaseType != nil {
			return typeDisplayName(td.BaseType)
		}
		return ""
	}
	if td.Name.NS == lexicon.NamespaceXSD {
		return "xs:" + td.Name.Local
	}
	return td.Name.Local
}

// checkListEnumeration enforces the enumeration facet on a list value. List
// enumeration members are themselves space-separated lists, and the comparison is
// performed in the item type's VALUE space (XSD §3.16): the instance and each
// enumeration member are split into items and compared item-by-item via
// fixedValueMatches on the item type, so an xs:list itemType="xs:int" with
// enumeration "1 2" accepts the value-equal instance "01 +2". QName/NOTATION item
// types resolve the instance items against valueNS and each member's items against
// the member's captured EnumerationNS bindings.
func checkListEnumeration(ctx context.Context, value string, valueNS map[string]string, fs *FacetSet, itemType *TypeDef, elemName, filename string, line int, vc *validationContext) error {
	if len(fs.Enumeration) == 0 {
		return nil
	}
	for i, ev := range fs.Enumeration {
		var enumNS map[string]string
		if i < len(fs.EnumerationNS) {
			enumNS = fs.EnumerationNS[i]
		}
		if fixedListMatches(ctx, value, ev, &TypeDef{Variety: TypeVarietyList, ItemType: itemType}, valueNS, enumNS, vc.schema, vc.version) {
			return nil
		}
	}
	set := "'" + strings.Join(fs.Enumeration, "', '") + "'"
	msg := fmt.Sprintf("[facet 'enumeration'] The value '%s' is not an element of the set {%s}.", value, set)
	vc.reportValidityError(ctx, filename, line, elemName, msg)
	return fmt.Errorf("enumeration")
}

// checkListPattern enforces the pattern facet on the whole-list value. Multiple
// patterns in the same restriction step are ORed, matching checkFacets.
func checkListPattern(ctx context.Context, value string, fs *FacetSet, elemName, filename string, line int, vc *validationContext) error {
	if len(fs.Patterns) == 0 {
		return nil
	}
	matched := false
	anyValid := false
	for _, re := range fs.compiledPatterns {
		if re == nil {
			continue
		}
		anyValid = true
		if re.MatchString(value) {
			matched = true
			break
		}
	}
	if !anyValid || matched {
		return nil
	}
	var msg string
	if len(fs.Patterns) == 1 {
		msg = fmt.Sprintf("[facet 'pattern'] The value '%s' is not accepted by the pattern '%s'.", value, fs.Patterns[0])
	} else {
		msg = fmt.Sprintf("[facet 'pattern'] The value '%s' is not accepted by the patterns '%s'.", value, strings.Join(fs.Patterns, "', '"))
	}
	vc.reportValidityError(ctx, filename, line, elemName, msg)
	return fmt.Errorf("pattern")
}

// validateFacets checks all applicable facets for a type and its ancestors.
func validateFacets(ctx context.Context, value string, valueNS map[string]string, td *TypeDef, builtinLocal, elemName, filename string, line int, vc *validationContext) error {
	// Collect all facets along the type chain (most derived first).
	var anyErr error
	ws := resolveWhiteSpace(td)
	for cur := range baseChain(td) {
		if cur.Facets != nil {
			if err := checkFacets(ctx, value, valueNS, cur.Facets, builtinLocal, ws, elemName, filename, line, vc); err != nil {
				anyErr = err
			}
		}
	}
	return anyErr
}
