package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProhibitedAttributeUse checks that an attribute use declared with
// use="prohibited" does not contribute an allowed attribute: an instance
// carrying such an attribute is rejected with "is not allowed", matching
// xmllint. A prohibited use must also never be admitted by an attribute
// wildcard, and must not block a same-QName non-prohibited use declared
// elsewhere (non-prohibited wins).
func TestProhibitedAttributeUse(t *testing.T) {
	t.Parallel()

	t.Run("prohibited ref attribute is rejected", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified"
  attributeFormDefault="qualified">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="t:a" use="prohibited"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		instanceXML := `<t:root xmlns:t="urn:t" t:a="x"/>`

		var out string
		err := compileAndValidate(t, schemaXML, instanceXML, &out)
		require.Error(t, err)
		require.Contains(t, out, "is not allowed")
	})

	t.Run("prohibited does not block same-QName non-prohibited use", func(t *testing.T) {
		t.Parallel()
		// An attribute group contributes a prohibited use of 'a', while the
		// complex type itself declares a non-prohibited use of the same name.
		// The non-prohibited use wins, so the attribute is accepted.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string" use="prohibited"/>
  </xs:attributeGroup>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute name="a" type="xs:string"/>
      <xs:attributeGroup ref="g"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

		require.NoError(t, compileAndValidate(t, schemaXML, `<root a="ok"/>`, nil))
	})

	t.Run("prohibited use is not admitted by a wildcard", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="a" use="prohibited"/>
      <xs:anyAttribute processContents="lax"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

		var out string
		err := compileAndValidate(t, schemaXML, `<root a="x"/>`, &out)
		require.Error(t, err)
		require.Contains(t, out, "is not allowed")
	})

	t.Run("non-prohibited attribute of same name accepted", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="a"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

		require.NoError(t, compileAndValidate(t, schemaXML, `<root a="x"/>`, nil))
	})
}
