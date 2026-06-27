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

	t.Run("rejects derived builtin widening base named restricted type", func(t *testing.T) {
		t.Parallel()
		// Base @a is a USER simple type (SmallInt) that restricts xs:int with
		// maxInclusive="10". The derived restriction redeclares @a as xs:int —
		// widening the base back to its builtin ancestor and admitting values
		// (11, 12, ...) the base forbids. Must be rejected: walking the BASE side
		// to its builtin ancestor is unsound.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="SmallInt">
    <xs:restriction base="xs:int">
      <xs:maxInclusive value="10"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="t:SmallInt"/>
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
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects derived builtin widening base inline restricted type", func(t *testing.T) {
		t.Parallel()
		// Same widening, but the base attribute's restricted simple type is an
		// inline (anonymous) <xs:simpleType> rather than a named one.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a">
      <xs:simpleType>
        <xs:restriction base="xs:int">
          <xs:maxInclusive value="10"/>
        </xs:restriction>
      </xs:simpleType>
    </xs:attribute>
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
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects derived attr type from a different builtin family", func(t *testing.T) {
		t.Parallel()
		// Base @a is xs:int; the derived restriction redeclares @a as xs:boolean.
		// boolean is not derived from int (a non-string/non-numeric builtin pair),
		// so the builtin hierarchy must DECIDE this case and reject it, not treat
		// it as "unknown" and accept it.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:int"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:boolean"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts narrowing fixed value across types", func(t *testing.T) {
		t.Parallel()
		// Base @a is xs:decimal fixed="1.0"; the derived restriction narrows @a to
		// xs:int fixed="1". The two fixed lexicals are value-equal (1.0 == 1), but
		// "1.0" is not a valid xs:int lexical — so each must be compared under its
		// OWN type. A valid narrowing must be accepted.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:decimal" fixed="1.0"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:int" fixed="1"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts derived attr type that is a member of base union type", func(t *testing.T) {
		t.Parallel()
		// Base @a is a USER union (IntOrString = xs:int | xs:string) with no facets.
		// The derived restriction redeclares @a as xs:int — a member of the union,
		// so per cos-st-derived-ok.2.2.4 it is validly derived from the union and
		// must be ACCEPTED (the base's value space already admits every xs:int).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="IntOrString">
    <xs:union memberTypes="xs:int xs:string"/>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="t:IntOrString"/>
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

	t.Run("rejects builtin list type restricted by an atomic type", func(t *testing.T) {
		t.Parallel()
		// Base @a is xs:IDREFS (a builtin LIST type); the derived restriction
		// redeclares @a as xs:string (atomic). A list type is unrelated to an
		// atomic type, so the restriction admits values the base does not and must
		// be rejected — the builtin list types must be DECIDED, not treated as
		// "unknown" and silently accepted.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:IDREFS"/>
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
}
