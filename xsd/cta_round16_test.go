package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestRound16SubstAndBlock covers the round-16 gauntlet findings: block
// enforcement at the xs:all / strict-wildcard / anyType-descendant governing
// sites (BLOCK-002); block on a BaseType-linked 1.1 built-in narrowing
// (REVIEW-001); a user union rejecting an unrelated simple alternative
// (REVIEW-002); and xs:anySimpleType accepting a user list/union-derived complex
// simpleContent alternative (REVIEW-003).
func TestRound16SubstAndBlock(t *testing.T) {
	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema"`

	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	// BLOCK-002: block enforced for an xs:all child.
	t.Run("block enforced on xs:all child", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">
    <xs:complexType><xs:all>
      <xs:element name="e" type="xs:integer" block="restriction"/>
    </xs:all></xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		require.ErrorIs(t, validate(t, schema, `<r><e `+xsiNS+` xsi:type="xs:int">5</e></r>`), xsd.ErrValidationFailed)
	})

	// BLOCK-002: block enforced for a strict-wildcard-matched global element.
	t.Run("block enforced on strict wildcard-matched global element", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="g" type="xs:integer" block="restriction"/>
  <xs:element name="r">
    <xs:complexType><xs:sequence>
      <xs:any processContents="strict"/>
    </xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		require.ErrorIs(t, validate(t, schema, `<r><g `+xsiNS+` xsi:type="xs:int">5</g></r>`), xsd.ErrValidationFailed)
	})

	// BLOCK-002: block enforced for a global element assessed through xs:anyType.
	t.Run("block enforced on anyType-descendant global element", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc" type="xs:anyType"/>
  <xs:element name="g" type="xs:integer" block="restriction"/>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		require.ErrorIs(t, validate(t, schema, `<doc><g `+xsiNS+` xsi:type="xs:int">5</g></doc>`), xsd.ErrValidationFailed)
	})

	// REVIEW-001: block on a BaseType-linked 1.1 built-in narrowing
	// (xs:dateTimeStamp ⊂ xs:dateTime, both linked, derivation None before the fix).
	t.Run("block=restriction rejects dateTimeStamp over dateTime", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:dateTime" block="restriction"/>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		inst := `<e ` + xsiNS + ` xsi:type="xs:dateTimeStamp">2020-01-01T00:00:00Z</e>`
		require.ErrorIs(t, validate(t, schema, inst), xsd.ErrValidationFailed)
		// Without block it is accepted (dateTimeStamp is a valid narrowing).
		const sNoBlock = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:dateTime"/>
</xs:schema>`
		schemaNB, err := compile(t, sNoBlock)
		require.NoError(t, err)
		require.NoError(t, validate(t, schemaNB, inst))
	})

	// REVIEW-002: a user union rejects an unrelated simple alternative.
	t.Run("user union rejects unrelated simple alternative", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="NumOrBool"><xs:union memberTypes="xs:integer xs:boolean"/></xs:simpleType>
  <xs:element name="e" type="NumOrBool">
    <xs:alternative test="true()" type="xs:string"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("user union accepts member-derived alternative", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="NumOrBool"><xs:union memberTypes="xs:integer xs:boolean"/></xs:simpleType>
  <xs:element name="e" type="NumOrBool">
    <xs:alternative test="true()" type="xs:nonNegativeInteger"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.NoError(t, err)
	})

	// REVIEW-003: xs:anySimpleType accepts a complex simpleContent alternative whose
	// content is a USER list type (whose BaseType is not pointer-linked to
	// xs:anySimpleType).
	t.Run("anySimpleType accepts complex simpleContent over user list type", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="Names"><xs:list itemType="xs:Name"/></xs:simpleType>
  <xs:complexType name="NamesWithAttr">
    <xs:simpleContent>
      <xs:extension base="Names"><xs:attribute name="tag" type="xs:string"/></xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="xs:anySimpleType">
    <xs:alternative test="true()" type="NamesWithAttr"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.NoError(t, err)
	})

	// REVIEW-003 guard: xs:anySimpleType still REJECTS an element-only complex
	// alternative (not derived from anySimpleType — it derives from xs:anyType).
	t.Run("anySimpleType rejects element-only complex alternative", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Box"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:element name="e" type="xs:anySimpleType">
    <xs:alternative test="true()" type="Box"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}
