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

	t.Run("rejects ref with local default against inherited base fixed", func(t *testing.T) {
		t.Parallel()
		// Global @t:a is fixed="1". The base type references it (inheriting the
		// fixed constraint); the derived restriction references it again but with a
		// LOCAL default="2". A 'default' does NOT satisfy a base 'fixed' constraint
		// (it would admit values other than the fixed one), and the use's local
		// default must not silently absorb the declaration's inherited fixed value.
		// Must be rejected.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attribute name="a" type="xs:int" fixed="1"/>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute ref="t:a"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute ref="t:a" default="2"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), fixedInconsistent)
	})

	t.Run("accepts ref with matching local fixed against inherited base fixed", func(t *testing.T) {
		t.Parallel()
		// Same setup, but the derived restriction references @t:a with a LOCAL
		// fixed="1" matching the inherited base fixed value — a valid restriction.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attribute name="a" type="xs:int" fixed="1"/>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute ref="t:a"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute ref="t:a" fixed="1"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects derived attr type unrelated to base union members", func(t *testing.T) {
		t.Parallel()
		// Base @a is a USER union (IntOrBool = xs:int | xs:boolean) with no facets.
		// The derived restriction redeclares @a as xs:date — NOT a member of the
		// union and not derived from either member, so it admits values (dates) the
		// base does not. Per cos-st-derived-ok.2.2.4 it must be REJECTED. Regression
		// guard: builtinBaseLocal(union) is empty, so the union rule must run BEFORE
		// the builtin-base early return that would otherwise accept it.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="IntOrBool">
    <xs:union memberTypes="xs:int xs:boolean"/>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="t:IntOrBool"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:date"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts derived member of a no-facet base union", func(t *testing.T) {
		t.Parallel()
		// Base @a is a NO-FACET union: IntOrString (xs:int | xs:string). The derived
		// @a (xs:int) is validly derived from one of the union's {member type
		// definitions}. Per cos-st-derived-ok.2.2.4 a facet-free union base admits a
		// derived type validly derived from ANY of its member types — so this must be
		// ACCEPTED. Regression guard against over-correcting into a union
		// false-reject.
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

	t.Run("accepts derived member of a faceted base union", func(t *testing.T) {
		t.Parallel()
		// Base @a is a FACETED union: FacetedUnion = (xs:int | xs:string) restricted
		// with an enumeration {1, "hello"}. The derived @a (xs:int) is validly derived
		// from one of the union's {member type definitions}. Per W3C cos-st-derived-ok
		// (Simple) clause 2.2.4 the derivation relation holds when D is validly derived
		// from one of B's member types; it does NOT add a "base union has no facets"
		// condition — facet validity is a separate construction/validation constraint,
		// not part of this derivation relation. So this must be ACCEPTED. Regression
		// guard against re-adding a facet gate to the derivation check.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="IntOrString">
    <xs:union memberTypes="xs:int xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="FacetedUnion">
    <xs:restriction base="t:IntOrString">
      <xs:enumeration value="1"/>
      <xs:enumeration value="hello"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="t:FacetedUnion"/>
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

	t.Run("rejects unrelated user-defined list types", func(t *testing.T) {
		t.Parallel()
		// Base @a has type IntList (xs:list itemType="xs:int"); the derived restriction
		// redeclares @a as StringList (xs:list itemType="xs:string"). Neither list is
		// the same as nor derived from the other, so StringList admits values (token
		// lists of strings) the base does not. Both have empty builtinBaseLocal, so the
		// builtin-base shortcut would wrongly accept it; the list-base branch (cos-st-
		// derived-ok.2.2) must REJECT, since the only valid derivation from a list base
		// not caught by the pointer chain is from xs:anySimpleType.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="StringList">
    <xs:list itemType="xs:string"/>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="t:IntList"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="t:StringList"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects non-restriction ref with local default vs global fixed", func(t *testing.T) {
		t.Parallel()
		// Global @t:a is fixed="1". A PLAIN (non-restriction) complexType references
		// it with a LOCAL default="2". au-props-correct.3: a 'default' does not
		// satisfy the declaration's 'fixed' constraint, so it must be REJECTED even
		// though no restriction derivation is involved.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attribute name="a" type="xs:int" fixed="1"/>
  <xs:complexType name="HasRef">
    <xs:sequence/>
    <xs:attribute ref="t:a" default="2"/>
  </xs:complexType>
  <xs:element name="root" type="t:HasRef"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), fixedInconsistent)
	})

	t.Run("rejects non-restriction ref with mismatched local fixed vs global fixed", func(t *testing.T) {
		t.Parallel()
		// Global @t:a is fixed="1"; a plain complexType references it with a LOCAL
		// fixed="2" — a different value-space value. au-props-correct.3 requires the
		// use's fixed to equal the declaration's, so it must be REJECTED.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attribute name="a" type="xs:int" fixed="1"/>
  <xs:complexType name="HasRef">
    <xs:sequence/>
    <xs:attribute ref="t:a" fixed="2"/>
  </xs:complexType>
  <xs:element name="root" type="t:HasRef"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), fixedInconsistent)
	})

	t.Run("accepts non-restriction ref with matching local fixed vs global fixed", func(t *testing.T) {
		t.Parallel()
		// Global @t:a is fixed="1"; a plain complexType references it with a LOCAL
		// fixed="01" — lexically different but value-space-equal for xs:int. The use
		// satisfies au-props-correct.3, so it must be ACCEPTED.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:attribute name="a" type="xs:int" fixed="1"/>
  <xs:complexType name="HasRef">
    <xs:sequence/>
    <xs:attribute ref="t:a" fixed="01"/>
  </xs:complexType>
  <xs:element name="root" type="t:HasRef"/>
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
