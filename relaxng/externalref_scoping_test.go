package relaxng_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// An <externalRef> target is an INDEPENDENT schema. It must not see the
// includer's grammar scope: a <parentRef> inside the external target must not
// resolve to the including grammar, and a bare external <ref> must not bind to
// the includer's defines. xmllint rejects both as compile errors. These tests
// pin that behavior (codex 661-4) and confirm a self-contained externalRef
// still loads and validates.
func compileExternalRefErrors(t *testing.T, fsys fstest.MapFS, file string) string {
	t.Helper()
	data, err := fsys.ReadFile(file)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = relaxng.NewCompiler().FS(fsys).BaseDir("schemas").ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	return compileErrors
}

func TestExternalRefScopingIsIndependent(t *testing.T) {
	t.Parallel()

	const ns = `xmlns="http://relaxng.org/ns/structure/1.0"`

	t.Run("parentRef in external target cannot reach includer scope", func(t *testing.T) {
		t.Parallel()

		// The includer defines "shared". The external target uses
		// <parentRef name="shared"/>. If externalRef scoping leaked, the
		// external grammar's parent would be the includer and the parentRef
		// would resolve there. It must NOT — the external grammar has no
		// parent, so this is a fatal compile error.
		main := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <start><externalRef href="extref.rng"/></start>
  <define name="shared">
    <element name="shared"><text/></element>
  </define>
</grammar>`

		extref := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <start>
    <element name="root">
      <parentRef name="shared"/>
    </element>
  </start>
</grammar>`

		fsys := fstest.MapFS{
			"schemas/main.rng":   &fstest.MapFile{Data: []byte(main)},
			"schemas/extref.rng": &fstest.MapFile{Data: []byte(extref)},
		}

		got := compileExternalRefErrors(t, fsys, "schemas/main.rng")
		require.NotEmpty(t, got,
			"parentRef in an externalRef target must not reach the includer scope")
		require.True(t, strings.Contains(got, "parent grammar scope") ||
			strings.Contains(got, "no matching definition"),
			"want unresolved-parentRef error; got: %s", got)
	})

	t.Run("bare external ref cannot bind to includer defines", func(t *testing.T) {
		t.Parallel()

		// The includer defines "shared". The external target is a bare
		// pattern (non-grammar) containing <ref name="shared"/>. A bare
		// external ref must not bind to the includer's defines.
		main := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <start><externalRef href="extref.rng"/></start>
  <define name="shared">
    <element name="shared"><text/></element>
  </define>
</grammar>`

		extref := `<?xml version="1.0"?>
<element name="root" ` + ns + `>
  <ref name="shared"/>
</element>`

		fsys := fstest.MapFS{
			"schemas/main.rng":   &fstest.MapFile{Data: []byte(main)},
			"schemas/extref.rng": &fstest.MapFile{Data: []byte(extref)},
		}

		got := compileExternalRefErrors(t, fsys, "schemas/main.rng")
		require.NotEmpty(t, got,
			"a bare external <ref> must not bind to the includer's defines")
		require.True(t, strings.Contains(got, "no matching definition") ||
			strings.Contains(got, "has no parent grammar scope"),
			"want unresolved-ref error; got: %s", got)
	})

	t.Run("self-contained external grammar loads and validates", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <start><externalRef href="extref.rng"/></start>
</grammar>`

		// A self-contained external grammar with its own define resolved
		// internally must still compile and validate.
		extref := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <start>
    <element name="root">
      <ref name="body"/>
    </element>
  </start>
  <define name="body">
    <element name="body"><text/></element>
  </define>
</grammar>`

		fsys := fstest.MapFS{
			"schemas/main.rng":   &fstest.MapFile{Data: []byte(main)},
			"schemas/extref.rng": &fstest.MapFile{Data: []byte(extref)},
		}

		got := compileExternalRefErrors(t, fsys, "schemas/main.rng")
		require.Empty(t, got, "self-contained externalRef must compile cleanly; got: %s", got)

		data, err := fsys.ReadFile("schemas/main.rng")
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		grammar, err := relaxng.NewCompiler().FS(fsys).BaseDir("schemas").Compile(t.Context(), doc)
		require.NoError(t, err)

		instance, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><body>hi</body></root>`))
		require.NoError(t, err)
		require.NoError(t, relaxng.NewValidator(grammar).Validate(t.Context(), instance),
			"valid instance must pass against self-contained externalRef")

		bad, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><wrong>hi</wrong></root>`))
		require.NoError(t, err)
		require.Error(t, relaxng.NewValidator(grammar).Validate(t.Context(), bad),
			"invalid instance must fail")
	})
}
