package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_DerivedWildcardReadmitsBaseOpen covers QUADRANT B of the restriction
// content-interaction matrix: a restriction that DROPS the base's open content but
// re-introduces its admitted language as a DECLARED WILDCARD must keep that declared
// wildcard a valid restriction of the base open-content wildcard (namespace ⊆,
// processContents at least as strong), otherwise the declared wildcard (which wins
// attribution) accepts children the base's open content validated more strictly.
func TestOpenContent_DerivedWildcardReadmitsBaseOpen(t *testing.T) {
	t.Parallel()

	// base B: open content (baseOC) over a declared sequence holding an optional nested
	// GROUP (NO declared wildcard); R restricts B by mapping a declared wildcard
	// (derivedWC) onto that base group (the Wildcard-restricting-ModelGroup conservatism
	// in restriction_particle.go lets it compile), with the open content derivedOC
	// ("" = none).
	build := func(baseOC, derivedWC, derivedOC string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="B">
    ` + baseOC + `
    <xs:sequence><xs:choice minOccurs="0"><xs:element name="a" type="xs:string"/></xs:choice></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="t:B">
    ` + derivedOC + `
    <xs:sequence>` + derivedWC + `</xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}

	t.Run("derived skip wildcard re-admitting strict base open content is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(
			`<xs:openContent mode="interleave"><xs:any namespace="##any" processContents="strict"/></xs:openContent>`,
			`<xs:any namespace="##any" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>`,
			``)
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "a skip declared wildcard accepts what the base strict open content rejected")
	})

	t.Run("derived strict wildcard within the base open content namespace is accepted", func(t *testing.T) {
		t.Parallel()
		schema := build(
			`<xs:openContent mode="interleave"><xs:any namespace="##any" processContents="strict"/></xs:openContent>`,
			`<xs:any namespace="##any" processContents="strict" minOccurs="0" maxOccurs="unbounded"/>`,
			``)
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "a strict wildcard ⊆ namespace with pc at least as strong is a valid restriction of the base open content")
	})

	t.Run("derived wildcard admitting a namespace the base open content excludes is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(
			`<xs:openContent mode="interleave"><xs:any namespace="urn:x" processContents="strict"/></xs:openContent>`,
			`<xs:any namespace="##any" processContents="strict" minOccurs="0" maxOccurs="unbounded"/>`,
			``)
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "##any admits namespaces the base open content (urn:x only) excluded")
	})

	t.Run("base WITHOUT open content: derived wildcard restricting a base element group is rejected", func(t *testing.T) {
		t.Parallel()
		schema := build(
			``,
			`<xs:any namespace="##any" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>`,
			``)
		_, _, cerr := compileV11(t, schema)
		// With no open content on either type, no quadrant-B guard governs this and
		// the particle-level wildcard-restricts-model-group check applies soundly: a
		// skip ##any wildcard admits names (and empty/overlong content) the base
		// element group `sequence(choice(a){0,1})` rejects, so it is NOT a language
		// subset and the restriction is invalid.
		require.Error(t, cerr, "a wildcard restricting a base element group with no open content is not a language subset")
	})
}

// TestOpenContent_DerivedWildcardReadmitsBaseOpenSuffix covers the suffix order-loss
// refinement of quadrant B: when the BASE open content is SUFFIX mode (open children
// must be trailing), a derived declared wildcard re-admitting that namespace makes the
// children DECLARED content that can appear non-trailing, so it is rejected outright —
// even a strict subset wildcard that would be valid under an INTERLEAVE base.
func TestOpenContent_DerivedWildcardReadmitsBaseOpenSuffix(t *testing.T) {
	t.Parallel()

	build := func(baseOC, derivedWC string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="B">
    ` + baseOC + `
    <xs:sequence><xs:choice minOccurs="0"><xs:element name="a" type="xs:string"/></xs:choice></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="t:B">
    <xs:sequence>` + derivedWC + `</xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}
	const suffix = `<xs:openContent mode="suffix"><xs:any namespace="##any" processContents="strict"/></xs:openContent>`
	const interleave = `<xs:openContent mode="interleave"><xs:any namespace="##any" processContents="strict"/></xs:openContent>`
	const strictWC = `<xs:any namespace="##any" processContents="strict" minOccurs="0" maxOccurs="unbounded"/>`

	t.Run("base suffix + derived re-admitting wildcard is rejected (order loss)", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(suffix, strictWC))
		require.Error(t, cerr, "a declared wildcard re-admits the suffix open content non-trailing, losing the ordering constraint")
	})

	t.Run("base interleave + same strict subset wildcard is accepted (no ordering)", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(interleave, strictWC))
		require.NoError(t, cerr, "interleave imposes no ordering, so a strict subset wildcard is a valid restriction")
	})

	t.Run("base suffix + derived has no re-admitting declared wildcard is unaffected", func(t *testing.T) {
		t.Parallel()
		// R drops the suffix open content and keeps the declared group (no wildcard) — no
		// quadrant-B interaction, so it compiles as before.
		_, _, cerr := compileV11(t, build(suffix, `<xs:choice minOccurs="0"><xs:element name="a" type="xs:string"/></xs:choice>`))
		require.NoError(t, cerr, "no derived declared wildcard re-admits the base open content")
	})
}

