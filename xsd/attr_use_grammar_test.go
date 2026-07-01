package xsd_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// TestAttributeUseGrammar covers the schema-representation constraints on the
// `use` attribute of <xs:attribute> (version-independent, enforced in the
// default XSD 1.0 compiler): `use` is prohibited on a global (top-level)
// attribute declaration, and on a local attribute use its value must be one of
// {optional, prohibited, required}. Mirrors W3C msMeta/Attribute_w3c.xml
// attF/attJ/attO cases.
func TestAttributeUseGrammar(t *testing.T) {
	const main = "test.xsd"
	compile := func(t *testing.T, schema string) string {
		t.Helper()
		fsys := fstest.MapFS{main: &fstest.MapFile{Data: []byte(schema)}}
		return compileFSErrors(t, fsys, main)
	}

	t.Run("use on global attribute is rejected", func(t *testing.T) {
		for _, use := range []string{"required", "optional", "prohibited"} {
			schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="foo" type="xs:string" use="` + use + `"/>
</xs:schema>`
			errStr := compile(t, schema)
			require.Contains(t, errStr, "The attribute 'use' is not allowed",
				"global attribute with use=%q must be rejected; got: %q", use, errStr)
		}
	})

	t.Run("invalid use enum on local attribute is rejected", func(t *testing.T) {
		for _, use := range []string{"default", "fixed", "foo", ""} {
			schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:attribute name="a" type="xs:string" use="` + use + `"/>
  </xs:complexType>
</xs:schema>`
			errStr := compile(t, schema)
			require.Contains(t, errStr, "must be one of 'optional', 'prohibited', or 'required'",
				"local attribute with use=%q must be rejected; got: %q", use, errStr)
		}
	})

	t.Run("valid use enum on local attribute compiles", func(t *testing.T) {
		for _, use := range []string{"optional", "prohibited", "required"} {
			schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:attribute name="a" type="xs:string" use="` + use + `"/>
  </xs:complexType>
</xs:schema>`
			errStr := compile(t, schema)
			require.Empty(t, errStr, "local attribute with use=%q must compile clean; got: %q", use, errStr)
		}
	})

	t.Run("global attribute without use compiles", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="foo" type="xs:string"/>
</xs:schema>`
		require.Empty(t, compile(t, schema))
	})
}
