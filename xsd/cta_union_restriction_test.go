package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAUnionRestrictionSubstitutability covers gauntlet finding
// XSDCTA-861-FINAL-001: a declared type that RESTRICTS a union keeps union variety
// only via its base chain (its direct Variety/MemberTypes are empty), so
// isValidlySubstitutable must resolve variety/members through the base chain.
// Alternatives derived from the union's members are then accepted, while an
// unrelated alternative is still rejected.
func TestVersion11CTAUnionRestrictionSubstitutability(t *testing.T) {
	compile := func(t *testing.T, s string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		return cerr
	}

	// Outer restricts an INLINE union(SmallInt, xs:boolean) (e.g. via a pattern
	// facet), so Outer.Variety/MemberTypes are empty — the union variety is only
	// reachable through Outer's base.
	const types = `
  <xs:simpleType name="SmallInt"><xs:restriction base="xs:integer"><xs:maxInclusive value="100"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="Outer">
    <xs:restriction>
      <xs:simpleType><xs:union memberTypes="SmallInt xs:boolean"/></xs:simpleType>
      <xs:pattern value=".*"/>
    </xs:restriction>
  </xs:simpleType>`

	t.Run("alternative derived from a union member (SmallInt) is accepted", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + types + `
  <xs:element name="e" type="Outer">
    <xs:alternative test="true()" type="SmallInt"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})

	t.Run("alternative equal to a union member (xs:boolean) is accepted", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + types + `
  <xs:element name="e" type="Outer">
    <xs:alternative test="true()" type="xs:boolean"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})

	t.Run("alternative restricting a union member is accepted", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + types + `
  <xs:simpleType name="SmallerInt"><xs:restriction base="SmallInt"><xs:maxInclusive value="50"/></xs:restriction></xs:simpleType>
  <xs:element name="e" type="Outer">
    <xs:alternative test="true()" type="SmallerInt"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})

	t.Run("unrelated alternative (xs:string) is rejected", func(t *testing.T) {
		t.Parallel()
		s := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + types + `
  <xs:element name="e" type="Outer">
    <xs:alternative test="true()" type="xs:string"/>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})
}
