package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileAndValidate compiles schemaXML with the given compiler and validates
// instanceXML against it, returning the validation error (or nil).
func compileAndValidateV(t *testing.T, c xsd.Compiler, schemaXML, instanceXML string) error {
	t.Helper()
	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := c.Compile(t.Context(), schemaDOC)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), doc)
}

// TestVersionToggle exercises the XSD-version selection end-to-end through the
// public Compiler API and the vc:minVersion auto-detection, using the "+INF"
// xs:double lexical form (valid only in XSD 1.1).
func TestVersionToggle(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v" type="xs:double"/>
</xs:schema>`
	const schemaVC11 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning" vc:minVersion="1.1">
  <xs:element name="v" type="xs:double"/>
</xs:schema>`
	const instancePlusINF = `<v>+INF</v>`
	const instanceINF = `<v>INF</v>`

	t.Run("default (1.0) rejects +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler(), schemaXML, instancePlusINF)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("explicit 1.0 rejects +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML, instancePlusINF)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("explicit 1.1 accepts +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML, instancePlusINF)
		require.NoError(t, err)
	})

	t.Run("vc:minVersion=1.1 auto-detects 1.1 and accepts +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler(), schemaVC11, instancePlusINF)
		require.NoError(t, err)
	})

	t.Run("explicit 1.0 overrides vc:minVersion=1.1", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaVC11, instancePlusINF)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("plain INF accepted in both versions", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), schemaXML, instanceINF))
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML, instanceINF))
	})
}
