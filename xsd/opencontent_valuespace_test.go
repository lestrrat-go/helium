package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_DropConstraintValueSpace covers the soundness finding that
// globalDropsLocalConstraint must compare `fixed`/`default` in VALUE space (via
// fixedValueMatches, honoring each declaration's FixedNS/DefaultNS), not lexically:
// a QName/NOTATION value with the same lexical form but a different prefix→namespace
// binding is a DIFFERENT value (must reject), while a numeric value differing only
// lexically (e.g. "1.0" vs "1.00") is EQUAL (must accept).
func TestOpenContent_DropConstraintValueSpace(t *testing.T) {
	t.Parallel()

	// base B declares interleave-strict open content + a local element e (baseLocalE,
	// dropped by R); the global e (globalE) re-admits it.
	build := func(globalE, baseLocalE string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  ` + globalE + `
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence>
      ` + baseLocalE + `
      <xs:element name="keep" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="t:B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element name="keep" type="xs:string" minOccurs="0"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}

	t.Run("QName fixed with same lexical but different namespace is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(
			`<xs:element name="e" type="xs:QName" fixed="p:v" xmlns:p="urn:global"/>`,
			`<xs:element name="e" type="xs:QName" minOccurs="0" fixed="p:v" xmlns:p="urn:base"/>`)
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "p:v@urn:base and p:v@urn:global are different QName values")
	})

	t.Run("QName fixed with the same namespace value is accepted", func(t *testing.T) {
		t.Parallel()
		schema := build(
			`<xs:element name="e" type="xs:QName" fixed="p:v" xmlns:p="urn:same"/>`,
			`<xs:element name="e" type="xs:QName" minOccurs="0" fixed="q:v" xmlns:q="urn:same"/>`)
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "p:v and q:v both bind to urn:same, so they are the same QName value")
	})

	t.Run("decimal fixed differing only lexically is accepted (value-space equal)", func(t *testing.T) {
		t.Parallel()
		schema := build(
			`<xs:element name="e" type="xs:decimal" fixed="1.00"/>`,
			`<xs:element name="e" type="xs:decimal" minOccurs="0" fixed="1.0"/>`)
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "1.0 and 1.00 are the same xs:decimal value")
	})

	t.Run("QName default with same lexical but different namespace is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(
			`<xs:element name="e" type="xs:QName" default="p:v" xmlns:p="urn:global"/>`,
			`<xs:element name="e" type="xs:QName" minOccurs="0" default="p:v" xmlns:p="urn:base"/>`)
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "the global default p:v@urn:global differs in value from the base local default p:v@urn:base")
	})

	t.Run("QName default with the same namespace value is accepted", func(t *testing.T) {
		t.Parallel()
		schema := build(
			`<xs:element name="e" type="xs:QName" default="p:v" xmlns:p="urn:same"/>`,
			`<xs:element name="e" type="xs:QName" minOccurs="0" default="q:v" xmlns:q="urn:same"/>`)
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "the defaults resolve to the same QName value")
	})
}
