package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// xs:include / xs:import / xs:redefine paths that "../"-escape the
// compiler's baseDir must be rejected with a clear error. Defense-in-depth
// for callers wiring in permissive fs.FS implementations; os.DirFS would
// already refuse via fs.ValidPath, but the check here ensures consistent
// behavior independent of the FS choice.
func TestCompile_RejectsParentEscapeInSchemaLocation(t *testing.T) {
	t.Parallel()

	const escapeXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"/>`
	const importerEscape = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:import namespace="urn:escape" schemaLocation="../escape.xsd"/>
</xs:schema>`

	cases := []struct {
		name string
		body string
	}{
		{
			name: "include",
			body: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="../escape.xsd"/>
</xs:schema>`,
		},
		{
			name: "import",
			body: importerEscape,
		},
		{
			name: "redefine",
			body: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="../escape.xsd"/>
</xs:schema>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{
				"schemas/main.xsd": &fstest.MapFile{Data: []byte(tc.body)},
				// A file at the parent level the attacker might want to read.
				"escape.xsd": &fstest.MapFile{Data: []byte(escapeXSD)},
			}

			data, err := fsys.ReadFile("schemas/main.xsd")
			require.NoError(t, err)
			doc, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err)

			_, err = xsd.NewCompiler().FS(fsys).BaseDir("schemas").Label("schemas/main.xsd").Compile(t.Context(), doc)
			require.Error(t, err, "compile must reject schemaLocation that escapes baseDir")
			require.True(t, strings.Contains(err.Error(), "escapes base directory"),
				"error should mention escape; got: %v", err)
		})
	}
}
