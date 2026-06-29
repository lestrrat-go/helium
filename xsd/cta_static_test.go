package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileCTASchema compiles src in XSD 1.1 mode and returns the resulting error
// (nil when the schema is valid).
func compileCTASchema(t *testing.T, src string) error {
	t.Helper()
	doc, perr := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, perr)
	_, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	return err
}

// TestVersion11CTAStaticErrors covers the XSD 1.1 conditional-type-assignment
// schema-representation constraints on the xs:alternative @test XPath and on the
// {type table} ordering, mirroring the saxonData/CTA cta9001err-cta9003err cases.
func TestVersion11CTAStaticErrors(t *testing.T) {
	// The user types live in urn:t (prefix t), so a t:-prefixed type reference in a
	// @test exercises the user-defined-type rejection (an UNPREFIXED user type is
	// already rejected by the underlying XPath prefix validation).
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string">
    <xs:attribute name="kind" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="der"><xs:simpleContent><xs:restriction base="t:base"/></xs:simpleContent></xs:complexType>
  <xs:simpleType name="smallInt"><xs:restriction base="xs:int"><xs:maxInclusive value="1"/></xs:restriction></xs:simpleType>`

	t.Run("testless alternative not last is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative type="t:der"/>
    <xs:alternative test="@kind='x'" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("testless final alternative is valid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="@kind='x'" type="t:der"/>
    <xs:alternative type="t:der"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})

	t.Run("undefined variable in test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="$kind='x'" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("user-defined type in instance-of test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="@kind instance of t:smallInt" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("built-in type in instance-of test is valid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="@kind instance of xs:string" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})

	t.Run("built-in type via non-xs prefix bound to XSD namespace is valid", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:x1="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string">
    <xs:attribute name="kind" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="der"><xs:simpleContent><xs:restriction base="base"/></xs:simpleContent></xs:complexType>
  <xs:element name="e" type="base">
    <xs:alternative test="@kind instance of x1:string" type="der"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})

	t.Run("cast to user-defined type in test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="@kind cast as t:der = 'x'" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})
}

// TestVersion11CTAStaticIsXSD10ByteIdentical confirms the new CTA static checks
// are gated on XSD 1.1: in 1.0 an xs:alternative is ignored entirely, so a schema
// that would trip a 1.1 CTA static error still compiles.
func TestVersion11CTAStaticIsXSD10ByteIdentical(t *testing.T) {
	src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string">
    <xs:attribute name="kind" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="der"><xs:simpleContent><xs:restriction base="base"/></xs:simpleContent></xs:complexType>
  <xs:element name="e" type="base">
    <xs:alternative test="$kind='x'" type="der"/>
    <xs:alternative type="der"/>
  </xs:element>
</xs:schema>`
	doc, perr := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, perr)
	_, err := xsd.NewCompiler().Compile(t.Context(), doc) // default = XSD 1.0
	require.NoError(t, err)
}

// TestVersion11CTAElementConsistentTypeTables covers the XSD 1.1 extension to
// Element Declarations Consistent (cos-element-consistent): two element particles
// with the same expanded name in one content model must have the SAME {type
// table}. Mirrors saxonData/CTA cta9009err (different tables) and cta9010err
// (table vs no table).
func TestVersion11CTAElementConsistentTypeTables(t *testing.T) {
	const types = `
  <xs:complexType name="zz"><xs:simpleContent><xs:extension base="xs:string">
    <xs:attribute name="type" type="xs:integer"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="zzi"><xs:simpleContent><xs:restriction base="zz">
    <xs:assertion test="$value castable as xs:integer"/></xs:restriction></xs:simpleContent></xs:complexType>
  <xs:complexType name="zzd"><xs:simpleContent><xs:restriction base="zz">
    <xs:assertion test="$value castable as xs:double"/></xs:restriction></xs:simpleContent></xs:complexType>`

	t.Run("different type tables are inconsistent", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing"><xs:sequence>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/><xs:alternative test="@type='2'" type="zzd"/></xs:element>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/></xs:element>
  </xs:sequence></xs:complexType>` + types + `
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("type table vs no table is inconsistent", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing"><xs:sequence>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/></xs:element>
    <xs:element name="a" type="zz"/>
  </xs:sequence></xs:complexType>` + types + `
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("identical type tables are consistent", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing"><xs:sequence>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/><xs:alternative test="@type='2'" type="zzd"/></xs:element>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/><xs:alternative test="@type='2'" type="zzd"/></xs:element>
  </xs:sequence></xs:complexType>` + types + `
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})
}
