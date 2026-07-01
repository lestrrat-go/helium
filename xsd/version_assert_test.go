package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11Assert covers XSD 1.1 xs:assert on a complex type: the assertion
// is evaluated in 1.1, ignored in 1.0, and a malformed test expression is a
// compile error in 1.1.
func TestVersion11Assert(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="range">
    <xs:complexType>
      <xs:attribute name="min" type="xs:int"/>
      <xs:attribute name="max" type="xs:int"/>
      <xs:assert test="xs:integer(@min) le xs:integer(@max)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T, c xsd.Compiler, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return c.Compile(t.Context(), doc)
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	t.Run("1.1 assertion satisfied", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validate(t, schema, `<range min="1" max="5"/>`))
	})

	t.Run("1.1 assertion violated", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.ErrorIs(t, validate(t, schema, `<range min="5" max="1"/>`), xsd.ErrValidationFailed)
	})

	t.Run("1.0 ignores xs:assert", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler(), schemaXML)
		require.NoError(t, err)
		// The assert would fail, but 1.0 does not enforce it.
		require.NoError(t, validate(t, schema, `<range min="5" max="1"/>`))
	})

	t.Run("1.1 malformed assert XPath is a compile error", func(t *testing.T) {
		t.Parallel()
		const bad = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:assert test="@a +"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), bad)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}
