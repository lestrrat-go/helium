package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// An xs:import declares the namespace it expects the referenced schema to
// contribute. If the schema found at schemaLocation has a *different*
// targetNamespace, the import must be rejected — otherwise a schema imported
// as one namespace silently contributes declarations from another. This
// mirrors the libxml2/XSD src-import constraint and the existing xs:include
// target-namespace check. Compile-time schema errors surface through the
// ErrorHandler (not the returned error), so the test inspects collected errors.
func TestCompile_ImportNamespaceMismatch(t *testing.T) {
	compileErrors := func(t *testing.T, fsys fstest.MapFS) string {
		t.Helper()
		data, err := fsys.ReadFile("main.xsd")
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("main.xsd").ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return b.String()
	}

	// main imports urn:expected, but other.xsd declares urn:other. Reject.
	t.Run("declared namespace differs from imported targetNamespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			"main.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import namespace="urn:expected" schemaLocation="other.xsd"/>
</xs:schema>`)},
			"other.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:other">
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.True(t, strings.Contains(got, "urn:other") && strings.Contains(got, "urn:expected"),
			"error must name both the imported targetNamespace and the requested namespace; got: %q", got)
	})

	// no-namespace import: namespace attr absent, but other.xsd has a TNS. Reject.
	t.Run("no-namespace import of a namespaced schema", func(t *testing.T) {
		fsys := fstest.MapFS{
			"main.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
</xs:schema>`)},
			"other.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:other">
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.True(t, strings.Contains(got, "urn:other"),
			"error must name the imported targetNamespace; got: %q", got)
	})

	// Matching namespace: import succeeds, no error, and the imported
	// declaration resolves.
	t.Run("matching namespace still works", func(t *testing.T) {
		fsys := fstest.MapFS{
			"main.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:o="urn:expected"
  targetNamespace="urn:main">
  <xs:import namespace="urn:expected" schemaLocation="other.xsd"/>
  <xs:element name="root" type="o:t"/>
</xs:schema>`)},
			"other.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:expected">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.Empty(t, got, "import with a matching targetNamespace must compile without error")
	})

	// No-namespace import of a no-namespace schema: valid, must still work.
	t.Run("no-namespace import of a no-namespace schema works", func(t *testing.T) {
		fsys := fstest.MapFS{
			"main.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			"other.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.Empty(t, got, "no-namespace import of a no-namespace schema must compile without error")
	})
}
