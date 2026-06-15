package xsd_test

import "testing"

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
