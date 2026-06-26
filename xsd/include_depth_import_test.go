package xsd_test

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// An over-deep xs:include chain nested inside an IMPORTED schema must abort
// compilation, not be silently swallowed. loadImport processes the imported
// schema's own includes in a fallback that demotes ordinary load failures to
// warnings; the include-depth guard (errIncludeDepthExceeded) must be classified
// fatal by IsFatalSchemaLoad so it survives that demotion and propagates. Before
// the fix the guard was not fatal, so a hostile deep include chain inside an
// import was ignored.
func TestCompile_ImportedSchema_OverDeepInclude_Fatal(t *testing.T) {
	const (
		nsA = "urn:a"
		nsB = "urn:b"
	)

	// Depth far exceeds defaultMaxIncludeDepth (40); chain0 includes chain1, ...,
	// so the nested-include depth guard fires while processing the imported schema.
	const chainLen = 60

	fsys := fstest.MapFS{
		"main.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + nsA + `">
  <xs:import namespace="` + nsB + `" schemaLocation="imp.xsd"/>
</xs:schema>`)},
		"imp.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + nsB + `">
  <xs:include schemaLocation="chain0.xsd"/>
</xs:schema>`)},
	}
	for i := range chainLen {
		body := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + nsB + `">`
		if i+1 < chainLen {
			body += fmt.Sprintf("\n  <xs:include schemaLocation=%q/>", fmt.Sprintf("chain%d.xsd", i+1))
		}
		body += "\n</xs:schema>"
		fsys[fmt.Sprintf("chain%d.xsd", i)] = &fstest.MapFile{Data: []byte(body)}
	}

	data, err := fsys.ReadFile("main.xsd")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(fsys).Compile(t.Context(), doc)
	require.Error(t, err, "an over-deep include chain inside an imported schema must abort compilation")
	require.True(t, strings.Contains(err.Error(), "max include depth"),
		"error must mention the include depth limit; got: %v", err)
}
