package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRestrictionParticleSubsumption (C-003) verifies that complexContent
// restriction derivation checks the content-model particles for the
// derivation-ok-restriction (Particle Valid (Restriction)) constraint, not just
// the attribute uses. A derived restriction whose content model is NOT a valid
// restriction of the base (reordered, added, or widened particles) must be
// rejected at compile time. Conservative valid restrictions (narrowing
// occurrence, dropping an optional trailing element) must still compile.
func TestRestrictionParticleSubsumption(t *testing.T) {
	t.Parallel()

	const notValidRestriction = "not a valid restriction"

	t.Run("rejects reordered sequence", func(t *testing.T) {
		t.Parallel()
		// Base sequence a,b ; derived restriction sequence b,a — not a valid
		// restriction (particle order must be preserved by recurse).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="b" type="xs:string"/>
          <xs:element name="a" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects added element", func(t *testing.T) {
		t.Parallel()
		// Base sequence a ; derived restriction adds b — a restriction may only
		// shrink, never add a particle the base does not allow.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
          <xs:element name="b" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects renamed element", func(t *testing.T) {
		t.Parallel()
		// Base element a ; derived restriction has element c with no counterpart.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="c" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects widened occurrence", func(t *testing.T) {
		t.Parallel()
		// Base allows a once; derived restriction allows it twice — widening
		// maxOccurs is not a restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="1"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="2"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts identical content model", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
          <xs:element name="b" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts narrowed occurrence", func(t *testing.T) {
		t.Parallel()
		// Base allows a 0..unbounded; derived restriction narrows to exactly 1.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="a" type="xs:string" minOccurs="1" maxOccurs="1"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts dropping optional trailing element", func(t *testing.T) {
		t.Parallel()
		// Base sequence a, b? ; derived restriction drops the optional b.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}
