package xmlschema

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// validateValue validates a text value against a simple type definition.
// It writes any errors to out and returns an error if the value is invalid.
func validateValue(value string, td *TypeDef, elemName, filename string, line int, out *strings.Builder) error {
	// Collapse whitespace: trim leading/trailing whitespace.
	trimmed := strings.TrimSpace(value)

	// Check if this is a list type.
	if resolveVariety(td) == TypeVarietyList {
		return validateListValue(trimmed, td, elemName, filename, line, out)
	}

	// Check if this is a union type.
	if resolveVariety(td) == TypeVarietyUnion {
		return validateUnionValue(value, td, elemName, filename, line, out)
	}

	// Find the builtin base type by walking the BaseType chain.
	builtinLocal := builtinBaseLocal(td)

	// Validate against the builtin type's lexical space.
	if err := validateBuiltinValue(trimmed, builtinLocal); err != nil {
		typeName := typeDisplayName(td)
		msg := fmt.Sprintf("'%s' is not a valid value of the atomic type '%s'.", trimmed, typeName)
		out.WriteString(validityError(filename, line, elemName, msg))
		return err
	}

	// Validate facets along the type chain.
	return validateFacets(trimmed, td, builtinLocal, elemName, filename, line, out)
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
func validateUnionValue(value string, td *TypeDef, elemName, filename string, line int, out *strings.Builder) error {
	members := resolveUnionMembers(td)

	// First, check restriction facets on the union type itself (e.g., enumeration).
	// If the type has facets and the value doesn't match them, that's the error.
	trimmed := strings.TrimSpace(value)
	if td.Facets != nil {
		var facetBuf strings.Builder
		if err := checkFacets(trimmed, td.Facets, "", elemName, filename, line, &facetBuf); err != nil {
			// Report union-level error instead of facet-specific error.
			typeName := unionTypeDisplayName(td)
			msg := fmt.Sprintf("'%s' is not a valid value of the %s.", trimmed, typeName)
			out.WriteString(validityError(filename, line, elemName, msg))
			return err
		}
	}

	// Try each member type. If any accepts the value, it's valid.
	for _, member := range members {
		var buf strings.Builder
		if err := validateValue(value, member, elemName, filename, line, &buf); err == nil {
			return nil
		}
	}

	// All member types failed — report union-level error.
	// Use raw value (not trimmed) for the error message to match libxml2 behavior.
	typeName := unionTypeDisplayName(td)
	msg := fmt.Sprintf("'%s' is not a valid value of the %s.", value, typeName)
	out.WriteString(validityError(filename, line, elemName, msg))
	return fmt.Errorf("union validation failed")
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
func validateListValue(value string, td *TypeDef, elemName, filename string, line int, out *strings.Builder) error {
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
				out.WriteString(validityError(filename, line, elemName, msg))
				facetErr = fmt.Errorf("length")
			}
			if cur.Facets.MinLength != nil && itemCount < *cur.Facets.MinLength {
				msg := fmt.Sprintf("[facet 'minLength'] The value has a length of '%d'; this underruns the allowed minimum length of '%d'.", itemCount, *cur.Facets.MinLength)
				out.WriteString(validityError(filename, line, elemName, msg))
				facetErr = fmt.Errorf("minLength")
			}
			if cur.Facets.MaxLength != nil && itemCount > *cur.Facets.MaxLength {
				msg := fmt.Sprintf("[facet 'maxLength'] The value has a length of '%d'; this exceeds the allowed maximum length of '%d'.", itemCount, *cur.Facets.MaxLength)
				out.WriteString(validityError(filename, line, elemName, msg))
				facetErr = fmt.Errorf("maxLength")
			}
		}
		cur = cur.BaseType
	}

	if facetErr != nil {
		typeName := typeQualifiedName(td)
		msg := fmt.Sprintf("'%s' is not a valid value of the list type '%s'.", value, typeName)
		out.WriteString(validityError(filename, line, elemName, msg))
		return facetErr
	}

	return nil
}

