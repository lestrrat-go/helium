package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAReviewR5 covers the round-5 gauntlet finding: a complex type
// with <xs:simpleContent> carries ContentType == ContentTypeSimple, so the
// substitutability check must use a real simple-vs-complex discriminator and
// reject a simple alternative against such a declared type.
func TestVersion11CTAReviewR5(t *testing.T) {
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}

	// XSDCTA-861-R5-001: a NAMED complex type with simpleContent + a simple
	// alternative must be rejected (xs:string is not substitutable for a complex
	// type definition, even though both report ContentTypeSimple).
	t.Run("named complex simpleContent type with simple alternative is rejected", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Sc">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="requiredAttr" type="xs:string" use="required"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="Sc">
    <xs:alternative test="true()" type="xs:string"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// Same violation with an INLINE complex simpleContent declared type is also a
	// schema error (the inline type's kind must be tracked too).
	t.Run("inline complex simpleContent type with simple alternative is rejected", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:string">
          <xs:attribute name="requiredAttr" type="xs:string" use="required"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
    <xs:alternative test="true()" type="xs:string"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// A complex simpleContent ALTERNATIVE that is NOT derived from the simple
	// declared type is rejected (the reverse direction). Sc extends xs:string, so it
	// is unrelated to the declared xs:integer.
	t.Run("unrelated complex simpleContent alternative for simple declared type is rejected", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Sc">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="a" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="Sc"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// But a complex simpleContent alternative that genuinely EXTENDS the simple
	// declared type IS validly substitutable and must compile (guard against
	// over-rejection of the reverse direction).
	t.Run("derived complex simpleContent alternative for simple declared type is accepted", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Sc">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="a" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" type="xs:string">
    <xs:alternative test="true()" type="Sc"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.NoError(t, err)
	})

	// Guard against over-rejection: a genuine simple-to-simple derivation
	// (xs:nonNegativeInteger ⊂ xs:integer) must still compile and govern.
	t.Run("simple-to-simple derivation still accepted", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="xs:nonNegativeInteger"/>
  </xs:element>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		validate := func(instance string) error {
			doc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, perr)
			return xsd.NewValidator(schema).Validate(t.Context(), doc)
		}
		// The nonNegativeInteger alternative governs: -5 is rejected, 5 accepted.
		require.ErrorIs(t, validate(`<e>-5</e>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(`<e>5</e>`))
	})
}
