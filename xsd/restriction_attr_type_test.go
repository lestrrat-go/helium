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

	t.Run("accepts derived member of a faceted base union (XSD 1.0)", func(t *testing.T) {
		t.Parallel()
		// Base @a is a FACETED union: FacetedUnion = (xs:int | xs:string) restricted
		// with an enumeration {1, "hello"}. The derived restriction redeclares @a as
		// bare xs:int, which is validly derived from one of the union's {member type
		// definitions}. XSD 1.0 cos-st-derived-ok (§3.14.6) has NO "facets empty"
		// condition on a union base, so this is a VALID restriction and must be
		// ACCEPTED. The "facets empty" gate is an XSD 1.1-only condition (§3.16.6.3);
		// this package targets XSD 1.0 (libxml2 parity). Regression guard documenting
		// the 1.0 behavior.
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

	t.Run("rejects atomic base redeclared as a user union", func(t *testing.T) {
		t.Parallel()
		// Base @a is the atomic builtin xs:string; the derived restriction redeclares
		// @a as a constructed user xs:union (StrOrInt = xs:string | xs:int). A
		// constructed union is not derived from xs:string through the pointer chain and
		// admits non-string values (e.g. ints), so it WIDENS the base value space. A
		// constructed list/union can only derive from xs:anySimpleType or via a real
		// base-type chain, so this must be REJECTED.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="StrOrInt">
    <xs:union memberTypes="xs:string xs:int"/>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="t:StrOrInt"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects atomic base redeclared as a user list", func(t *testing.T) {
		t.Parallel()
		// Base @a is the atomic builtin xs:string; the derived restriction redeclares
		// @a as a constructed user xs:list (StrList = list of xs:string). A constructed
		// list is not derived from xs:string through the pointer chain and admits
		// multi-token values the base does not, so it WIDENS the value space. A
		// constructed list/union can only derive from xs:anySimpleType or via a real
		// base-type chain, so this must be REJECTED.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="StrList">
    <xs:list itemType="xs:string"/>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="t:StrList"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
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

	t.Run("rejects user list base restricted by xs:anySimpleType", func(t *testing.T) {
		t.Parallel()
		// Base @a is a user LIST type (xs:list itemType="xs:int"); the derived
		// restriction redeclares @a as xs:anySimpleType. anySimpleType is the simple
		// ur-type — a SUPERTYPE of the list — so deriving the list "down to" it would
		// WIDEN to accept non-list values. A restriction can never validly produce a
		// supertype, so it must be rejected.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="t:IntList"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:anySimpleType"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("rejects builtin list base restricted by an unrelated user list", func(t *testing.T) {
		t.Parallel()
		// Base @a is xs:IDREFS (a builtin LIST type, registered as a bare name with
		// no list marker); the derived restriction redeclares @a as a user list
		// (xs:list itemType="xs:string"). The user list is not derived from xs:IDREFS
		// (it has no <xs:restriction base="xs:IDREFS"> pointer), and IDREFS is a list
		// of xs:IDREF — an unrelated list type that admits values the base does not,
		// so the restriction must be rejected rather than silently accepted because
		// the derived side has no builtin base name.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="StringList">
    <xs:list itemType="xs:string"/>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:IDREFS"/>
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

	t.Run("rejects untyped derived attr widening a narrower base attr type", func(t *testing.T) {
		t.Parallel()
		// Base @a is xs:int; the derived restriction redeclares @a with NO type
		// and NO inline <xs:simpleType>. Per XSD §3.2.2.1 an attribute with no type
		// has {type definition} = xs:anySimpleType, the simple ur-type — a
		// SUPERTYPE of xs:int. Redeclaring @a as the ur-type WIDENS the value space
		// to admit non-integers the base forbids, so it is NOT a valid restriction
		// and must be rejected. Regression guard: an absent attr type must NOT skip
		// the restriction-derivation check.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="xs:int"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), notValidRestriction)
	})

	t.Run("accepts derived member of an ALIAS over a faceted base union (XSD 1.0)", func(t *testing.T) {
		t.Parallel()
		// Base @a is AliasUnion, a no-facet <xs:restriction> ALIAS over FacetedUnion
		// (= (xs:int | xs:string) restricted with enumeration {1,"hello"}). The derived
		// restriction redeclares @a as bare xs:int, which is validly derived from one
		// of the union's {member type definitions}. XSD 1.0 cos-st-derived-ok (§3.14.6)
		// has NO "facets empty" condition on a union base — facets inherited through
		// the alias do not block member-derivation — so this is a VALID restriction and
		// must be ACCEPTED. The "facets empty" gate is XSD 1.1-only (§3.16.6.3); this
		// package targets XSD 1.0 (libxml2 parity). Regression guard documenting the
		// 1.0 behavior.
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
  <xs:simpleType name="AliasUnion">
    <xs:restriction base="t:FacetedUnion"/>
  </xs:simpleType>
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" type="t:AliasUnion"/>
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

	t.Run("accepts typed derived fixed restricting untyped base fixed (anySimpleType)", func(t *testing.T) {
		t.Parallel()
		// Base @a has NO type and NO inline <xs:simpleType>, so per XSD §3.2.2.1 its
		// {type definition} = xs:anySimpleType (the simple ur-type), and it pins
		// fixed="x". The derived restriction redeclares @a as xs:string fixed="x".
		// xs:string is validly derived from the simple ur-type (cos-st-derived-ok)
		// and the fixed LITERAL is identical, so the fixed value is preserved — a
		// VALID XSD 1.0 restriction that must be ACCEPTED. xs:anySimpleType has no
		// primitive value-space family, so the cross-member value comparison cannot
		// route through it; fixedConstraintRestricts must fall back to literal
		// equality for the ur-type instead of false-rejecting.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" fixed="x"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:string" fixed="x"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects typed derived fixed differing from untyped base fixed (anySimpleType)", func(t *testing.T) {
		t.Parallel()
		// Same shape as the accept case, but the derived fixed LITERAL ("y") differs
		// from the base fixed literal ("x"). The base ur-type fixed value is NOT
		// preserved, so the restriction must still be REJECTED — the ur-type fallback
		// is literal equality, not blanket acceptance.
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" fixed="x"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="t:Base">
        <xs:sequence/>
        <xs:attribute name="a" type="xs:string" fixed="y"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema), fixedInconsistent)
	})

	t.Run("accepts prohibited ref with differing local fixed (no au component)", func(t *testing.T) {
		t.Parallel()
		// Global t:a is fixed="1". The derived restriction marks the inherited
		// attribute use as use="prohibited" with a differing local fixed="2".
		// Per XSD 1.0 §3.2.2 a prohibited ref corresponds to NO attribute-use
		// component — the attribute is removed — so its harmless local 'fixed'
		// must NOT be compared with the referenced global declaration's 'fixed'
		// (au-props-correct.3 does not apply). This must compile clean.
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
        <xs:attribute ref="t:a" use="prohibited" fixed="2"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Empty(t, compileFatalErrors(t, schema))
	})

	t.Run("rejects prohibited ref carrying a default (default requires use=optional)", func(t *testing.T) {
		t.Parallel()
		// Distinct compile-time rule: 'default' requires use="optional", so a
		// prohibited ref carrying a 'default' is still a schema error regardless
		// of the global's value constraint. Skipping au-props-correct.3 for a
		// prohibited ref must NOT also waive this rule.
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
        <xs:attribute ref="t:a" use="prohibited" default="x"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrors(t, schema),
			"The value of the attribute 'use' must be 'optional' if the attribute 'default' is present.")
	})
}
