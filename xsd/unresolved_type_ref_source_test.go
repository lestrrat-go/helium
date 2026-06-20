package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestUnresolvedTypeRefElementKind verifies that an unresolved type reference is
// reported with the OWNER type's actual element kind (complexType vs simpleType),
// not a hard-coded "simpleType". A complexContent restriction/extension over a
// missing base is a complex type, so the diagnostic must say "element complexType"
// and "local complex type".
func TestUnresolvedTypeRefElementKind(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	t.Run("complexContent restriction over missing base reports complexType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:restriction base="MissingBase">
        <xs:sequence/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="T"/>
</xs:schema>`
		errs := compileErrors(t, schemaXML)
		require.Contains(t, errs, "does not resolve to a(n) type definition")
		require.Contains(t, errs, "element complexType:",
			"a complex type's unresolved base ref must report element complexType; got: %q", errs)
		require.NotContains(t, errs, "element simpleType:",
			"must not mis-report a complex type as simpleType; got: %q", errs)
	})

	t.Run("simpleType restriction over missing base still reports simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="S">
    <xs:restriction base="MissingBase"/>
  </xs:simpleType>
  <xs:element name="root" type="S"/>
</xs:schema>`
		errs := compileErrors(t, schemaXML)
		require.Contains(t, errs, "element simpleType:",
			"a simple type's unresolved base ref must report element simpleType; got: %q", errs)
	})
}

// TestUnresolvedTypeRefImportSource verifies that an unresolved type reference in
// an IMPORTED schema is attributed to the imported file (whose line number it
// carries), not the top-level schema label. Before the fix reportUnresolvedTypeRef
// hard-coded c.filename, mis-attributing the imported file's line to the importer.
func TestUnresolvedTypeRefImportSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "main.xsd"
		impXSD  = "imp.xsd"
	)

	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:i="urn:i">
  <xs:import namespace="urn:i" schemaLocation="imp.xsd"/>
  <xs:element name="root" type="i:S"/>
</xs:schema>`)},
		impXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:i">
  <xs:simpleType name="S">
    <xs:restriction base="MissingBase"/>
  </xs:simpleType>
</xs:schema>`)},
	}

	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	require.NoError(t, collector.Close())
	_, errStr := partitionCompileErrors(collector.Errors())

	require.Contains(t, errStr, "does not resolve to a(n) type definition")
	require.Contains(t, errStr, impXSD,
		"the unresolved-base diagnostic must be attributed to the imported file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"diagnostic must not cite the top-level schema label with the imported file's line; got: %q", errStr)
}

// TestUnresolvedElementRefIncludeSource verifies that an unresolved element ref in
// an INCLUDED schema is attributed to the included file, not the including label.
// elemRefSource now carries the declaring source captured at collection time.
func TestUnresolvedElementRefIncludeSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "main.xsd"
		incXSD  = "inc.xsd"
	)

	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
</xs:schema>`)},
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="Missing"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)},
	}

	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	requireCompileResultErr(t, err)
	require.NoError(t, collector.Close())
	_, errStr := partitionCompileErrors(collector.Errors())

	require.Contains(t, errStr, "does not resolve to a(n) element declaration")
	require.Contains(t, errStr, incXSD,
		"the unresolved element-ref diagnostic must be attributed to the included file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"diagnostic must not cite the top-level schema label with the included file's line; got: %q", errStr)
}
