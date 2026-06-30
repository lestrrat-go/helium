package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_DropsBaseLocalBlockDefault covers the two remaining soundness cells
// of globalDropsLocalConstraint on the DROPPED-local path: {block}/{disallowed
// substitutions} loss and asymmetric {default} loss. When a dropped base local is
// re-admitted via the global, validateWildcardChild applies the GLOBAL's block (so a
// base-local block="#all" is lost) and validate.go substitutes the GLOBAL's default
// (so an empty element the base rejected becomes valid).
func TestOpenContent_DropsBaseLocalBlockDefault(t *testing.T) {
	t.Parallel()

	build := func(globalDecl, etype, localAttr string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  ` + globalDecl + `
  <xs:complexType name="ET"><xs:attribute name="id" type="xs:int"/></xs:complexType>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence>
      <xs:element name="e" type="` + etype + `" minOccurs="0"` + localAttr + `/>
      <xs:element name="keep" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="t:B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element name="keep" type="xs:string" minOccurs="0"/></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}

	t.Run("dropped local block=#all re-admitted via no-block global is rejected", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:element name="e" type="t:ET"/>`, "t:ET", ` block="#all"`))
		require.Error(t, cerr, "the global does not block the derivations/substitutions the base local blocked")
	})

	t.Run("dropped local block=#all re-admitted via block-compatible global is accepted", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:element name="e" type="t:ET" block="#all"/>`, "t:ET", ` block="#all"`))
		require.NoError(t, cerr, "the global blocks everything the base local blocked")
	})

	t.Run("dropped no-default local re-admitted via a defaulting global is rejected", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:element name="e" type="xs:int" default="5"/>`, "xs:int", ``))
		require.Error(t, cerr, "the global supplies a default the base local lacks, validating an empty element the base rejected")
	})

	t.Run("dropped local re-admitted via a global with the same default is accepted", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:element name="e" type="xs:int" default="5"/>`, "xs:int", ` default="5"`))
		require.NoError(t, cerr, "the global's default matches the base local's, so nothing is lost")
	})

	t.Run("dropped local WITH default re-admitted via a no-default global is accepted", func(t *testing.T) {
		t.Parallel()
		// a BASE-LOCAL default forbids nothing, so losing it is sound (asymmetric).
		_, _, cerr := compileV11(t, build(`<xs:element name="e" type="xs:int"/>`, "xs:int", ` default="5"`))
		require.NoError(t, cerr, "a base-local default is not a constraint")
	})
}

// TestOpenContent_KeptNarrowBlockDefault covers the same two cells on the
// KEPT-but-narrowed path (shared globalDropsLocalConstraint helper): the excess
// occurrences beyond the derived particle's maxOccurs spill into the enforcing
// interleave open content governed by the global, losing the base local's block /
// gaining the global's default.
func TestOpenContent_KeptNarrowBlockDefault(t *testing.T) {
	t.Parallel()

	build := func(globalDecl, etype, localAttr string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  ` + globalDecl + `
  <xs:complexType name="ET"><xs:attribute name="id" type="xs:int"/></xs:complexType>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence>
      <xs:element name="e" type="` + etype + `" minOccurs="0" maxOccurs="3"` + localAttr + `/>
      <xs:element name="keep" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="t:B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence>
      <xs:element name="e" type="` + etype + `" minOccurs="0" maxOccurs="1"` + localAttr + `/>
      <xs:element name="keep" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}

	t.Run("kept-narrowed local block=#all with a no-block global is rejected", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:element name="e" type="t:ET"/>`, "t:ET", ` block="#all"`))
		require.Error(t, cerr, "the spilled excess is governed by the no-block global, losing the base local's block")
	})

	t.Run("kept-narrowed no-default local with a defaulting global is rejected", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:element name="e" type="xs:int" default="5"/>`, "xs:int", ``))
		require.Error(t, cerr, "the spilled excess empty element gets the global's default the base local lacked")
	})

	t.Run("kept-narrowed local with a block-compatible global is accepted", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:element name="e" type="t:ET" block="#all"/>`, "t:ET", ` block="#all"`))
		require.NoError(t, cerr, "the global blocks everything the base local blocked")
	})
}

// TestOpenContent_KeptNarrowTypeTable covers the {type table} cell on the
// KEPT-but-narrowed path (now enforced via globalDropsLocalConstraint, shared with
// the dropped-local path): a kept local that narrows maxOccurs spills excess into the
// enforcing interleave open content governed by the global, whose CTA {type table}
// must be equivalent to the base local's.
func TestOpenContent_KeptNarrowTypeTable(t *testing.T) {
	t.Parallel()

	// base/derived local e carry a {type table} (alternative→TA, baseAlt); the global
	// e carries globalAlt. base maxOccurs=3, derived maxOccurs=1 (narrowed).
	build := func(globalAlt string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="ET"><xs:attribute name="k" type="xs:string"/></xs:complexType>
  <xs:complexType name="TA"><xs:complexContent><xs:restriction base="t:ET"><xs:attribute name="k" type="xs:string"/></xs:restriction></xs:complexContent></xs:complexType>
  <xs:complexType name="TB"><xs:complexContent><xs:restriction base="t:ET"><xs:attribute name="k" type="xs:string"/></xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="e" type="t:ET">` + globalAlt + `</xs:element>
  <xs:complexType name="B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence>
      <xs:element name="e" type="t:ET" minOccurs="0" maxOccurs="3"><xs:alternative test="@k='a'" type="t:TA"/></xs:element>
      <xs:element name="keep" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="t:B">
    <xs:openContent mode="interleave"><xs:any namespace="##targetNamespace" processContents="strict"/></xs:openContent>
    <xs:sequence>
      <xs:element name="e" type="t:ET" minOccurs="0" maxOccurs="1"><xs:alternative test="@k='a'" type="t:TA"/></xs:element>
      <xs:element name="keep" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}

	// NOTE: the kept-narrowed type-table REJECT cell is UNREACHABLE as an independent
	// false-accept — a restriction cannot change an element's CTA {type table} (so the
	// kept local's table equals the base local's), and an enforcing open content that
	// re-admits the name is already caught by the static wildcard-EDC
	// (checkWildcardElementConsistent) on both the base and derived types. The shared
	// helper enforcing {type table} is therefore defense-in-depth here; this accept case
	// confirms it does not NEWLY reject a valid (equivalent-table) kept-narrowed schema.
	t.Run("kept-narrowed local with a type-table-equivalent global is accepted", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:alternative test="@k='a'" type="t:TA"/>`))
		require.NoError(t, cerr, "the global's type table matches the base local's, so nothing is lost")
	})
}
