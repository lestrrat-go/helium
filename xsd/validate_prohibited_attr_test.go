package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestProhibitedAttributeUse checks that an attribute use declared with
// use="prohibited" does not contribute an allowed attribute: an instance
// carrying such an attribute (with no attribute wildcard) is rejected with "is
// not allowed", matching xmllint, and a prohibited use must not block a
// same-QName non-prohibited use declared elsewhere (non-prohibited wins). The
// interaction with an attribute wildcard is version-specific: XSD 1.1 retains
// the prohibited use in {attribute uses} and rejects the name even when a
// wildcard would admit it, while XSD 1.0 has no such retention — the name
// matches no use and falls through to the {attribute wildcard}.
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

	t.Run("prohibited use and a wildcard is version-specific", func(t *testing.T) {
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

		// XSD 1.1 rejects the prohibited name outright — the wildcard cannot
		// re-admit it (W3C addB034/addB136/attZ002 are the 1.0 counterpart).
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML, `<root a="x"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)

		// XSD 1.0 (default): the prohibited use is absent from {attribute uses},
		// so the attribute matches no use and the lax wildcard admits it.
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), schemaXML, `<root a="x"/>`))
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

// TestDeclaredXMLAttributeValue verifies that in XSD 1.0 a declared
// XML-namespace attribute use (a ref to xml:base/xml:lang/xml:space/xml:id) not
// only satisfies its presence requirement but also has its VALUE validated
// against the standard xml: namespace type — xml:space against the {default,
// preserve} enumeration, xml:lang against the union of xs:language and the empty
// string, xml:base against xs:anyURI, and xml:id against xs:ID. (In XSD 1.1 xml:
// attributes are not special and are handled by the ordinary ref path.)
func TestDeclaredXMLAttributeValue(t *testing.T) {
	t.Parallel()

	schema := func(local, use string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:import namespace="http://www.w3.org/XML/1998/namespace"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:attribute ref="xml:` + local + `" use="` + use + `"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	}

	t.Run("xml:space enumeration is enforced", func(t *testing.T) {
		t.Parallel()
		s := schema("space", "required")
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), s, `<root xml:space="preserve"/>`))
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), s, `<root xml:space="default"/>`))
		require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler(), s, `<root xml:space="bogus"/>`), xsd.ErrValidationFailed)
	})

	t.Run("xml:lang language value is enforced", func(t *testing.T) {
		t.Parallel()
		s := schema("lang", "required")
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), s, `<root xml:lang="en-US"/>`))
		// The empty string is a legal xml:lang (the union's second member).
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), s, `<root xml:lang=""/>`))
		require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler(), s, `<root xml:lang="en US"/>`), xsd.ErrValidationFailed)
	})

	t.Run("xml:base accepts a URI and xml:id validates as ID", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), schema("base", "required"), `<root xml:base="a"/>`))
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), schema("id", "required"), `<root xml:id="i1"/>`))
		require.ErrorIs(t, compileAndValidateV(t, xsd.NewCompiler(), schema("id", "required"), `<root xml:id="1bad"/>`), xsd.ErrValidationFailed)
	})
}
