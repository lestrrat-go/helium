package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A path-escape ("../") in an xs:include/xs:import/xs:redefine schemaLocation
// must be fatal even when it appears in a schema reached via a nested xs:import,
// exactly as it is on the outer (top-level) import path. Before the fix the
// nested impC.processIncludes call demoted errSchemaPathEscape to a warning, so
// an imported schema's escaping nested reference was silently swallowed and
// compilation succeeded. Mirror the outer fatal handling: a nested escape aborts
// compilation.
func TestCompile_RejectsNestedParentEscapeInSchemaLocation(t *testing.T) {
	t.Parallel()

	// Top-level schema imports an in-bounds intermediate schema (no escape here).
	const mainXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:import namespace="urn:intermediate" schemaLocation="intermediate.xsd"/>
</xs:schema>`

	// The escape target an attacker would want to reach, sitting above the base.
	const escapeXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"/>`

	cases := []struct {
		name string
		// intermediate is the imported schema that contains the nested,
		// escaping reference.
		intermediate string
	}{
		{
			name: "nested include",
			intermediate: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:intermediate">
  <xs:include schemaLocation="../escape.xsd"/>
</xs:schema>`,
		},
		{
			name: "nested import",
			intermediate: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:intermediate">
  <xs:import namespace="urn:escape" schemaLocation="../escape.xsd"/>
</xs:schema>`,
		},
		{
			name: "nested redefine",
			intermediate: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:intermediate">
  <xs:redefine schemaLocation="../escape.xsd"/>
</xs:schema>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{
				"schemas/main.xsd":         &fstest.MapFile{Data: []byte(mainXSD)},
				"schemas/intermediate.xsd": &fstest.MapFile{Data: []byte(tc.intermediate)},
				// A file at the parent level the attacker might want to read.
				"escape.xsd": &fstest.MapFile{Data: []byte(escapeXSD)},
			}

			data, err := fsys.ReadFile("schemas/main.xsd")
			require.NoError(t, err)
			doc, err := helium.NewParser().Parse(t.Context(), data)
			require.NoError(t, err)

			_, err = xsd.NewCompiler().FS(fsys).BaseDir("schemas").Label("schemas/main.xsd").Compile(t.Context(), doc)
			require.Error(t, err, "compile must reject nested schemaLocation that escapes baseDir")
			require.True(t, strings.Contains(err.Error(), "escapes base directory"),
				"error should mention escape; got: %v", err)
		})
	}
}