// builtinBaseLocal returns the local name of the builtin XSD base type.
func builtinBaseLocal(td *TypeDef) string {
	cur := td
	for cur != nil {
		if cur.Name.NS == xsdNS && cur.Name.Local != "" {
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
	if td.Name.NS == xsdNS {
		return "xs:" + td.Name.Local
	}
	return td.Name.Local
}

// validateBuiltinValue validates a value against a builtin XSD type's lexical space.
func validateBuiltinValue(value, builtinLocal string) error {
	switch builtinLocal {
	case "decimal":
		return validateDecimal(value)
	case "integer", "nonPositiveInteger", "negativeInteger",
		"long", "int", "short", "byte",
		"nonNegativeInteger", "unsignedLong", "unsignedInt", "unsignedShort", "unsignedByte",
		"positiveInteger":
		return validateInteger(value)
	case "hexBinary":
		return validateHexBinary(value)
	case "date":
		return validateDate(value)
	case "boolean":
		return validateBoolean(value)
	case "language":
		return validateLanguage(value)
	default:
		// No value-space validation for other types (string, etc.)
		return nil
	}
}

// decimalRegex matches the lexical space of xs:decimal.
// Pattern: optional sign, then digits with optional decimal point.
var decimalRegex = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)$`)

func validateDecimal(value string) error {
	if !decimalRegex.MatchString(value) {
		return fmt.Errorf("invalid decimal")
	}
	return nil
}

// integerRegex matches the lexical space of xs:integer.
var integerRegex = regexp.MustCompile(`^[+-]?\d+$`)

func validateInteger(value string) error {
	if !integerRegex.MatchString(value) {
		return fmt.Errorf("invalid integer")
	}
	return nil
}

// hexBinaryRegex matches the lexical space of xs:hexBinary.
// Must be even number of hex digits, or empty.
var hexBinaryRegex = regexp.MustCompile(`^([0-9a-fA-F]{2})*$`)

func validateHexBinary(value string) error {
	if !hexBinaryRegex.MatchString(value) {
		return fmt.Errorf("invalid hexBinary")
	}
	return nil
}

// dateRegex is a basic match for xs:date: YYYY-MM-DD with optional timezone.
var dateRegex = regexp.MustCompile(`^-?\d{4,}-\d{2}-\d{2}([Zz]|[+-]\d{2}:\d{2})?$`)

func validateDate(value string) error {
	if !dateRegex.MatchString(value) {
		return fmt.Errorf("invalid date")
	}
	return nil
}

// languageRegex matches the lexical space of xs:language (RFC 3066).
var languageRegex = regexp.MustCompile(`^[a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*$`)

func validateLanguage(value string) error {
	if !languageRegex.MatchString(value) {
		return fmt.Errorf("invalid language")
	}
	return nil
}

func validateBoolean(value string) error {
	switch value {
	case "true", "false", "1", "0":
		return nil
	}
	return fmt.Errorf("invalid boolean")
}

// validateFacets checks all applicable facets for a type and its ancestors.
func validateFacets(value string, td *TypeDef, builtinLocal, elemName, filename string, line int, out *strings.Builder) error {
	// Collect all facets along the type chain (most derived first).
	var anyErr error
	cur := td
	for cur != nil {
		if cur.Facets != nil {
			if err := checkFacets(value, cur.Facets, builtinLocal, elemName, filename, line, out); err != nil {
				anyErr = err
			}
		}
		cur = cur.BaseType
	}
	return anyErr
}

func checkFacets(value string, fs *FacetSet, builtinLocal, elemName, filename string, line int, out *strings.Builder) error {
	var anyErr error

	// Enumeration.
	if len(fs.Enumeration) > 0 {
		found := false
		for _, ev := range fs.Enumeration {
			if value == ev {
				found = true
				break
			}
		}
		if !found {
			set := "'" + strings.Join(fs.Enumeration, "', '") + "'"
			msg := fmt.Sprintf("[facet 'enumeration'] The value '%s' is not an element of the set {%s}.", value, set)
			out.WriteString(validityError(filename, line, elemName, msg))
			anyErr = fmt.Errorf("enumeration")
		}
	}

	// minInclusive.
	if fs.MinInclusive != nil {
		if !checkMinInclusive(value, *fs.MinInclusive) {
			msg := fmt.Sprintf("[facet 'minInclusive'] The value '%s' is less than the minimum value allowed ('%s').", value, *fs.MinInclusive)
			out.WriteString(validityError(filename, line, elemName, msg))
			anyErr = fmt.Errorf("minInclusive")
		}
	}

	// maxInclusive.
	if fs.MaxInclusive != nil {
		if !checkMaxInclusive(value, *fs.MaxInclusive) {
			msg := fmt.Sprintf("[facet 'maxInclusive'] The value '%s' is greater than the maximum value allowed ('%s').", value, *fs.MaxInclusive)
			out.WriteString(validityError(filename, line, elemName, msg))
			anyErr = fmt.Errorf("maxInclusive")
		}
	}

	// totalDigits.
	if fs.TotalDigits != nil {
		digits := countTotalDigits(value)
		if digits > *fs.TotalDigits {
			msg := fmt.Sprintf("[facet 'totalDigits'] The value '%s' has more digits than are allowed ('%d').", value, *fs.TotalDigits)
			out.WriteString(validityError(filename, line, elemName, msg))
			anyErr = fmt.Errorf("totalDigits")
		}
	}

	// Length facets — interpretation depends on the builtin base type.
	valueLen := facetLength(value, builtinLocal)

	if fs.Length != nil {
		if valueLen != *fs.Length {
			msg := fmt.Sprintf("[facet 'length'] The value has a length of '%d'; this differs from the allowed length of '%d'.", valueLen, *fs.Length)
			out.WriteString(validityError(filename, line, elemName, msg))
			anyErr = fmt.Errorf("length")
		}
	}

	if fs.MinLength != nil {
		if valueLen < *fs.MinLength {
			msg := fmt.Sprintf("[facet 'minLength'] The value has a length of '%d'; this underruns the allowed minimum length of '%d'.", valueLen, *fs.MinLength)
			out.WriteString(validityError(filename, line, elemName, msg))
			anyErr = fmt.Errorf("minLength")
		}
	}

	if fs.MaxLength != nil {
		if valueLen > *fs.MaxLength {
			msg := fmt.Sprintf("[facet 'maxLength'] The value has a length of '%d'; this exceeds the allowed maximum length of '%d'.", valueLen, *fs.MaxLength)
			out.WriteString(validityError(filename, line, elemName, msg))
			anyErr = fmt.Errorf("maxLength")
		}
	}

	// Pattern.
	if fs.Pattern != nil {
		re, err := regexp.Compile("^(?:" + *fs.Pattern + ")$")
		if err == nil && !re.MatchString(value) {
			msg := fmt.Sprintf("[facet 'pattern'] The value '%s' is not accepted by the pattern '%s'.", value, *fs.Pattern)
			out.WriteString(validityError(filename, line, elemName, msg))
			anyErr = fmt.Errorf("pattern")
		}
	}

	return anyErr
}

// checkMinInclusive compares value >= min as decimal numbers.
func checkMinInclusive(value, min string) bool {
	v, ok1 := new(big.Rat).SetString(value)
	m, ok2 := new(big.Rat).SetString(min)
	if !ok1 || !ok2 {
		return true // can't compare, don't error
	}
	return v.Cmp(m) >= 0
}

// checkMaxInclusive compares value <= max as decimal numbers.
func checkMaxInclusive(value, max string) bool {
	v, ok1 := new(big.Rat).SetString(value)
	m, ok2 := new(big.Rat).SetString(max)
	if !ok1 || !ok2 {
		return true
	}
	return v.Cmp(m) <= 0
}

// countTotalDigits counts the total number of significant digits in a decimal value.
// Per XML Schema spec: strip sign, then count digits in the numeral excluding
// leading zeros before the integer part and trailing zeros after the fraction.
// Examples: "0.123" → 3, "0.023" → 3, "123" → 3, "12.3" → 3, "0.0" → 1
func countTotalDigits(value string) int {
	// Strip sign.
	s := value
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}

	dotIdx := strings.Index(s, ".")
	if dotIdx < 0 {
		// No decimal point — count digits in integer, stripping leading zeros.
		s = strings.TrimLeft(s, "0")
		if s == "" {
			return 1
		}
		return len(s)
	}

	// Has decimal point. Integer part = s[:dotIdx], fraction part = s[dotIdx+1:]
	intPart := strings.TrimLeft(s[:dotIdx], "0")
	fracPart := strings.TrimRight(s[dotIdx+1:], "0")

	total := len(intPart) + len(fracPart)
	if total == 0 {
		return 1 // "0.0" has 1 digit
	}
	return total
}

// facetLength returns the effective length of a value for facet checking.
// The interpretation depends on the builtin base type.
func facetLength(value, builtinLocal string) int {
	switch builtinLocal {
	case "hexBinary":
		// Length in octets (bytes) = len(hexString) / 2.
		return len(value) / 2
	case "base64Binary":
		// Length in octets — simplified.
		s := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, value)
		s = strings.TrimRight(s, "=")
		return len(s) * 3 / 4
	default:
		// String types: length in characters.
		return len([]rune(value))
	}
}
