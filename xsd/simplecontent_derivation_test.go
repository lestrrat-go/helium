package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileDefaultErr compiles schemaXML under the default (XSD 1.0) compiler and
// returns the compile error (nil on success).
func compileDefaultErr(t *testing.T, schemaXML string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	_, err = xsd.NewCompiler().Compile(t.Context(), doc)
	return err
}

// compile11Err compiles schemaXML under the XSD 1.1 compiler and returns the
// compile error (nil on success).
func compile11Err(t *testing.T, schemaXML string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	_, err = xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	return err
}

// TestSimpleContentRestrictionDerivation covers cos-st-restricts / Derivation
// Valid (Restriction, Simple): a simpleContent restriction's derived content
// simple type must validly RESTRICT the base's effective content simple type.
// Version-INDEPENDENT — enforced in both XSD 1.0 and 1.1.
func TestSimpleContentRestrictionDerivation(t *testing.T) {
	// particlesZ018: a list of xs:int is NOT a valid restriction of an
	// xs:decimal-content base — a list (base xs:anySimpleType) does not descend
	// from xs:decimal. Must be rejected in BOTH versions.
	listRestrictsDecimal := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B1">
    <xs:simpleContent>
      <xs:extension base="xs:decimal">
        <xs:attribute name="foo"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="C2">
    <xs:simpleContent>
      <xs:restriction base="B1">
        <xs:simpleType>
          <xs:list itemType="xs:int"/>
        </xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	t.Run("list restricting decimal content rejected (1.0)", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileDefaultErr(t, listRestrictsDecimal), xsd.ErrCompilationFailed)
	})
	t.Run("list restricting decimal content rejected (1.1)", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compile11Err(t, listRestrictsDecimal), xsd.ErrCompilationFailed)
	})

	// A nested simpleType whose primitive base is unrelated to the base content
	// (xs:string content narrowed to an xs:date-derived type) is also invalid.
	dateRestrictsString := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B1">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="foo"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="C2">
    <xs:simpleContent>
      <xs:restriction base="B1">
        <xs:simpleType>
          <xs:restriction base="xs:date"/>
        </xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	t.Run("unrelated primitive restriction rejected (1.0)", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileDefaultErr(t, dateRestrictsString), xsd.ErrCompilationFailed)
	})

	// GUARD: a valid enumeration narrowing of an xs:string-content base compiles.
	enumRestrictsString := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B1">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="foo"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="C2">
    <xs:simpleContent>
      <xs:restriction base="B1">
        <xs:simpleType>
          <xs:restriction base="xs:string">
            <xs:enumeration value="a"/>
            <xs:enumeration value="b"/>
          </xs:restriction>
        </xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	t.Run("enumeration narrowing of string accepted (1.0)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileDefaultErr(t, enumRestrictsString))
	})
	t.Run("enumeration narrowing of string accepted (1.1)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compile11Err(t, enumRestrictsString))
	})

	// GUARD: a facet-only narrowing (no nested simpleType) of a numeric content
	// base compiles (facets never change the primitive base).
	facetRestrictsInt := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B1">
    <xs:simpleContent>
      <xs:extension base="xs:int">
        <xs:attribute name="foo"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="C2">
    <xs:simpleContent>
      <xs:restriction base="B1">
        <xs:maxInclusive value="10"/>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	t.Run("facet-only narrowing accepted (1.0)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileDefaultErr(t, facetRestrictsInt))
	})
	t.Run("facet-only narrowing accepted (1.1)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compile11Err(t, facetRestrictsInt))
	})

	// GUARD (anySimpleType exemption): a list restricting an xs:anySimpleType-content
	// base IS valid — anything restricts the simple ur-type (§3.16.3 clause 2.2.3).
	listRestrictsAnySimpleType := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B1">
    <xs:simpleContent>
      <xs:extension base="xs:anySimpleType">
        <xs:attribute name="foo"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="C2">
    <xs:simpleContent>
      <xs:restriction base="B1">
        <xs:simpleType>
          <xs:list itemType="xs:int"/>
        </xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	t.Run("list restricting anySimpleType content accepted (1.0)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileDefaultErr(t, listRestrictsAnySimpleType))
	})
}
