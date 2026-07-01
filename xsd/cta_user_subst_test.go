package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAUserSubstitutability covers the definitive substitutability fix:
// the permissive "any two simple types" user fallback is removed, so an alternative
// must be genuinely derived from the declared (user) type — directly or, for a
// union, from a member.
func TestVersion11CTAUserSubstitutability(t *testing.T) {
	compile := func(t *testing.T, s string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		return cerr
	}

	// A user simple type restricted from a built-in.
	const smallInt = `<xs:simpleType name="SmallInt"><xs:restriction base="xs:integer"><xs:maxInclusive value="100"/></xs:restriction></xs:simpleType>`

	t.Run("user SmallInt declared + unrelated xs:string alt is rejected", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + smallInt + `
  <xs:element name="e" type="SmallInt">
    <xs:alternative test="true()" type="xs:string"/>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("union(SmallInt, boolean) declared + unrelated xs:string alt is rejected", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + smallInt + `
  <xs:simpleType name="SmallOrBool"><xs:union memberTypes="SmallInt xs:boolean"/></xs:simpleType>
  <xs:element name="e" type="SmallOrBool">
    <xs:alternative test="true()" type="xs:string"/>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("user SmallInt declared + valid restriction of SmallInt is accepted", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + smallInt + `
  <xs:simpleType name="SmallerInt"><xs:restriction base="SmallInt"><xs:maxInclusive value="50"/></xs:restriction></xs:simpleType>
  <xs:element name="e" type="SmallInt">
    <xs:alternative test="true()" type="SmallerInt"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})

	t.Run("union with a user member + alt derived from that member is accepted", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + smallInt + `
  <xs:simpleType name="SmallerInt"><xs:restriction base="SmallInt"><xs:maxInclusive value="50"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="SmallOrBool"><xs:union memberTypes="SmallInt xs:boolean"/></xs:simpleType>
  <xs:element name="e" type="SmallOrBool">
    <xs:alternative test="true()" type="SmallerInt"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})

	// A nested union member must also be reached recursively (the union recursion
	// uses the strict predicate, not the removed fallback).
	t.Run("nested union member derivation is accepted", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + smallInt + `
  <xs:simpleType name="SmallerInt"><xs:restriction base="SmallInt"><xs:maxInclusive value="50"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="Inner"><xs:union memberTypes="SmallInt xs:boolean"/></xs:simpleType>
  <xs:simpleType name="Outer"><xs:union memberTypes="Inner xs:date"/></xs:simpleType>
  <xs:element name="e" type="Outer">
    <xs:alternative test="true()" type="SmallerInt"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})

	t.Run("nested union with unrelated alt is rejected", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  ` + smallInt + `
  <xs:simpleType name="Inner"><xs:union memberTypes="SmallInt xs:boolean"/></xs:simpleType>
  <xs:simpleType name="Outer"><xs:union memberTypes="Inner xs:date"/></xs:simpleType>
  <xs:element name="e" type="Outer">
    <xs:alternative test="true()" type="xs:string"/>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})
}
