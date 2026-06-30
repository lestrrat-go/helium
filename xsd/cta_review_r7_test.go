package xsd_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTADefaultXSDNamespacePredeclaredXS covers the CTA @test static
// check accepting a built-in xs: type name even when the schema document does NOT
// literally declare xmlns:xs but instead uses the default XSD namespace. xpath3
// predeclares the xs prefix in the static context (and Expression.Validate accepts
// it), so a @test like "@kind instance of xs:string" must NOT be rejected as a
// user-defined type just because the schema element lacks an explicit xmlns:xs.
func TestVersion11CTADefaultXSDNamespacePredeclaredXS(t *testing.T) {
	// The schema uses the DEFAULT XSD namespace (xmlns="...XSD") so its built-in type
	// references are unprefixed; a separate prefix (t) carries the target namespace
	// for the user types. Crucially NO xmlns:xs is declared, so the xs:alternative
	// @test's "xs:string" relies SOLELY on xpath3's predeclared xs prefix.
	const src = `<schema xmlns="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t">
  <complexType name="base"><simpleContent><extension base="string">
    <attribute name="kind" type="string"/></extension></simpleContent></complexType>
  <complexType name="der"><simpleContent><restriction base="t:base"/></simpleContent></complexType>
  <element name="e" type="t:base">
    <alternative test="@kind instance of xs:string" type="t:der"/>
  </element>
</schema>`
	require.NoError(t, compileCTASchema(t, src))
}

// TestVersion11CTAOverrideXPathDefaultNSToken covers finding #3: a surviving CTA
// alternative in an xs:override TARGET document must inherit the TARGET document's
// own xpathDefaultNamespace token, not the overriding document's stale value.
//
// a.xsd has xpathDefaultNamespace="##targetNamespace" and an element whose
// alternative @test="self::item" (an unprefixed name test) must resolve item to the
// target namespace, so the Pos type is selected and a value of -5 is rejected.
// main.xsd overrides a.xsd's simpleType t (so the item element survives) and has NO
// xpathDefaultNamespace of its own. If the per-document token is not threaded
// through the override load, the surviving alternative inherits main.xsd's empty
// token, item resolves to no namespace, Pos is never selected, and -5 is wrongly
// accepted.
func TestVersion11CTAOverrideXPathDefaultNSToken(t *testing.T) {
	const xsd11 = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	fsys := fstest.MapFS{
		fileMain: &fstest.MapFile{Data: []byte(`<xs:schema ` + xsd11 + ` targetNamespace="urn:tns" xmlns:t="urn:tns">
  <xs:override schemaLocation="a.xsd">
    <xs:simpleType name="t"><xs:restriction base="xs:integer"><xs:maxInclusive value="16"/></xs:restriction></xs:simpleType>
  </xs:override>
</xs:schema>`)},
		fileA: &fstest.MapFile{Data: []byte(`<xs:schema ` + xsd11 + ` targetNamespace="urn:tns" xmlns:t="urn:tns"
    elementFormDefault="qualified" xpathDefaultNamespace="##targetNamespace">
  <xs:simpleType name="t"><xs:restriction base="xs:integer"/></xs:simpleType>
  <xs:complexType name="Base"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="Pos"><xs:complexContent><xs:restriction base="t:Base">
    <xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="item" type="t:Base">
    <xs:alternative test="self::item" type="t:Pos"/>
  </xs:element>
</xs:schema>`)},
	}

	schema, err := compileOverride(t, fsys)
	require.NoError(t, err)

	// Pos selected via the target-namespace-resolved self::item test, so value=-5
	// (not a positiveInteger) must be rejected.
	require.ErrorIs(t, overrideValidate(t, schema, `<item xmlns="urn:tns" value="-5"/>`), xsd.ErrValidationFailed)

	// A positive value is accepted (sanity: the Pos type is genuinely selected).
	require.NoError(t, overrideValidate(t, schema, `<item xmlns="urn:tns" value="5"/>`))
}