// TestOpenContent_QuadrantBSuffixStrictNoGlobal covers F1: the quadrant-B suffix
// order-loss reject must only fire when the derived declared wildcard ACTUALLY admits
// a child. A STRICT wildcard with no matching global element in the intersecting
// namespace admits nothing, so no ordering is lost → EXEMPT.
func TestOpenContent_QuadrantBSuffixStrictNoGlobal(t *testing.T) {
	t.Parallel()

	build := func(derivedWC string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:complexType name="B">
    <xs:openContent mode="suffix"><xs:any namespace="##any" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:choice minOccurs="0"><xs:element name="a" type="xs:string"/></xs:choice></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="t:B">
    <xs:sequence>` + derivedWC + `</xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="t:R"/>
</xs:schema>`
	}

	t.Run("strict wildcard with no matching global is exempt (accept)", func(t *testing.T) {
		t.Parallel()
		// ##other excludes the target namespace; the only global element (doc) is in the
		// target namespace, so the strict wildcard admits no child → no order loss.
		_, _, cerr := compileV11(t, build(`<xs:any namespace="##other" processContents="strict" minOccurs="0" maxOccurs="unbounded"/>`))
		require.NoError(t, cerr, "a strict wildcard with no matching global admits nothing, so no suffix order is lost")
	})

	t.Run("strict wildcard WITH a matching global is rejected (order loss)", func(t *testing.T) {
		t.Parallel()
		// ##any admits the global element doc, so the wildcard re-admits a child non-trailing.
		_, _, cerr := compileV11(t, build(`<xs:any namespace="##any" processContents="strict" minOccurs="0" maxOccurs="unbounded"/>`))
		require.Error(t, cerr, "a strict wildcard admitting a global re-admits the suffix language non-trailing")
	})

	t.Run("skip wildcard is rejected (always admits)", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:any namespace="##any" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>`))
		require.Error(t, cerr)
	})

	t.Run("lax wildcard is rejected (always admits)", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, build(`<xs:any namespace="##any" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>`))
		require.Error(t, cerr)
	})
}

// TestOpenContent_InterleavePartitionStrictWildcardFailure covers F2: a declared
// optional STRICT xs:any that greedily consumes a child and fails its validation must
// not prevent the valid partition where that child is OPEN content.
func TestOpenContent_InterleavePartitionStrictWildcardFailure(t *testing.T) {
	t.Parallel()

	schema := func(openPC string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="` + openPC + `"/></xs:openContent>
    <xs:sequence>
      <xs:any namespace="##any" processContents="strict" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
	}

	t.Run("child invalid as strict-declared but valid as lax-open is accepted", func(t *testing.T) {
		t.Parallel()
		// <unknown> has no global declaration: it fails the strict DECLARED wildcard but
		// is valid against the LAX open content (laxly assessed). The optional declared
		// wildcard is satisfied by zero matches.
		require.NoError(t, validateOC(t, schema("lax"), `<doc xmlns="urn:t"><unknown>x</unknown></doc>`),
			"the child belongs to the open partition, leaving the optional declared wildcard empty")
	})

	t.Run("child invalid as both declared and open is rejected", func(t *testing.T) {
		t.Parallel()
		// With a STRICT open content too, <unknown> (no global) is invalid as declared AND
		// as open content → must still reject (no false-accept).
		require.Error(t, validateOC(t, schema("strict"), `<doc xmlns="urn:t"><unknown>x</unknown></doc>`),
			"a child invalid as both declared and open content must reject")
	})
}
