package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestRestrictionUnionMemberDerivation covers the version-INDEPENDENT
// cos-st-derived-ok clause 2.2.4 case: a complexContent restriction may narrow a
// base element whose type is a UNION to a MEMBER type of that union (or a type
// derived from a member). This is valid in BOTH XSD 1.0 and 1.1 (W3C msMeta
// addB150 / test93568.xsd is expected-valid with no version qualifier), and
// libxml2 accepts it in 1.0 too.
func TestRestrictionUnionMemberDerivation(t *testing.T) {
	compile := func(t *testing.T, v xsd.Version, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(v).Compile(t.Context(), doc)
	}

	// addB150: element A typed as a union(xs:decimal, xs:string) restricted to the
	// union MEMBER xs:string. Valid in both versions.
	const memberSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A" mixed="true">
    <xs:sequence>
      <xs:element name="A" type="deci-string" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:complexContent>
      <xs:restriction base="A">
        <xs:sequence>
          <xs:element name="A" type="xs:string" maxOccurs="10"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:simpleType name="deci-string">
    <xs:union memberTypes="xs:decimal xs:string"/>
  </xs:simpleType>
</xs:schema>`

	t.Run("union member restriction accepted in 1.0", func(t *testing.T) {
		t.Parallel()
		_, err := compile(t, xsd.Version10, memberSchema)
		require.NoError(t, err)
	})

	t.Run("union member restriction accepted in 1.1", func(t *testing.T) {
		t.Parallel()
		_, err := compile(t, xsd.Version11, memberSchema)
		require.NoError(t, err)
	})

	// Guard: a NON-member type must STILL be rejected in both versions. xs:boolean
	// is not a member of union(xs:decimal, xs:string) and does not derive from one,
	// so narrowing element A to xs:boolean is an invalid restriction.
	const nonMemberSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A" mixed="true">
    <xs:sequence>
      <xs:element name="A" type="deci-string" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:complexContent>
      <xs:restriction base="A">
        <xs:sequence>
          <xs:element name="A" type="xs:boolean" maxOccurs="10"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:simpleType name="deci-string">
    <xs:union memberTypes="xs:decimal xs:string"/>
  </xs:simpleType>
</xs:schema>`

	t.Run("non-member restriction rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		_, err := compile(t, xsd.Version10, nonMemberSchema)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("non-member restriction rejected in 1.1", func(t *testing.T) {
		t.Parallel()
		_, err := compile(t, xsd.Version11, nonMemberSchema)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}
