package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestElementConsistentIncludeSource verifies that a cos-element-consistent
// diagnostic for an inconsistent complex type pulled in via xs:include is
// attributed to the INCLUDED file (whose line number it carries), not the
// top-level schema label. Before the fix the diagnostic always cited c.filename
// (the main schema) while reporting the included file's line number, producing a
// mismatched file:line that points into the wrong document.
func TestElementConsistentIncludeSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "main.xsd"
		incXSD  = "inc.xsd"
	)

	// main.xsd has padding lines so that, were the diagnostic mis-attributed to
	// main.xsd, the inc.xsd line number would clearly not line up with main.xsd.
	// The inconsistent complex type lives entirely in inc.xsd.
	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
</xs:schema>`)},
		incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="bad">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a" type="xs:string"/>
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

	require.Contains(t, errStr, "but different type definitions, appear in the content model.",
		"expected the cos-element-consistent diagnostic")
	// The diagnostic must cite the included file, not the main schema label.
	require.Contains(t, errStr, incXSD,
		"diagnostic must be attributed to the included file; got: %q", errStr)
	require.False(t, strings.Contains(errStr, mainXSD+":"),
		"diagnostic must not cite the top-level schema label with the included file's line; got: %q", errStr)
}
