package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestXsiNilLexical checks that the value of xsi:nil is parsed as an xs:boolean
// (after whitespace collapse). Only "true"/"1" mean nilled; "false"/"0" mean
// not-nilled; any other lexical (e.g. "maybe") is a lexical error rather than
// being silently treated as if the attribute were absent.
func TestXsiNilLexical(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string" nillable="true"/>
</xs:schema>`

	const xsiDecl = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	t.Run("nil=true is nilled (no content allowed)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="true"/>`, nil))
		// Content is rejected because the element is nilled.
		var out string
		err := compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="true">x</root>`, &out)
		require.Error(t, err)
		require.Contains(t, out, "nilled")
	})

	t.Run("nil=1 is nilled (no content allowed)", func(t *testing.T) {
		t.Parallel()
		var out string
		err := compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="1">x</root>`, &out)
		require.Error(t, err)
		require.Contains(t, out, "nilled")
	})

	t.Run("nil=false is not nilled (content allowed)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="false">x</root>`, nil))
	})

	t.Run("nil=0 is not nilled (content allowed)", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="0">x</root>`, nil))
	})

	t.Run("nil=true with surrounding whitespace is collapsed", func(t *testing.T) {
		t.Parallel()
		var out string
		err := compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="  true  ">x</root>`, &out)
		require.Error(t, err)
		require.Contains(t, out, "nilled")
	})

	t.Run("nil=maybe is a lexical error", func(t *testing.T) {
		t.Parallel()
		var out string
		err := compileAndValidate(t, schemaXML,
			`<root `+xsiDecl+` xsi:nil="maybe">x</root>`, &out)
		require.Error(t, err)
		require.NotContains(t, out, "nilled")
		require.Contains(t, out, "not a valid value")
	})
}
