package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestXsiTypeAnyTypeBlockRestriction covers cvc-elt.4.3 / Type Derivation OK for a
// typeless (xs:anyType-typed) element whose block forbids restriction. EVERY simple
// type derives from xs:anyType through xs:anySimpleType by RESTRICTION, so a
// block="restriction"/"#all" must reject a simple xsi:type (union or list) on such
// an element. Mirrors W3C msMeta/Element_w3c elemT026-029 (restriction) and
// elemT054-057 (#all).
func TestXsiTypeAnyTypeBlockRestriction(t *testing.T) {
	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema"`

	compileValidate := func(t *testing.T, block, instance string) error {
		t.Helper()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" ` + block + `/>
  <xs:simpleType name="u">
    <xs:union memberTypes="xs:integer xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="l">
    <xs:list itemType="xs:integer"/>
  </xs:simpleType>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		sc, err := xsd.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(sc).Validate(t.Context(), idoc)
	}

	unionInst := `<e ` + xsiNS + ` xsi:type="u">5</e>`
	listInst := `<e ` + xsiNS + ` xsi:type="l">1 2 3</e>`

	t.Run("block=restriction rejects a union xsi:type", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileValidate(t, `block="restriction"`, unionInst), xsd.ErrValidationFailed)
	})

	t.Run("block=restriction rejects a list xsi:type", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileValidate(t, `block="restriction"`, listInst), xsd.ErrValidationFailed)
	})

	t.Run("block=#all rejects a union xsi:type", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileValidate(t, `block="#all"`, unionInst), xsd.ErrValidationFailed)
	})

	t.Run("block=#all rejects a list xsi:type", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileValidate(t, `block="#all"`, listInst), xsd.ErrValidationFailed)
	})

	t.Run("no block accepts a simple xsi:type", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileValidate(t, ``, unionInst))
	})

	t.Run("block=extension accepts a simple xsi:type", func(t *testing.T) {
		t.Parallel()
		// A simple type reaches xs:anyType only by restriction, so an extension-only
		// block does not apply.
		require.NoError(t, compileValidate(t, `block="extension"`, listInst))
	})
}
