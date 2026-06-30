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

	t.Run("base WITHOUT open content is unaffected (derived declared wildcard compiles)", func(t *testing.T) {
		t.Parallel()
		schema := build(
			``,
			`<xs:any namespace="##any" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>`,
			``)
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "no base open content means no quadrant-B interaction")
	})
}
