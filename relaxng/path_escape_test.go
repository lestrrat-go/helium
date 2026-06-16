package relaxng_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// <include href="../..."> / <externalRef href="../..."> paths that
// "../"-escape the compiler's baseDir must be rejected with a clear
// "escapes base directory" error. Defense-in-depth for callers wiring in
// permissive fs.FS implementations; os.DirFS would already refuse via
// fs.ValidPath, but the check here gives consistent behavior independent
// of the FS choice. Mirrors xsd/path_escape_test.go.
func TestCompile_RejectsParentEscapeInHref(t *testing.T) {
	t.Parallel()

	const includer = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <include href="../escape.rng"/>
  <start><ref name="root"/></start>
</grammar>`

	const externalRefer = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><externalRef href="../escape.rng"/></start>
</grammar>`

	const escapeRNG = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <define name="root"><element name="root"><text/></element></define>
  <start><ref name="root"/></start>
</grammar>`

	fsys := fstest.MapFS{
		"schemas/main.rng":   &fstest.MapFile{Data: []byte(includer)},
		"schemas/extref.rng": &fstest.MapFile{Data: []byte(externalRefer)},
		"escape.rng":         &fstest.MapFile{Data: []byte(escapeRNG)},
	}

	compile := func(t *testing.T, file string) string {
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

	t.Run("include", func(t *testing.T) {
		t.Parallel()
		got := compile(t, "schemas/main.rng")
		require.True(t, strings.Contains(got, "escapes base directory"),
			"error should mention escape; got: %s", got)
	})

	t.Run("externalRef", func(t *testing.T) {
		t.Parallel()
		got := compile(t, "schemas/extref.rng")
		require.True(t, strings.Contains(got, "escapes base directory"),
			"error should mention escape; got: %s", got)
	})
}

// An absolute href cannot be proven contained in a *relative* baseDir:
// filepath.Rel returns an error in that case, and the guard must treat that
// as a containment failure rather than silently allowing the load. Without
// the fix this reintroduces the absolute-href escape.
func TestCompile_RejectsAbsoluteHrefWithRelativeBaseDir(t *testing.T) {
	t.Parallel()

	const includer = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <include href="/etc/passwd"/>
  <start><ref name="root"/></start>
</grammar>`

	const escapeRNG = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <define name="root"><element name="root"><text/></element></define>
  <start><ref name="root"/></start>
</grammar>`

	fsys := fstest.MapFS{
		"schemas/main.rng": &fstest.MapFile{Data: []byte(includer)},
		"etc/passwd":       &fstest.MapFile{Data: []byte(escapeRNG)},
	}

	data, err := fsys.ReadFile("schemas/main.rng")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = relaxng.NewCompiler().FS(fsys).BaseDir("schemas").ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.True(t, strings.Contains(compileErrors, "escapes base directory"),
		"absolute href against relative baseDir must be rejected; got: %s", compileErrors)
}

// A local filesystem path that merely contains "://" somewhere (not as a
// leading scheme) must not be mistaken for a remote URI and skip the
// containment check.
func TestCompile_RejectsSchemeLikeLocalPathEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outsideDir := filepath.Join(root, "x://outside")
	baseDir := filepath.Join(root, "base")
	require.NoError(t, os.MkdirAll(outsideDir, 0o755))
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	const escapeRNG = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <define name="root"><element name="root"><text/></element></define>
  <start><ref name="root"/></start>
</grammar>`
	outsidePath := filepath.Join(outsideDir, "escape.rng")
	require.NoError(t, os.WriteFile(outsidePath, []byte(escapeRNG), 0o644))

	schema := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <include href="` + outsidePath + `"/>
  <start><ref name="root"/></start>
</grammar>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = relaxng.NewCompiler().FS(osFS{}).BaseDir(baseDir).ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.True(t, strings.Contains(compileErrors, "escapes base directory"),
		"scheme-like local path must not bypass containment; got: %s", compileErrors)
}
