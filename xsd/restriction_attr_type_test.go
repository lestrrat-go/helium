package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRestrictionAttrType (XSD-101) verifies derivation-ok-restriction for
// attribute uses: when a complexContent restriction redeclares a same-QName
// attribute, the derived attribute's type must be the same as, or derived by
// restriction from, the base attribute's type, and a base 'fixed' value
// constraint must be honoured. A base @a of xs:int restricted to a derived @a
// of xs:string is NOT a valid restriction and must be rejected at compile time.
func TestRestrictionAttrType(t *testing.T) {
	t.Parallel()

	const notValidRestriction = "is not a valid restriction of the corresponding attribute use"
	const fixedInconsistent = "is inconsistent with the 'fixed' value constraint"

	t.Run("rejects unrelated derived attr type restricting base attr type", func(t *testing.T) {
		t.Parallel()
		// Base @a is xs:int; the derived restriction redeclares @a as xs:string.
		// xs:string is not derived from xs:int, so the restriction admits values
		// (non-integers) the base does not and must be rejected.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:int"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:string"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts narrowing derived attr type", func(t *testing.T) {
		t.Parallel()
		// Base @a is xs:integer; the derived restriction narrows @a to xs:int,
		// which is derived by restriction from xs:integer — a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:integer"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:int"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts identical derived attr type", func(t *testing.T) {
		t.Parallel()
		// Control: redeclaring @a with the same type is a valid identity
		// restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:int"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:int"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects derived attr dropping base fixed value", func(t *testing.T) {
		t.Parallel()
		// Base @a is fixed to "1"; the derived restriction redeclares @a with no
		// value constraint, which would let any xs:int through — invalid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:int" fixed="1"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:int"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), fixedInconsistent)
	})

	t.Run("rejects derived attr with conflicting fixed value", func(t *testing.T) {
		t.Parallel()
		// Base @a is fixed to "1"; the derived restriction redeclares @a fixed to
		// "2" — a different value-space value, so invalid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:int" fixed="1"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:int" fixed="2"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), fixedInconsistent)
	})

	t.Run("accepts derived attr with value-space-equal fixed value", func(t *testing.T) {
		t.Parallel()
		// Base @a is fixed to "1"; the derived restriction redeclares @a fixed to
		// "01" — lexically different but value-space-equal for xs:int, so valid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:int" fixed="1"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:int" fixed="01"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}
