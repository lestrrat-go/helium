package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestSchemaNamespaceAttrRejected verifies that an attribute in the XML Schema
// namespace appearing on a schema-component element is a representation error.
// The schema-for-schemas allows only unqualified schema attributes plus foreign
// attributes from a namespace OTHER than the XSD namespace (##other), so an
// attribute in the XSD namespace itself (e.g. xsd:type, xsd:targetNamespace) is
// neither and must be rejected. Version-independent (1.0 and 1.1).
func TestSchemaNamespaceAttrRejected(t *testing.T) {
	t.Parallel()

	const mainXSD = "s.xsd"
	const want = "Attributes from the schema namespace"

	compile := func(t *testing.T, body string) string {
		t.Helper()
		fsys := fstest.MapFS{mainXSD: &fstest.MapFile{Data: []byte(body)}}
		doc, err := helium.NewParser().Parse(t.Context(), fsys[mainXSD].Data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, cerr)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr
	}

	t.Run("xsd:type on complexType and attribute (addB082)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:attribute name="foo" xsd:type="xsd:string"/>
  <xsd:complexType name="ct" xsd:type="xsd:integer"/>
  <xsd:element name="doc" type="ct"/>
</xsd:schema>`)
		require.Contains(t, errStr, want)
	})

	t.Run("xsd:targetNamespace on schema (test64756)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xsd:targetNamespace="http://foobar">
  <xsd:element name="foo"/>
</xsd:schema>`)
		require.Contains(t, errStr, want)
	})

	t.Run("schema-namespace attribute on notation (notatE002)", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:notation a:b="c" name="jpeg" public="image/jpeg" system="viewer.exe" xmlns:a="http://www.w3.org/2001/XMLSchema"/>
</xsd:schema>`)
		require.Contains(t, errStr, want)
	})

	t.Run("foreign-namespace attribute is allowed", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:o="urn:other">
  <xsd:element name="foo" o:note="hi" type="xsd:string"/>
</xsd:schema>`)
		require.False(t, strings.Contains(errStr, want),
			"foreign (non-schema-namespace) attributes must be permitted; got: %q", errStr)
	})
}
