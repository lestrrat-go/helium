package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEmptyContentCDATA checks that an empty complex type rejects non-blank
// character content represented as a CDATA section, the same way it rejects
// non-blank plain text. Whitespace-only CDATA is allowed, mirroring the
// handling of whitespace-only text.
func TestEmptyContentCDATA(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType/>
  </xs:element>
</xs:schema>`

	t.Run("rejects non-blank CDATA content", func(t *testing.T) {
		var out string
		err := compileAndValidate(t, schemaXML, "<e><![CDATA[x]]></e>", &out)
		require.Error(t, err)
		require.Contains(t, out, "Character content is not allowed")
	})

	t.Run("rejects non-blank text content", func(t *testing.T) {
		var out string
		err := compileAndValidate(t, schemaXML, "<e>x</e>", &out)
		require.Error(t, err)
		require.Contains(t, out, "Character content is not allowed")
	})

	t.Run("allows whitespace-only CDATA content", func(t *testing.T) {
		require.NoError(t, compileAndValidate(t, schemaXML, "<e><![CDATA[   ]]></e>", nil))
	})

	t.Run("allows empty element", func(t *testing.T) {
		require.NoError(t, compileAndValidate(t, schemaXML, "<e/>", nil))
	})
}
