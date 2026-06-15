package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPatternFacetXSDRegexConstructs checks that pattern facets using XSD regex
// constructs Go's RE2 does not support (\i, \c name-character escapes, \p{Is...}
// Unicode block escapes) are enforced rather than silently skipped.
func TestPatternFacetXSDRegexConstructs(t *testing.T) {
	t.Parallel()

	t.Run("name-character escapes \\i\\c*", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="t"/>
  <xs:simpleType name="t">
    <xs:restriction base="xs:string">
      <xs:pattern value="\i\c*"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

		if err := compileAndValidate(t, schema, "<root>abc</root>", nil); err != nil {
			t.Errorf("abc should be valid: %v", err)
		}
		var out string
		if err := compileAndValidate(t, schema, "<root>1abc</root>", &out); err == nil {
			t.Errorf("1abc should be invalid (starts with a digit), but validated")
		}
	})

	t.Run("unicode block escape \\p{IsBasicLatin}", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="t"/>
  <xs:simpleType name="t">
    <xs:restriction base="xs:string">
      <xs:pattern value="\p{IsBasicLatin}+"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

		if err := compileAndValidate(t, schema, "<root>abc</root>", nil); err != nil {
			t.Errorf("abc should be valid: %v", err)
		}
		var out string
		if err := compileAndValidate(t, schema, "<root>café</root>", &out); err == nil {
			t.Errorf("café should be invalid (é is outside BasicLatin), but validated")
		}
	})
}

// TestPatternFacetCharClassSubtraction checks that XML Schema character-class
// subtraction ([a-z-[aeiou]]) is honored. RE2 has no subtraction, so the value
// is matched with the regexp2 backtracking engine instead.
func TestPatternFacetCharClassSubtraction(t *testing.T) {
	t.Parallel()

	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="t"/>
  <xs:simpleType name="t">
    <xs:restriction base="xs:string">
      <xs:pattern value="[a-z-[aeiou]]"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

	// 'b' is in a-z but not in the subtracted vowels: must be accepted.
	if err := compileAndValidate(t, schema, "<root>b</root>", nil); err != nil {
		t.Errorf("b should be valid ([a-z] minus vowels): %v", err)
	}
	// 'a' is a vowel: must be rejected.
	var out string
	if err := compileAndValidate(t, schema, "<root>a</root>", &out); err == nil {
		t.Errorf("a should be invalid (subtracted vowel), but validated")
	}
}

// TestPatternFacetInvalidRegexp checks that a pattern facet whose value is not a
// valid XSD regular expression is reported as a schema error instead of being
// silently ignored.
func TestPatternFacetInvalidRegexp(t *testing.T) {
	t.Parallel()

	schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="t"/>
  <xs:simpleType name="t">
    <xs:restriction base="xs:string">
      <xs:pattern value="[\q]"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

	_, errs := compileWithErrors(t, schema)
	require.NotEmpty(t, errs, "invalid pattern regexp should be a schema error")
	require.Contains(t, errs, "not a valid regular expression")
}

// TestPatternFacetRejectsNonXSDConstructs checks that constructs outside the XSD
// regex grammar are rejected even when the pattern would otherwise compile under
// the regexp2 backtracking engine. XSD has no back-references, and \b is a
// Perl-specific escape; both must be schema errors rather than silently accepted.
func TestPatternFacetRejectsNonXSDConstructs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		pattern string
	}{
		// Back-reference combined with character-class subtraction would route to
		// regexp2 (which supports back-references) and be silently accepted.
		{"backref with subtraction", `([a-z-[aeiou]])\1`},
		// \b word boundary compiles under RE2 but is not valid XSD regex.
		{"perl word boundary", `\bword\b`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="t"/>
  <xs:simpleType name="t">
    <xs:restriction base="xs:string">
      <xs:pattern value="` + tc.pattern + `"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

			_, errs := compileWithErrors(t, schema)
			require.NotEmpty(t, errs, "non-XSD regex construct should be a schema error")
		})
	}
}
