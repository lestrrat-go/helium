package xsd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// validateBuiltinValue validates a value against a builtin XSD type's lexical space.
func validateBuiltinValue(v, builtinLocal string) error {
	return value.ValidateBuiltin(v, builtinLocal)
}

// validateQName validates a QName value.
func validateQName(v string) error {
	return value.ValidateBuiltin(v, "QName")
}

// languageRegex matches the lexical space of xs:language (RFC 3066).
var languageRegex = regexp.MustCompile(`^[a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*$`)

// resolveWhiteSpace returns the effective whiteSpace facet value for a type,
// walking the base type chain. Returns "collapse" as default per XSD spec
// (most derived types default to collapse).
func resolveWhiteSpace(td *TypeDef) string {
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Facets != nil && cur.Facets.WhiteSpace != nil {
			return *cur.Facets.WhiteSpace
		}
		// Check if we've reached a built-in type with known whitespace behavior.
		if cur.Name.NS == lexicon.NamespaceXSD {
			switch cur.Name.Local {
			case "string":
				return "preserve"
			case "normalizedString":
				return "replace"
			}
			// All other built-in types default to collapse.
			return "collapse"
		}
	}
	return "collapse"
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

	// Check if this is a list type.
	if resolveVariety(td) == TypeVarietyList {
		return validateListValue(ctx, trimmed, td, elemName, filename, line, vc)
	}

	// Check if this is a union type.
	if resolveVariety(td) == TypeVarietyUnion {
		return validateUnionValue(ctx, value, valueNS, td, elemName, filename, line, vc)
	}

	// Find the builtin base type by walking the BaseType chain.
	builtinLocal := builtinBaseLocal(td)

	// Validate against the builtin type's lexical space.
	if err := validateBuiltinValue(trimmed, builtinLocal); err != nil {
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

// resolveUnionMembers walks up the base type chain to find the union's member types.
func resolveUnionMembers(td *TypeDef) []*TypeDef {
	cur := td
	for cur != nil {
		if len(cur.MemberTypes) > 0 {
			return cur.MemberTypes
		}
		cur = cur.BaseType
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
	// type (the member that accepts the value), not purely lexically. Resolve
	// that member's builtin base type so checkFacets compares enumeration in the
	// correct value space — e.g. a union of xs:int with enumeration "5" accepts
	// "+5". When no member accepts the value the per-member loop below reports
	// the failure, so an empty builtinLocal here is harmless.
	trimmed := normalizeWhiteSpace(value, resolveWhiteSpace(td))
	if td.Facets != nil {
		memberLocal := ""
		if active := unionActiveMemberNS(ctx, value, valueNS, td); active != nil {
			memberLocal = builtinBaseLocal(active)
		}
		vc.suppressDepth++
		err := checkFacets(ctx, trimmed, valueNS, td.Facets, memberLocal, elemName, filename, line, vc)
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

// unionActiveMemberNS returns the union member type that accepts value under the
// given namespace context, resolving prefixes (for QName/NOTATION members) from
// valueNS. It mirrors unionActiveMember but threads an in-scope namespace map
// directly rather than a DOM node. Returns nil when no member accepts the value.
func unionActiveMemberNS(ctx context.Context, value string, valueNS map[string]string, td *TypeDef) *TypeDef {
	for _, m := range resolveUnionMembers(td) {
		if m == nil {
			continue
		}
		sub := &validationContext{
			errorHandler:  helium.NilErrorHandler{},
			suppressDepth: 1,
		}
		if validateValue(ctx, value, valueNS, m, "", "", 0, sub) == nil {
			return m
		}
	}
	return nil
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
	cur := td
	for cur != nil {
		if cur.Variety != TypeVarietyAtomic {
			return cur.Variety
		}
		cur = cur.BaseType
	}
	return TypeVarietyAtomic
}

// validateListValue validates a space-separated list value against a list type.
func validateListValue(ctx context.Context, value string, td *TypeDef, elemName, filename string, line int, vc *validationContext) error {
	// Split value into items by whitespace.
	var items []string
	if value != "" {
		items = strings.Fields(value)
	}
	itemCount := len(items)

	// Check length facets using item count.
	var facetErr error
	cur := td
	for cur != nil {
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
		cur = cur.BaseType
	}

	// Apply the value-level facets — enumeration and pattern — to the
	// whitespace-collapsed whole-list value, walking the base chain. The list's
	// length facets are interpreted as item counts above; checkFacets would
	// instead measure character length, so enumeration and pattern are applied
	// here on their own rather than via the generic checkFacets path.
	for cur := td; cur != nil; cur = cur.BaseType {
		if cur.Facets == nil {
			continue
		}
		if err := checkListEnumeration(ctx, value, cur.Facets, elemName, filename, line, vc); err != nil {
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

	// Validate each list item against the item type.
	itemType := resolveItemType(td)
	if itemType != nil {
		for _, item := range items {
			if err := validateValue(ctx, item, nil, itemType, elemName, filename, line, vc); err != nil {
				return err
			}
		}
	}

	return nil
}

// resolveItemType walks the type chain to find the item type for a list type.
func resolveItemType(td *TypeDef) *TypeDef {
	cur := td
	for cur != nil {
		if cur.ItemType != nil {
			return cur.ItemType
		}
		cur = cur.BaseType
	}
	return nil
}

// builtinBaseLocal returns the local name of the builtin XSD base type.
func builtinBaseLocal(td *TypeDef) string {
	cur := td
	for cur != nil {
		if cur.Name.NS == lexicon.NamespaceXSD && cur.Name.Local != "" {
			return cur.Name.Local
		}
		cur = cur.BaseType
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

// checkListEnumeration enforces the enumeration facet on the whitespace-collapsed
// whole-list value. List enumeration values are themselves space-separated lists,
// so both the instance value and each member are collapsed before comparison.
func checkListEnumeration(ctx context.Context, value string, fs *FacetSet, elemName, filename string, line int, vc *validationContext) error {
	if len(fs.Enumeration) == 0 {
		return nil
	}
	collapsed := collapseSpaces(value)
	for _, ev := range fs.Enumeration {
		if collapseSpaces(ev) == collapsed {
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
	cur := td
	for cur != nil {
		if cur.Facets != nil {
			if err := checkFacets(ctx, value, valueNS, cur.Facets, builtinLocal, elemName, filename, line, vc); err != nil {
				anyErr = err
			}
		}
		cur = cur.BaseType
	}
	return anyErr
}
