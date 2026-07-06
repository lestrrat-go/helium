package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBaseWildcardPointlessSequence covers the §3.9.6 pointless-particle rule at
// the top of a complexContent restriction: a base whose content model is a
// sequence wrapping a SINGLE wildcard (xs:any) is equivalent to that wildcard
// particle, so a derived sequence restricting it is checked by
// NSRecurseCheckCardinality (Sequence:Any) — the derived group's total
// element-emission range must be within the base wildcard's occurrence range and
// every leaf must be admitted by the wildcard — NOT the element-by-element
// positional mapping (which would wrongly reject a derived element{1,1} against a
// base wildcard{2,3}). W3C particlesHa080 (valid) / particlesHa081 (invalid).
// Version-independent (both cases live in the xsd10 suite).
func TestBaseWildcardPointlessSequence(t *testing.T) {
	t.Parallel()

	const notValidRestriction = "not a valid restriction"

	t.Run("accepts derived sequence within base wildcard range", func(t *testing.T) {
		t.Parallel()
		// W3C particlesHa080: base sequence(any ##any {2,3}); derived
		// sequence(e, e, any ##targetNamespace) emits exactly 3 elements, within
		// the base wildcard's {2,3} range, and every leaf is admitted by ##any.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:any namespace="##any" minOccurs="2" maxOccurs="3"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:element name="e" type="xs:string"/>
          <xs:element name="e" type="xs:string"/>
          <xs:any namespace="##targetNamespace"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects derived sequence exceeding base wildcard maximum", func(t *testing.T) {
		t.Parallel()
		// W3C particlesHa081: base sequence(any ##any {2,2}); derived
		// sequence(e, e, any) emits 3 elements, exceeding the base wildcard's
		// maximum of 2 — an invalid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:any namespace="##any" minOccurs="2" maxOccurs="2"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:element name="e" type="xs:string"/>
          <xs:element name="e" type="xs:string"/>
          <xs:any namespace="##targetNamespace"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})
}
