package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// src-import (§4.2.6.1) constrains the @namespace of an <xs:import>, and a
// schema's @targetNamespace must not be the empty string. Both are
// version-independent schema-representation constraints surfaced through the
// ErrorHandler. These mirror W3C msMeta/Schema_w3c cases schZ014_a (namespace=""),
// schF4/schZ010 (import of own targetNamespace), schF3 (no-namespace import into a
// no-targetNamespace schema) and schZ014_b (targetNamespace="").
func TestCompile_SrcImportNamespaceConstraints(t *testing.T) {
	compileErrors := func(t *testing.T, fsys fstest.MapFS) string {
		t.Helper()
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label(importMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return b.String()
	}

	// namespace="" is not a namespace name; the absent namespace is imported by
	// OMITTING @namespace. Reject (schZ014_a).
	t.Run("empty import namespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:import namespace=""/>
</xs:schema>`)},
		}
		require.Contains(t, compileErrors(t, fsys), "must not be the empty string")
	})

	// src-import.1.1: @namespace must not match the importing schema's own
	// targetNamespace. Reject (schF4/schZ010).
	t.Run("import of own targetNamespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import namespace="urn:main"/>
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)},
		}
		require.Contains(t, compileErrors(t, fsys), "must not match the target namespace")
	})

	// src-import.1.2: an <import> without @namespace requires the enclosing schema
	// to have a targetNamespace. Reject (schF3).
	t.Run("no-namespace import in no-targetNamespace schema", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:import schemaLocation="other.xsd"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)},
		}
		require.Contains(t, compileErrors(t, fsys), "requires the enclosing schema to have a targetNamespace")
	})

	// A present targetNamespace must not be the empty string. Reject (schZ014_b).
	t.Run("empty schema targetNamespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="">
</xs:schema>`)},
		}
		require.Contains(t, compileErrors(t, fsys), "targetNamespace of a schema must not be the empty string")
	})

	// Guard against over-rejection: importing a DIFFERENT namespace is valid.
	t.Run("import of a different namespace is accepted", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import namespace="urn:other" schemaLocation="other.xsd"/>
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:other">
  <xs:element name="o" type="xs:string"/>
</xs:schema>`)},
		}
		require.Empty(t, compileErrors(t, fsys))
	})

	// Guard against over-rejection: a no-namespace import IS valid when the
	// enclosing schema has a targetNamespace (imports the absent namespace).
	t.Run("no-namespace import with a targetNamespace is accepted", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="o" type="xs:string"/>
</xs:schema>`)},
		}
		require.Empty(t, compileErrors(t, fsys))
	})
}
