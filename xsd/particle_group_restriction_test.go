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

	t.Run("rejects local element widening an untyped member's inherited type", func(t *testing.T) {
		t.Parallel()
		// The global member m1 OMITS @type, so it inherits head's xs:integer. A
		// derived LOCAL element named m1 typed xs:string is NOT a valid restriction:
		// it admits <m1>abc</m1>, which the base (routing m1 through the untyped,
		// integer-governed member) rejects. The type-derivation check must resolve
		// the member's inherited effective type, not skip it on a nil raw Type.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:element name="head" type="xs:integer"/>
  <xs:element name="m1" substitutionGroup="t:head"/>
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

// TestElementRestrictsSubstMember covers the XSD 1.0 element-to-element
// substitution-group-member restriction path (restriction_particle.go
// elementRestrictsElement / derivedElementRestrictsBaseSubstMember): a derived
// element that is an instance-admissible substitution-group member of the base
// element — and validly restricts it — is accepted, but ONLY in the direct
// positional element:element mapping (recurseOrdered) and ONLY when the base
// element is at-most-once, so the group-mapping (MapAndSum) path and repeating
// base elements stay rejected.
func TestElementRestrictsSubstMember(t *testing.T) {
	t.Parallel()

	const notValidRestriction = "not a valid restriction"

	t.Run("accepts transitive subst member restricting at-most-once base element", func(t *testing.T) {
		t.Parallel()
		// W3C elemZ027_e: base sequence(element ref d); derived sequence(element ref c)
		// where c -> b -> d is a transitive substitution-group chain and d is
		// maxOccurs=1. The direct element:element mapping accepts it.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" substitutionGroup="b" type="xs:int"/>
  <xs:element name="b" substitutionGroup="c" type="xs:int"/>
  <xs:element name="c" substitutionGroup="d" type="xs:anyType"/>
  <xs:element name="d" type="xs:anyType"/>
  <xs:complexType name="base">
    <xs:sequence><xs:element ref="d"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence><xs:element ref="c"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="base"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects subst member restricting a repeating base element", func(t *testing.T) {
		t.Parallel()
		// W3C elemZ026: base sequence(element ref basicBit maxOccurs=unbounded);
		// derived sequence(element ref restrictedBasicBit maxOccurs=unbounded). The
		// base element repeats (maxOccurs>1), so the subst-member narrowing is not
		// accepted (an interval occurrence check is hole-blind).
		schema := `<schema xmlns="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <complexType name="basicBitType" abstract="true">
    <sequence><element name="testElement" type="token" maxOccurs="unbounded"/></sequence>
  </complexType>
  <complexType name="restrictedBasicBitType">
    <complexContent>
      <restriction base="t:basicBitType">
        <sequence><element name="testElement" type="token" maxOccurs="1"/></sequence>
      </restriction>
    </complexContent>
  </complexType>
  <element name="basicBit" type="t:basicBitType" abstract="true"/>
  <element name="restrictedBasicBit" type="t:restrictedBasicBitType" substitutionGroup="t:basicBit"/>
  <complexType name="basicBitContainerType">
    <sequence><element ref="t:basicBit" maxOccurs="unbounded"/></sequence>
  </complexType>
  <complexType name="restrictedBasicBitContainerType">
    <complexContent>
      <restriction base="t:basicBitContainerType">
        <sequence><element ref="t:restrictedBasicBit" maxOccurs="unbounded"/></sequence>
      </restriction>
    </complexContent>
  </complexType>
</schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects subst member buried in a sequence-restricts-choice mapping", func(t *testing.T) {
		t.Parallel()
		// W3C particlesZ028 (invalid in 1.0, valid in 1.1): base
		// sequence(group ref abs {0,unbounded}, d{0,1}) where abs is a choice of
		// abstract heads; derived sequence(sequence(a,b){0,1}, d{1,1}) where a/b are
		// substitution members. The subst-member acceptance is confined to the direct
		// element:element mapping, so it is NOT used inside the sequence:choice
		// MapAndSum here — the restriction stays 1.0-invalid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element abstract="true" name="aba" type="xs:string"/>
  <xs:element abstract="true" name="abb" type="xs:int"/>
  <xs:element abstract="true" name="abc" type="xs:date"/>
  <xs:element name="a" substitutionGroup="aba" type="xs:string"/>
  <xs:element name="b" substitutionGroup="abb" type="xs:int"/>
  <xs:element name="c" substitutionGroup="abc" type="xs:date"/>
  <xs:element name="d" type="xs:anyURI"/>
  <xs:complexType name="test">
    <xs:sequence>
      <xs:group maxOccurs="unbounded" minOccurs="0" ref="abs"/>
      <xs:element minOccurs="0" ref="d"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="test1">
    <xs:complexContent>
      <xs:restriction base="test">
        <xs:sequence>
          <xs:sequence minOccurs="0" maxOccurs="1">
            <xs:element ref="a"/>
            <xs:element ref="b"/>
          </xs:sequence>
          <xs:element ref="d"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:group name="abs">
    <xs:choice>
      <xs:element ref="aba"/>
      <xs:element ref="abb"/>
      <xs:element ref="abc"/>
    </xs:choice>
  </xs:group>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects local element sharing subst member name but widening type", func(t *testing.T) {
		t.Parallel()
		// A derived LOCAL element named m1 typed xs:string is not a valid restriction
		// of a base subst-member m1 typed xs:integer — it would admit <m1>abc</m1>,
		// which the base rejects. Matching by name alone must not accept it.
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
          <xs:sequence><xs:element name="m1" type="xs:string"/></xs:sequence>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects local element widening an untyped subst member's inherited type", func(t *testing.T) {
		t.Parallel()
		// The base subst member m1 OMITS @type, inheriting head's xs:integer. A derived
		// LOCAL element m1 typed xs:string admits <m1>abc</m1>, which the base rejects.
		// The type-derivation check must resolve the member's inherited effective type,
		// not skip it on the member's nil raw Type.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:element name="head" type="xs:integer"/>
  <xs:element name="m1" substitutionGroup="t:head"/>
  <xs:complexType name="base">
    <xs:sequence><xs:element ref="t:head"/></xs:sequence>
  </xs:complexType>
  <xs:element name="doc">
    <xs:complexType>
      <xs:complexContent>
        <xs:restriction base="t:base">
          <xs:sequence><xs:element name="m1" type="xs:string"/></xs:sequence>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts built-in-derived type restricting a subst member", func(t *testing.T) {
		t.Parallel()
		// A derived subst member m1 typed xs:int validly restricts a base subst member
		// m1 typed xs:integer — xs:int IS derived from xs:integer, but the 1.0 built-ins
		// are not BaseType-linked, so the NameAndTypeOK type check must be built-in-aware
		// (strictBuiltinAwareDerivedFrom) rather than a plain base-chain walk.
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
          <xs:sequence><xs:element name="m1" type="xs:int"/></xs:sequence>
        </xs:restriction>
      </xs:complexContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})
}
