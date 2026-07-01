package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAXPathDefaultNSInheritHost covers the XSD 1.1 {xpath default
// namespace} mapping for an xs:alternative whose schema-level
// xpathDefaultNamespace="##defaultNamespace" is INHERITED: per the spec the
// ##defaultNamespace keyword resolves against the [in-scope namespaces] of the
// HOST element (the xs:alternative), NOT the <schema> root, even though the
// attribute is declared on <schema>. (This is the spec-correct behavior; an
// earlier interpretation resolved it against the root.)
//
// The xs:alternative redeclares the default namespace via xmlns="urn:B", so the
// inherited ##defaultNamespace is urn:B. The unprefixed @test name test
// (self::item) therefore selects the Pos type only for a {urn:B}item context node.
func TestVersion11CTAXPathDefaultNSInheritHost(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns="urn:A" xmlns:a="urn:A" xmlns:b="urn:B" targetNamespace="urn:A"
    elementFormDefault="qualified" xpathDefaultNamespace="##defaultNamespace">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="value" type="xs:integer"/>
  </xs:complexType>
  <xs:complexType name="Pos">
    <xs:complexContent>
      <xs:restriction base="a:Base">
        <xs:sequence/>
        <xs:attribute name="value" type="xs:positiveInteger"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="item" type="a:Base">
    <xs:alternative test="self::item" type="a:Pos" xmlns="urn:B"/>
  </xs:element>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	require.NoError(t, err)

	validate := func(t *testing.T, instance string) error {
		t.Helper()
		idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, perr)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// The element declaration is {urn:A}item, so the only instance the schema can
	// validate is a {urn:A}item. With the inherited ##defaultNamespace resolving to
	// the alternative's redeclared default (urn:B), self::item tests {urn:B}item and
	// never matches the {urn:A}item context node, so Base governs and -5 is accepted.
	t.Run("inherited ##defaultNamespace resolves to host (urn:B): Base governs, -5 accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, `<item xmlns="urn:A" value="-5"/>`))
	})
}

// TestVersion11CTAXPathDefaultNSInheritHostNoRootDefault mirrors saxonData/CTA
// cta0005: the <schema> has xpathDefaultNamespace="##defaultNamespace" but NO
// default namespace of its own; the xs:alternative declares xmlns, so the inherited
// ##defaultNamespace resolves to the alternative's default namespace, letting an
// unprefixed name test in @test match the (namespaced) context node.
func TestVersion11CTAXPathDefaultNSInheritHostNoRootDefault(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:c" xmlns:c="urn:c"
    elementFormDefault="qualified" xpathDefaultNamespace="##defaultNamespace">
  <xs:complexType name="t">
    <xs:sequence><xs:element name="e" minOccurs="0" maxOccurs="unbounded" type="xs:decimal"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="treq">
    <xs:complexContent><xs:restriction base="c:t">
      <xs:sequence><xs:element name="e" minOccurs="1" maxOccurs="unbounded" type="xs:decimal"/></xs:sequence>
    </xs:restriction></xs:complexContent>
  </xs:complexType>
  <xs:element name="message" type="c:t">
    <xs:alternative test="self::message" type="c:treq" xmlns="urn:c"/>
  </xs:element>
</xs:schema>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	require.NoError(t, err)

	validate := func(t *testing.T, instance string) error {
		t.Helper()
		idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, perr)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// treq is selected (self::message matches {urn:c}message via the host's default
	// namespace), so an EMPTY message (no e) is invalid.
	t.Run("empty message rejected (treq selected via host default namespace)", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, validate(t, `<message xmlns="urn:c"/>`), xsd.ErrValidationFailed)
	})

	t.Run("non-empty message accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, `<message xmlns="urn:c"><e>1</e></message>`))
	})
}
