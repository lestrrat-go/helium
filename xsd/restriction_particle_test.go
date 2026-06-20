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

	t.Run("accepts reordered choice", func(t *testing.T) {
		t.Parallel()
		// Order is not significant in a choice, so restricting choice a|b to b|a is
		// a valid restriction (the order-preserving recurse must not fire here).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:choice>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string"/>
    </xs:choice>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:choice>
          <xs:element name="b" type="xs:string"/>
          <xs:element name="a" type="xs:string"/>
        </xs:choice>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts choice dropping an alternative", func(t *testing.T) {
		t.Parallel()
		// Restricting choice a|b down to just a (dropping a base alternative) is a
		// valid restriction. This mirrors the gedSchema IndivNameVariationType case.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:choice minOccurs="0" maxOccurs="unbounded">
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string"/>
    </xs:choice>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:choice minOccurs="0" maxOccurs="unbounded">
          <xs:element name="a" type="xs:string"/>
        </xs:choice>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects choice adding an alternative", func(t *testing.T) {
		t.Parallel()
		// A derived choice alternative with no counterpart in the base choice
		// admits content the base does not — not a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:choice>
      <xs:element name="a" type="xs:string"/>
    </xs:choice>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:choice>
          <xs:element name="a" type="xs:string"/>
          <xs:element name="c" type="xs:string"/>
        </xs:choice>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects empty restriction of required base", func(t *testing.T) {
		t.Parallel()
		// Base requires element a (minOccurs 1); restricting to empty content
		// (no model group) is not valid because the base is not emptiable.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base"/>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts empty restriction of emptiable base", func(t *testing.T) {
		t.Parallel()
		// Base content is fully optional, so restricting to empty content is valid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base"/>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts wildcard processContents narrowing", func(t *testing.T) {
		t.Parallel()
		// Base wildcard is lax; derived tightens to strict — a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="lax"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:any namespace="##any" processContents="strict"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects wildcard processContents weakening", func(t *testing.T) {
		t.Parallel()
		// Base wildcard is strict; derived weakens to skip — not a valid
		// restriction (a restriction may tighten but never weaken validation).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="strict"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:any namespace="##any" processContents="skip"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	// Recurse-As-If-Group (derived element restricts a base nested model group).
	// The derived element is treated as a singleton group mapped through the base
	// group's children: for a base sequence/all every UNMATCHED base child must
	// be emptiable; for a base choice the element must restrict SOME alternative.
	// The base nested group is reached during the order-preserving recurse over
	// the outer sequence.
	t.Run("rejects element restricting nested sequence leaving required base child", func(t *testing.T) {
		t.Parallel()
		// Base outer sequence holds a nested sequence (a,b both required). The
		// derived outer sequence maps a single element a onto that nested group;
		// the required base child b is unmatched, so it drops required content.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:sequence>
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
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects element restricting nested sequence with no matching base child", func(t *testing.T) {
		t.Parallel()
		// Base nested sequence (a, b?); derived maps element c onto it — c matches
		// no base child, so it is an added/renamed particle.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string" minOccurs="0"/>
      </xs:sequence>
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

	t.Run("accepts element restricting nested sequence with emptiable remainder", func(t *testing.T) {
		t.Parallel()
		// Base nested sequence (a, b?); derived maps element a onto it. The
		// unmatched base child b is optional (emptiable) — a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string" minOccurs="0"/>
      </xs:sequence>
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

	t.Run("accepts element restricting nested choice alternative", func(t *testing.T) {
		t.Parallel()
		// Base nested choice (a|b); derived maps element a onto it — restricting
		// one alternative of the base choice is valid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:choice>
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
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

	t.Run("rejects element restricting nested choice with no matching alternative", func(t *testing.T) {
		t.Parallel()
		// Base nested choice (a|b); derived maps element c onto it — c matches no
		// alternative, so it admits content the base choice does not.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:choice>
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
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

	// NSRecurseCheckCardinality (derived model GROUP restricts a base wildcard
	// particle). The derived group replaces the base <xs:any>; every
	// element/wildcard LEAF inside the derived group must be admitted by the base
	// wildcard's namespace constraint, and the group's effective occurrence range
	// must be within the base wildcard's range.
	t.Run("rejects group restricting wildcard with out-of-namespace element", func(t *testing.T) {
		t.Parallel()
		// Base outer sequence holds a wildcard restricted to ##other (urn:t
		// excluded). The derived group (replacing the wildcard) contains an element
		// in urn:t, which the base wildcard does not admit — not a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##other" processContents="lax" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="g" type="xs:string"/>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:sequence>
            <xs:element ref="t:g"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts group within base wildcard namespace", func(t *testing.T) {
		t.Parallel()
		// Base wildcard admits ##any unbounded; the derived group (replacing the
		// wildcard) holds an in-range element within cardinality — a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="lax" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="g" type="xs:string"/>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:sequence>
            <xs:element ref="t:g"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// MapAndSum (derived SEQUENCE restricts a base CHOICE). Every derived sequence
	// member must validly restrict SOME base choice branch AND the derived
	// sequence's total element-emission range must be within the base choice
	// particle's occurrence range — a single-item-max choice cannot be restricted
	// by a multi-item sequence.
	t.Run("rejects sequence restricting single-item choice with extra element", func(t *testing.T) {
		t.Parallel()
		// Base choice (a) matches at most one element; derived sequence(a,b) admits
		// two elements the base choice rejects — not a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:choice>
      <xs:element name="a" type="xs:string"/>
    </xs:choice>
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

	t.Run("rejects sequence restricting choice with unmatched member", func(t *testing.T) {
		t.Parallel()
		// Base choice (a|b) allows two items; derived sequence(a,c) is within the
		// cardinality but member c matches no base branch — not a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:choice minOccurs="0" maxOccurs="2">
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string"/>
    </xs:choice>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
          <xs:element name="c" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts sequence restricting fixed-count choice", func(t *testing.T) {
		t.Parallel()
		// Base choice {2,2} accepts exactly two elements (each matching a branch);
		// derived sequence(a,b) emits exactly two elements, each restricting a base
		// branch — a valid restriction. The check must compare the derived sequence's
		// TOTAL element-emission range (2,2) against the base choice's range (2,2), not
		// the derived group's raw occurrence range (1,1).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:choice minOccurs="2" maxOccurs="2">
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string"/>
    </xs:choice>
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

	t.Run("accepts sequence restricting multi-item choice", func(t *testing.T) {
		t.Parallel()
		// Base choice (a|b) with maxOccurs 2 matches up to two items; derived
		// sequence(a) emits one item that restricts the a branch — within cardinality
		// and every member maps onto a base branch, a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:choice minOccurs="0" maxOccurs="2">
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string"/>
    </xs:choice>
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

	// A derived model GROUP restricting a base single ELEMENT (reached through the
	// order-preserving recurse over an outer sequence). A base element accepts one
	// element; the group must be a pointless wrapper emitting exactly that element.
	t.Run("rejects group restricting element with extra member", func(t *testing.T) {
		t.Parallel()
		// Base outer sequence holds element a; derived maps a nested sequence(a,b)
		// onto it. The nested group emits two elements where the base element accepts
		// one — admits content the base does not.
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
          <xs:sequence>
            <xs:element name="a" type="xs:string"/>
            <xs:element name="b" type="xs:string"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts group pointless-wrapping a single element restriction", func(t *testing.T) {
		t.Parallel()
		// Base outer sequence holds element a; derived maps a nested sequence(a) onto
		// it. The nested group emits exactly the matching element — a pointless
		// wrapper, a valid restriction.
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
          <xs:sequence>
            <xs:element name="a" type="xs:string"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// NSRecurseCheckCardinality total-emission bound: a base <xs:any maxOccurs="1">
	// matches at most ONE element, so a derived group that can emit two-or-more
	// elements must be rejected even though each leaf individually fits the
	// wildcard's namespace and per-leaf cardinality.
	t.Run("rejects group restricting single-occurrence wildcard with two elements", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="lax" minOccurs="0" maxOccurs="1"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="g1" type="xs:string"/>
  <xs:element name="g2" type="xs:string"/>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:sequence>
            <xs:element ref="t:g1"/>
            <xs:element ref="t:g2"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts sequence restricting fixed-count wildcard", func(t *testing.T) {
		t.Parallel()
		// Base <xs:any minOccurs="2" maxOccurs="2"> accepts exactly two elements; the
		// derived two-element sequence emits exactly two in-namespace elements. The
		// check must compare the derived sequence's TOTAL element-emission range (2,2)
		// against the base wildcard's range (2,2), not the derived group's raw
		// occurrence range (1,1).
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="lax" minOccurs="2" maxOccurs="2"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="g1" type="xs:string"/>
  <xs:element name="g2" type="xs:string"/>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:sequence>
            <xs:element ref="t:g1"/>
            <xs:element ref="t:g2"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts group restricting two-occurrence wildcard with two elements", func(t *testing.T) {
		t.Parallel()
		// Base <xs:any maxOccurs="2"> matches up to two elements; the derived group
		// emits two in-namespace elements — within the wildcard's cardinality, a
		// valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="lax" minOccurs="0" maxOccurs="2"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="g1" type="xs:string"/>
  <xs:element name="g2" type="xs:string"/>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:sequence>
            <xs:element ref="t:g1"/>
            <xs:element ref="t:g2"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// Element-restricts-wildcard (NSCompat): a base wildcard restricted by a
	// derived element whose namespace the wildcard admits, within occurrence range,
	// stays a valid restriction (kept ACCEPTED — the cardinality fix above must not
	// reintroduce a false-accept nor a false-reject here).
	t.Run("accepts element restricting wildcard within namespace", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="lax" minOccurs="0" maxOccurs="1"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="g" type="xs:string"/>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:element ref="t:g"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects group restricting wildcard widening cardinality", func(t *testing.T) {
		t.Parallel()
		// Base wildcard admits ##any at most once; the derived group (replacing the
		// wildcard) is itself unbounded — its effective occurrence range exceeds
		// the base wildcard's, not a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:any namespace="##any" processContents="lax" minOccurs="0" maxOccurs="1"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="g" type="xs:string"/>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence>
          <xs:sequence maxOccurs="unbounded">
            <xs:element ref="t:g"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	// Repeated nested group: a valid restriction whose content is a repeated
	// nested group (the nested group's own occurrence range times its children's
	// emission) must be accepted. The recursion must descend into the nested
	// group rather than mis-folding its occurrence range.
	t.Run("accepts repeated nested group equivalent restriction", func(t *testing.T) {
		t.Parallel()
		// Base outer sequence holds a nested sequence(a,b) repeated exactly twice;
		// derived restriction repeats the same nested sequence twice (identical) —
		// every emitted sequence is valid against the base, a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:sequence minOccurs="2" maxOccurs="2">
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:sequence minOccurs="2" maxOccurs="2">
            <xs:element name="a" type="xs:string"/>
            <xs:element name="b" type="xs:string"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts repeated nested group narrowed restriction", func(t *testing.T) {
		t.Parallel()
		// Base nested sequence(a, b?) repeated 1..3; derived narrows the repetition
		// to 1..2 and drops the optional b — every derived emission is valid against
		// the base, a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:sequence minOccurs="1" maxOccurs="3">
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string" minOccurs="0"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:sequence minOccurs="1" maxOccurs="2">
            <xs:element name="a" type="xs:string"/>
          </xs:sequence>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// Prohibited (maxOccurs="0") leaves emit nothing: they neither require a base
	// counterpart nor block subsumption of the rest of the model.
	t.Run("accepts derived omitting a prohibited base leaf", func(t *testing.T) {
		t.Parallel()
		// Base sequence(a, b{0,0}) — b is prohibited and emits nothing; derived
		// drops it entirely. The prohibited leaf must contribute (0,0) and be
		// skipped, so the derivation is valid.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="b" type="xs:string" minOccurs="0" maxOccurs="0"/>
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

	t.Run("accepts derived prohibiting a leaf the base allows", func(t *testing.T) {
		t.Parallel()
		// Base sequence(a, b?); derived prohibits b (maxOccurs="0"). The derived
		// prohibited leaf emits nothing — it demands nothing of the base and is a
		// valid (narrowing) restriction of the optional base b.
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
          <xs:element name="b" type="xs:string" minOccurs="0" maxOccurs="0"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("accepts derived prohibited leaf with no base counterpart", func(t *testing.T) {
		t.Parallel()
		// Base sequence(a); derived keeps a and adds a prohibited leaf z
		// (maxOccurs="0") that has no counterpart in the base. The prohibited leaf
		// emits nothing, so it must be skipped during subsumption rather than
		// treated as an added particle.
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
          <xs:element name="z" type="xs:string" minOccurs="0" maxOccurs="0"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	// Value-space-equivalent fixed values: a derived fixed value that is
	// value-space-equal but lexically different from the base fixed value is a
	// valid restriction; a value-space-different fixed value is not.
	t.Run("accepts value-space-equal fixed restriction", func(t *testing.T) {
		t.Parallel()
		// Base element a fixed="1" (xs:integer); derived fixed="01" — value-space
		// equal, a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:integer" fixed="1"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="a" type="xs:integer" fixed="01"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects value-space-different fixed restriction", func(t *testing.T) {
		t.Parallel()
		// Base element a fixed="1" (xs:integer); derived fixed="2" — a different
		// value, not a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="a" type="xs:integer" fixed="1"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence>
          <xs:element name="a" type="xs:integer" fixed="2"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})
}
