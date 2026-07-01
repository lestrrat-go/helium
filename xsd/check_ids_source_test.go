package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestComponentIDDuplicateSource verifies that the schema-component @id
// uniqueness diagnostic (XSD 1.1 check_ids.go) for a component pulled in via
// xs:include or xs:import is attributed to the DECLARING file (whose line number
// it carries), not the top-level schema label. The check walks the nested
// document's DOM and reports at that document's line numbers, so citing
// c.filename (the including schema) instead of c.diagSource() produces a
// mismatched file:line pointing into the wrong document.
func TestComponentIDDuplicateSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "main.xsd"
		incXSD  = "inc.xsd"
		impXSD  = "imp.xsd"
	)

	const notUnique = "is not unique within the schema document"

	// The two same-@id components live entirely in the included/imported file, so
	// the diagnostic's line number is meaningful only when attributed to that file.
	assert := func(t *testing.T, fsys fstest.MapFS, declFile string) {
		t.Helper()
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())

		require.Contains(t, errStr, notUnique, "expected the duplicate schema-component @id diagnostic")
		require.Contains(t, errStr, declFile+":",
			"diagnostic must be attributed to the declaring file; got: %q", errStr)
		require.False(t, strings.Contains(errStr, mainXSD+":"),
			"diagnostic must not cite the top-level schema label; got: %q", errStr)
	}

	t.Run("included file", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
			incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" id="dup" type="xs:string"/>
  <xs:element name="b" id="dup" type="xs:string"/>
</xs:schema>`)},
		}
		assert(t, fsys, incXSD)
	})

	t.Run("imported file", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t">
  <xs:import namespace="urn:t" schemaLocation="imp.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
			impXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t">
  <xs:element name="a" id="dup" type="xs:string"/>
  <xs:element name="b" id="dup" type="xs:string"/>
</xs:schema>`)},
		}
		assert(t, fsys, impXSD)
	})
}
