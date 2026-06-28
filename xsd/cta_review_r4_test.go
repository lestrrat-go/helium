package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAReviewR4 covers the round-4 gauntlet findings: a global element
// reached through xs:anyType must still have its conditional type assignment
// applied, and a present-but-empty test="" must be a schema compilation error.
func TestVersion11CTAReviewR4(t *testing.T) {
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}
	mustCompile := func(t *testing.T, s string) *xsd.Schema {
		t.Helper()
		schema, err := compile(t, s)
		require.NoError(t, err)
		return schema
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	// XSDCTA-861-001: doc is xs:anyType, so its child <item> is a global element
	// reached through the anyType lax-assessment path. Its CTA must still select
	// PosHolder when @kind='pos' and reject a non-positive value.
	t.Run("anyType-descendant global element applies CTA", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc" type="xs:anyType"/>
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="kind" type="xs:string"/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="Holder"><xs:sequence/><xs:attribute name="kind" type="xs:string"/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="item" type="Holder">
    <xs:alternative test="@kind='pos'" type="PosHolder"/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// item reached via doc's anyType: kind='pos' selects PosHolder → -5 invalid.
		require.ErrorIs(t, validate(t, schema, `<doc><item kind="pos" value="-5"/></doc>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, schema, `<doc><item kind="pos" value="5"/></doc>`))
		// kind!=pos falls back to declared Holder (integer) → -5 valid.
		require.NoError(t, validate(t, schema, `<doc><item kind="any" value="-5"/></doc>`))
	})

	// XSDCTA-861-002: a present-but-empty test="" is not a testless default; it must
	// compile (and fail) as an invalid XPath.
	t.Run("empty test is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="" type="T"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// A genuinely absent @test (the unconditional default) must still compile.
	t.Run("absent test is the unconditional default", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/><xs:attribute name="kind" type="xs:string"/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="@kind='x'" type="T"/>
    <xs:alternative type="T"/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		require.NoError(t, validate(t, schema, `<e kind="y"/>`))
	})
}
