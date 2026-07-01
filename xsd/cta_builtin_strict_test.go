package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTABuiltinStrictDerivation covers round-14 findings: the built-in
// simple-type hierarchy must be applied DECISIVELY for a built-in declared type
// (no permissive simple-vs-simple fallback that false-accepts unrelated/list-vs-item
// pairs — Finding A), and the xsi:type-must-derive check must be built-in-aware so a
// narrowing to a built-in subtype is accepted while an unrelated xsi:type is still
// rejected (Finding B).
func TestVersion11CTABuiltinStrictDerivation(t *testing.T) {
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}

	// Finding A: a DIRECT (non-complex) list alternative for a declared item type is
	// rejected — xs:NMTOKENS is not derived from xs:NMTOKEN.
	t.Run("direct NMTOKENS alternative for NMTOKEN is rejected", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:NMTOKEN">
    <xs:alternative test="true()" type="xs:NMTOKENS"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// Finding A: a DIRECT unrelated atomic alternative is rejected — xs:string is not
	// derived from xs:integer (the old permissive simple-vs-simple fallback accepted it).
	t.Run("direct string alternative for integer is rejected", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="xs:string"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// Finding A guard: a DIRECT genuinely-derived atomic alternative is still accepted
	// (xs:nonNegativeInteger ⊂ xs:integer), and governs.
	t.Run("direct nonNegativeInteger alternative for integer is accepted and governs", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="xs:nonNegativeInteger"/>
  </xs:element>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		validate := func(instance string) error {
			idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, perr)
			return xsd.NewValidator(schema).Validate(t.Context(), idoc)
		}
		require.NoError(t, validate(`<e>5</e>`))
		// nonNegativeInteger governs: -5 is rejected (declared xs:integer would accept it).
		require.ErrorIs(t, validate(`<e>-5</e>`), xsd.ErrValidationFailed)
	})

	// Finding B: a CTA xs:error alternative is suppressed by an xsi:type that narrows
	// to a built-in subtype (xs:int over declared xs:integer), and validation then
	// succeeds — the built-in-aware xsi:type derivation check accepts xs:int ⊂ xs:integer.
	t.Run("xsi:type narrowing to built-in subtype suppresses CTA and validates", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type="xs:error"/>
  </xs:element>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		validate := func(instance string) error {
			idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, perr)
			return xsd.NewValidator(schema).Validate(t.Context(), idoc)
		}
		const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema"`
		// Without xsi:type, CTA selects xs:error → invalid.
		require.ErrorIs(t, validate(`<e>5</e>`), xsd.ErrValidationFailed)
		// xsi:type="xs:int" (⊂ xs:integer) suppresses CTA and validates.
		require.NoError(t, validate(`<e `+xsiNS+` xsi:type="xs:int">5</e>`))
		// xsi:type="xs:string" is NOT derived from xs:integer → still rejected.
		require.ErrorIs(t, validate(`<e `+xsiNS+` xsi:type="xs:string">5</e>`), xsd.ErrValidationFailed)
	})

	// Finding B (non-CTA): the built-in-aware xsi:type check also fixes the general
	// false-reject of xsi:type narrowing to a built-in subtype.
	t.Run("xsi:type built-in subtype accepted without CTA", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:integer"/>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		validate := func(instance string) error {
			idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, perr)
			return xsd.NewValidator(schema).Validate(t.Context(), idoc)
		}
		const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema"`
		require.NoError(t, validate(`<e `+xsiNS+` xsi:type="xs:int">5</e>`))
		require.ErrorIs(t, validate(`<e `+xsiNS+` xsi:type="xs:string">5</e>`), xsd.ErrValidationFailed)
	})
}
