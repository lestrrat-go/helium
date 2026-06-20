package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAttrGroupNestedRefExpansion verifies that the attribute uses of a NESTED
// xs:attributeGroup ref child are expanded into a referencing type's effective
// attributes. parseNamedAttributeGroup records only direct xs:attribute children
// in c.schema.attrGroups; the nested refs live in c.attrGroupRefChildren and were
// formerly dropped when a group was expanded into a type, so a required /
// defaulted / prohibited attribute declared in a transitively-referenced group
// was silently lost. The fix recursively expands attrGroupRefChildren when
// appending a group's attribute uses to a type.
func TestAttrGroupNestedRefExpansion(t *testing.T) {
	t.Parallel()

	// g2 -> g1; g1 declares a REQUIRED attribute. A type referencing g2 must
	// require 'a': <root/> (missing it) is INVALID, <root a="..."/> is valid.
	t.Run("nested required attribute is enforced", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1">
    <xs:attribute name="a" type="xs:string" use="required"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g2">
    <xs:attributeGroup ref="g1"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g2"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`

		var out string
		err := compileAndValidate(t, schemaXML, `<root/>`, &out)
		require.Error(t, err, "missing nested required attribute must be rejected; got: %q", out)

		require.NoError(t, compileAndValidate(t, schemaXML, `<root a="x"/>`, nil),
			"instance carrying the nested required attribute must be valid")
	})

	// A nested attribute with a default is materialized onto an instance that
	// omits it.
	t.Run("nested default attribute is materialized", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1">
    <xs:attribute name="a" type="xs:string" default="dflt"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g2">
    <xs:attributeGroup ref="g1"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g2"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`

		schema, errs := compileWithErrors(t, schemaXML)
		require.Empty(t, errs, "unexpected compile errors")

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Label("test.xml").Validate(t.Context(), doc))

		root := doc.DocumentElement()
		require.NotNil(t, root)
		var found bool
		for _, at := range root.Attributes() {
			if at.LocalName() == "a" {
				found = true
				require.Equal(t, "dflt", at.Value(),
					"nested default attribute must be materialized with its default value")
			}
		}
		require.True(t, found, "nested default attribute must be inserted into the document")
	})

	// A use="prohibited" attribute declared inside an <xs:attributeGroup> is
	// pointless: xmllint warns and SKIPS it, so it is NOT propagated as a blocking
	// use into the referencing type. Therefore an xs:anyAttribute wildcard in that
	// type still admits the attribute and <root a="x"/> is VALID. Propagating the
	// prohibition would wrongly block the wildcard (over-rejection).
	t.Run("prohibited attribute in attr group is skipped, wildcard admits", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:string"/>
  <xs:attributeGroup name="g1">
    <xs:attribute ref="a" use="prohibited"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g2">
    <xs:attributeGroup ref="g1"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g2"/>
    <xs:anyAttribute processContents="lax"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`

		// Compiles (the skip emits a warning, not a fatal error).
		_, errs := compileWithErrors(t, schemaXML)
		require.Empty(t, errs, "unexpected compile errors")

		// The wildcard still admits the prohibited-but-skipped attribute.
		require.NoError(t, compileAndValidate(t, schemaXML, `<root a="x"/>`, nil),
			"prohibited use inside an attribute group must be skipped so the wildcard admits the attribute")
	})

	// A use="prohibited" attribute declared with a NAME (not ref) inside an
	// <xs:attributeGroup> is likewise skipped.
	t.Run("named prohibited attribute in attr group is skipped", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1">
    <xs:attribute name="a" type="xs:string" use="prohibited"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g1"/>
    <xs:anyAttribute processContents="lax"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`

		_, errs := compileWithErrors(t, schemaXML)
		require.Empty(t, errs, "unexpected compile errors")
		require.NoError(t, compileAndValidate(t, schemaXML, `<root a="x"/>`, nil),
			"named prohibited use inside an attribute group must be skipped so the wildcard admits the attribute")
	})
}
