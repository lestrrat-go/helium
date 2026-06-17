package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestXsiNilNoTypeElement checks that an element declaration with NO explicit
// type (which XSD defaults to xs:anyType) still runs xsi:nil lexical validation
// and nilled-empty enforcement. Previously these checks sat behind a nil-type
// early return, so a no-type nillable declaration accepted invalid xsi:nil
// lexicals and non-empty nilled content.
func TestXsiNilNoTypeElement(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" nillable="true"/>
</xs:schema>`

	const xsiDecl = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	t.Run("nil=true empty is accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="true"/>`, nil))
	})

	t.Run("nil=true with child content is rejected", func(t *testing.T) {
		t.Parallel()
		var out string
		err := compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="true"><child/></root>`, &out)
		require.Error(t, err)
		require.Contains(t, out, "nilled")
	})

	t.Run("nil=maybe is a lexical error", func(t *testing.T) {
		t.Parallel()
		var out string
		err := compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="maybe"/>`, &out)
		require.Error(t, err)
		require.NotContains(t, out, "nilled")
		require.Contains(t, out, "not a valid value")
	})
}

// TestSchemaNillableLexical checks that the schema-side nillable attribute is
// parsed as an xs:boolean (after whitespace collapse): nillable="1" means true,
// and an invalid lexical produces a schema parser error.
func TestSchemaNillableLexical(t *testing.T) {
	t.Parallel()

	const xsiDecl = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	t.Run("nillable=1 is honored as true", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" nillable="1"/>
</xs:schema>`
		// nilled element with empty content must be accepted (nillable honored).
		require.NoError(t, compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="true"/>`, nil))
	})

	t.Run("invalid nillable lexical is a schema error", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" nillable="maybe"/>
</xs:schema>`
		_, errs := compileWithErrors(t, schemaXML)
		require.NotEmpty(t, errs)
		require.Contains(t, errs, "not a valid value")
	})
}
