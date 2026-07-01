package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestXsiTypeBuiltinNarrowingBlock covers gauntlet finding XSDCTA-861-REVIEW-001:
// xsi:type narrowing to a built-in subtype is subject to the element declaration's
// block flags. All built-in simple-type derivation is by RESTRICTION, so
// block="restriction" / block="#all" must reject e.g. xsi:type="xs:int" over a
// declared xs:integer (the built-in base chain is not pointer-linked, so the block
// check must consult the built-in hierarchy).
func TestXsiTypeBuiltinNarrowingBlock(t *testing.T) {
	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema"`

	compileValidate := func(t *testing.T, block, instance string) error {
		t.Helper()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:integer" ` + block + `/>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		sc, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(sc).Validate(t.Context(), idoc)
	}

	intInst := `<e ` + xsiNS + ` xsi:type="xs:int">5</e>`

	t.Run("block=restriction rejects built-in restriction narrowing", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileValidate(t, `block="restriction"`, intInst), xsd.ErrValidationFailed)
	})

	t.Run("block=#all rejects built-in restriction narrowing", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileValidate(t, `block="#all"`, intInst), xsd.ErrValidationFailed)
	})

	t.Run("no block accepts built-in restriction narrowing", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileValidate(t, ``, intInst))
	})

	t.Run("block=extension accepts built-in restriction narrowing", func(t *testing.T) {
		t.Parallel()
		// xs:int is derived from xs:integer by RESTRICTION, so an extension-only block
		// does not apply.
		require.NoError(t, compileValidate(t, `block="extension"`, intInst))
	})

	// Also exercise a NESTED (non-root) element, whose per-child match site applies
	// the same block check.
	t.Run("block=restriction rejects narrowing on a nested element", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">
    <xs:complexType><xs:sequence>
      <xs:element name="e" type="xs:integer" block="restriction"/>
    </xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		sc, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><e `+xsiNS+` xsi:type="xs:int">5</e></r>`))
		require.NoError(t, err)
		require.ErrorIs(t, xsd.NewValidator(sc).Validate(t.Context(), idoc), xsd.ErrValidationFailed)
	})
}
