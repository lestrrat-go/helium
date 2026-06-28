package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAXPathDefaultNSInheritRoot covers CTA-861-001: an inherited
// schema-level xpathDefaultNamespace="##defaultNamespace" must resolve against the
// SCHEMA ROOT (where the attribute was declared), NOT against the xs:alternative
// element, even when the alternative redeclares the default namespace (xmlns).
//
// The root default namespace is urn:A, so the inherited ##defaultNamespace must be
// urn:A. The xs:alternative redeclares xmlns="urn:B". Its unprefixed @test name
// test (self::item) therefore selects the Pos type ONLY when the default element
// namespace resolves to urn:A (matching the {urn:A}item context node). With the
// pre-fix behavior the inherited value resolved to urn:B, so self::item missed and
// the declared Base type governed instead — a false ACCEPT of a non-positive value.
func TestVersion11CTAXPathDefaultNSInheritRoot(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns="urn:A" xmlns:a="urn:A" targetNamespace="urn:A"
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

	t.Run("inherited ##defaultNamespace resolves to root (urn:A): Pos selected, -5 rejected", func(t *testing.T) {
		t.Parallel()
		// self::item must match {urn:A}item -> Pos governs -> value=-5 is not a
		// positiveInteger -> invalid. The pre-fix bug resolved the inherited default
		// to urn:B, selected Base, and accepted -5.
		require.ErrorIs(t, validate(t, `<item xmlns="urn:A" value="-5"/>`), xsd.ErrValidationFailed)
	})

	t.Run("Pos selected: positive value accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, `<item xmlns="urn:A" value="5"/>`))
	})
}
