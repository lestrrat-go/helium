package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAttrGroupDuplicateSource verifies that the duplicate-attribute-use
// (ag-props-correct.2) diagnostic for an attribute group pulled in via
// xs:include or xs:import is attributed to the DECLARING file (whose line number
// it carries), not the top-level schema label. Before the fix the diagnostic
// always cited c.filename (the main schema) while reporting the included file's
// line number, producing a mismatched file:line that points into the wrong
// document.
func TestAttrGroupDuplicateSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "main.xsd"
		incXSD  = "inc.xsd"
		impXSD  = "imp.xsd"
	)

	const dup = "Duplicate attribute use"

	// The attribute group with the internal duplicate lives entirely in the
	// included/imported file, so the diagnostic's line number is meaningful only
	// when attributed to that file.
	assert := func(t *testing.T, fsys fstest.MapFS, declFile string) {
		t.Helper()
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())

		require.Contains(t, errStr, dup, "expected the duplicate-attribute-use diagnostic")
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
  <xs:attributeGroup name="g">
    <xs:attribute name="x" type="xs:string"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:attributeGroup>
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
  <xs:attributeGroup name="g">
    <xs:attribute name="x" type="xs:string"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		assert(t, fsys, impXSD)
	})
}

// TestRedefineAttrGroupDuplicateSource verifies that when an xs:redefine
// OVERRIDE replaces an attribute group and the override body itself introduces a
// duplicate attribute use, the ag-props-correct.2 diagnostic is attributed to
// the REDEFINING file (where the override element and its line number live), not
// the redefined (base) file the original group was loaded from. The override
// replaces the stored group, so its recorded source must be re-pointed to the
// redefining file; otherwise the diagnostic keeps the base file's source but the
// override's line number, producing a mismatched file:line.
func TestRedefineAttrGroupDuplicateSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "redef_main.xsd"
		baseXSD = "redef_base.xsd"
	)

	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="redef_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="dup" type="xs:string"/>
      <xs:attribute name="dup" type="xs:int"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)},
		baseXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
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

	require.Contains(t, errStr, "Duplicate attribute use",
		"expected the duplicate-attribute-use diagnostic")
	require.Contains(t, errStr, mainXSD+":",
		"override duplicate must be attributed to the redefining file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, baseXSD+":"),
		"override duplicate must not cite the redefined base file; got: %q", errStr)
}
