package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_ModeWhitespace covers the parser-normalization finding: @mode is an
// enumeration over xs:token (whiteSpace="collapse"), so a whitespace-padded value must
// be collapsed before the enum comparison rather than rejected.
func TestOpenContent_ModeWhitespace(t *testing.T) {
	t.Parallel()

	t.Run("openContent mode with surrounding whitespace compiles as suffix", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode=" suffix "><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "mode=' suffix ' must collapse to 'suffix'")
	})

	t.Run("openContent mode=' none ' compiles (collapses to none)", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode=" none "/>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "mode=' none ' must collapse to 'none'")
	})

	t.Run("openContent with an invalid mode still errors", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="bogus"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "an invalid mode value is still a schema error")
	})

	t.Run("defaultOpenContent mode with whitespace compiles", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:defaultOpenContent mode="  interleave  "><xs:any namespace="##any" processContents="skip"/></xs:defaultOpenContent>
  <xs:element name="doc"><xs:complexType>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "defaultOpenContent mode='  interleave  ' must collapse")
	})

	t.Run("defaultOpenContent appliesToEmpty with whitespace is treated as true", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:defaultOpenContent mode="interleave" appliesToEmpty=" true "><xs:any namespace="##any" processContents="skip"/></xs:defaultOpenContent>
  <xs:element name="doc"><xs:complexType>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "appliesToEmpty=' true ' must collapse to true and compile")
	})
}
