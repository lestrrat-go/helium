package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// An xs:redefine whose target document was ALREADY loaded (here via an earlier
// xs:include of the same schema) must not be silently ignored. Before the fix
// the includeVisited guard caused loadRedefine to return nil for the already-seen
// document, dropping the redefine's override children entirely and yielding a
// schema missing the intended override. The repeated redefinition is now
// reported as a schema error instead of being silently discarded.
func TestRedefine_AlreadyLoadedDocument_Reported(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "rr_main.xsd"
		baseXSD = "rr_base.xsd"
	)
	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="rr_base.xsd"/>
  <xs:redefine schemaLocation="rr_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="a" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
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
	require.ErrorIs(t, err, xsd.ErrCompilationFailed,
		"redefine of an already-loaded schema must be reported, not silently ignored")
	require.NoError(t, collector.Close())
	_, errStr := partitionCompileErrors(collector.Errors())
	require.Contains(t, errStr, "already loaded")
	require.Contains(t, errStr, "repeated redefinition")
}
