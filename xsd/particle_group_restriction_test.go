package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAllGroupRefMinOccurs verifies cos-all-limited enforcement on a direct
// xs:group reference that resolves to an xs:all model group: the referencing
// particle's {min occurs} must be 0 or 1 (and {max occurs} 1). A minOccurs="2"
// reference (W3C particlesEa023) is rejected in XSD 1.0. The all-group-ref
// occurrence rule is enforced independently of the UPA determinism check.
func TestAllGroupRefMinOccurs(t *testing.T) {
	t.Parallel()

	const wantMsg = "{min occurs} must be (0 | 1)"

	t.Run("rejects minOccurs=2 on all-group ref", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:group name="G">
    <xs:all>
      <xs:element name="a1"/>
      <xs:element name="a2"/>
    </xs:all>
  </xs:group>
  <xs:element name="doc">
    <xs:complexType>
      <xs:group ref="t:G" minOccurs="2"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), wantMsg)
	})

	t.Run("accepts minOccurs=0 on all-group ref", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:group name="G">
    <xs:all>
      <xs:element name="a1"/>
      <xs:element name="a2"/>
    </xs:all>
  </xs:group>
  <xs:element name="doc">
    <xs:complexType>
      <xs:group ref="t:G" minOccurs="0"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}

// TestGroupRestrictsElementPointlessness verifies that a derived model GROUP
// restricting a base single ELEMENT is accepted in XSD 1.0 ONLY when the group
// is §3.9.6-pointless (folds down to a single element via safe occurrence
// hoisting). A genuinely repeating group — whose own maxOccurs and whose member's
// maxOccurs are both > 1, emitting the element multiple times — is not a valid
// restriction (there is no Sequence/Choice:Element rule in XSD 1.0) and is
// rejected (W3C particlesHb011). A pointless 1/1 wrapper still restricts.
func TestGroupRestrictsElementPointlessness(t *testing.T) {
	t.Parallel()

	const notValidRestriction = "not a valid restriction"

	t.Run("rejects repeating group restricting an element", func(t *testing.T) {
		t.Parallel()
		// Base choice branch e1{0,10}; derived branch sequence maxOccurs="2" of
		// e1 maxOccurs="2" emits e1 one-to-four times — not a pointless wrapper.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="base">
    <xs:choice minOccurs="2" maxOccurs="unbounded">
      <xs:element name="e1" minOccurs="0" maxOccurs="10"/>
      <xs:element name="e2" minOccurs="0"/>
      <xs:element name="e3" minOccurs="0"/>
    </xs:choice>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="t:base">
          <xs:choice minOccurs="2" maxOccurs="unbounded">
            <xs:sequence maxOccurs="2">
              <xs:element name="e1" maxOccurs="2"/>
            </xs:sequence>
            <xs:element name="e2"/>
            <xs:element name="e3" minOccurs="1"/>
          </xs:choice>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects emptiable group over multi-occur member (emission hole)", func(t *testing.T) {
		t.Parallel()
		// Derived branch sequence{0,1} wrapping e1{2,2} emits e1 {0,2} times — a
		// hole at 1 — so it is NOT §3.9.6-pointless: folding to e1{0,2} would
		// widen the language (allow 1). The group{0,1} own max is 1, but the
		// member minOccurs=2 introduces the gap, so it must still be rejected.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="base">
    <xs:choice minOccurs="2" maxOccurs="unbounded">
      <xs:element name="e1" minOccurs="0" maxOccurs="10"/>
      <xs:element name="e2" minOccurs="0"/>
    </xs:choice>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="t:base">
          <xs:choice minOccurs="2" maxOccurs="unbounded">
            <xs:sequence minOccurs="0" maxOccurs="1">
              <xs:element name="e1" minOccurs="2" maxOccurs="2"/>
            </xs:sequence>
            <xs:element name="e2"/>
          </xs:choice>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts emptiable wrapper over at-most-once member", func(t *testing.T) {
		t.Parallel()
		// Derived branch sequence{0,1} wrapping e1{1,1} emits e1 {0,1} times —
		// no hole ({0}∪{1}=[0,1]) — so it folds exactly to e1{0,1}, a valid
		// restriction of base e1{0,10}. The optional group max is 1 and the
		// member minOccurs<=1, so the exact-fold branch accepts it.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="base">
    <xs:choice minOccurs="2" maxOccurs="unbounded">
      <xs:element name="e1" minOccurs="0" maxOccurs="10"/>
      <xs:element name="e2" minOccurs="0"/>
    </xs:choice>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="t:base">
          <xs:choice minOccurs="2" maxOccurs="unbounded">
            <xs:sequence minOccurs="0" maxOccurs="1">
              <xs:element name="e1"/>
            </xs:sequence>
            <xs:element name="e2"/>
          </xs:choice>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts pointless single-element wrapper restricting an element", func(t *testing.T) {
		t.Parallel()
		// Base choice branch e1{0,10}; derived branch is a 1/1 sequence wrapping a
		// single e1 — pointless, folds to e1 which validly restricts base e1.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="base">
    <xs:choice minOccurs="2" maxOccurs="unbounded">
      <xs:element name="e1" minOccurs="0" maxOccurs="10"/>
      <xs:element name="e2" minOccurs="0"/>
    </xs:choice>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="t:base">
          <xs:choice minOccurs="2" maxOccurs="unbounded">
            <xs:sequence>
              <xs:element name="e1"/>
            </xs:sequence>
            <xs:element name="e2"/>
          </xs:choice>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts substitution-member choice restricting a base element", func(t *testing.T) {
		t.Parallel()
		// Base sequence(element ref=head) where head is a substitution-group head
		// with concrete members m1/m2; a derived choice(m1, m2) restricts the base
		// element head — the base element admits its whole substitution group at
		// instance time, so {m1,m2} is a valid name-narrowing subset (W3C elemZ027).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="t:head"/>
  <xs:element name="m2" type="xs:string" substitutionGroup="t:head"/>
  <xs:complexType name="base">
    <xs:sequence><xs:element ref="t:head"/></xs:sequence>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="t:base">
          <xs:sequence>
            <xs:choice>
              <xs:element ref="t:m1"/>
              <xs:element ref="t:m2"/>
            </xs:choice>
          </xs:sequence>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects non-member element restricting a base element", func(t *testing.T) {
		t.Parallel()
		// x is NOT in head's substitution group, so a derived choice re-admitting x
		// is not a valid restriction of the base element head.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:element name="head" type="xs:string"/>
  <xs:element name="m1" type="xs:string" substitutionGroup="t:head"/>
  <xs:element name="x" type="xs:string"/>
  <xs:complexType name="base">
    <xs:sequence><xs:element ref="t:head"/></xs:sequence>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="t:base">
          <xs:sequence>
            <xs:choice>
              <xs:element ref="t:m1"/>
              <xs:element ref="t:x"/>
            </xs:choice>
          </xs:sequence>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects local element sharing member name but widening type", func(t *testing.T) {
		t.Parallel()
		// The global member m1 is xs:integer; a derived LOCAL element named m1 typed
		// xs:string is NOT a valid restriction — it would admit <m1>abc</m1>, which
		// the base (which admits m1 only through the global integer-typed member)
		// rejects. Matching by name alone would falsely accept this.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:element name="head" type="xs:integer"/>
  <xs:element name="m1" type="xs:integer" substitutionGroup="t:head"/>
  <xs:complexType name="base">
    <xs:sequence><xs:element ref="t:head"/></xs:sequence>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="t:base">
          <xs:sequence>
            <xs:choice>
              <xs:element name="m1" type="xs:string"/>
            </xs:choice>
          </xs:sequence>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})
}
