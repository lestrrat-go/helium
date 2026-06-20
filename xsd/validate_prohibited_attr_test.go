package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
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

	t.Run("prohibited fixed attribute is not inserted when absent", func(t *testing.T) {
		t.Parallel()
		// A prohibited use carrying a fixed value must never materialize that
		// value on an instance that omits the attribute: validating <root/>
		// succeeds and must not mutate the document by inserting a="x".
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified"
  attributeFormDefault="qualified">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="t:a" use="prohibited" fixed="x"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

		schema, errs := compileWithErrors(t, schemaXML)
		require.Empty(t, errs, "unexpected compile errors")

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<t:root xmlns:t="urn:t"/>`))
		require.NoError(t, err)

		require.NoError(t, xsd.NewValidator(schema).Label("test.xml").Validate(t.Context(), doc))

		root := doc.DocumentElement()
		require.NotNil(t, root)
		for _, a := range root.Attributes() {
			require.NotEqualf(t, "a", a.LocalName(),
				"prohibited fixed/default attribute must not be inserted into the document")
		}
	})

	t.Run("prohibited default attribute is not inserted when absent", func(t *testing.T) {
		t.Parallel()
		// Same as above but with a default value instead of fixed. (An
		// unqualified ref keeps the compile-time default-requires-optional
		// check from firing, isolating the insertion behavior.)
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="a" type="xs:string" default="x"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="a" use="prohibited"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

		schema, errs := compileWithErrors(t, schemaXML)
		require.Empty(t, errs, "unexpected compile errors")

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		require.NoError(t, xsd.NewValidator(schema).Label("test.xml").Validate(t.Context(), doc))

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Empty(t, root.Attributes(),
			"prohibited default attribute must not be inserted into the document")
	})

	t.Run("prohibited ref with default is rejected at compile time", func(t *testing.T) {
		t.Parallel()
		// default/fixed are incompatible with use="prohibited"; default
		// requires use="optional". The check must also apply to ref attributes.
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:attribute name="a" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="t:a" use="prohibited" default="x"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

		_, errs := compileWithErrors(t, schemaXML)
		require.Contains(t, errs,
			"must be 'optional' if the attribute 'default' is present")
	})
}
