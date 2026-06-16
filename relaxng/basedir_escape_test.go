package relaxng_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// recordingFS wraps an fs.FS and records every path handed to Open, so a
// test can assert that no path outside the configured BaseDir ever reaches
// the filesystem.
type recordingFS struct {
	fsys   fs.FS
	mu     sync.Mutex
	opened []string
}

func (r *recordingFS) Open(name string) (fs.File, error) {
	r.mu.Lock()
	r.opened = append(r.opened, name)
	r.mu.Unlock()
	return r.fsys.Open(name)
}

func (r *recordingFS) openedPaths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.opened))
	copy(out, r.opened)
	return out
}

// osFS opens absolute OS paths verbatim, mirroring the package default
// (iofs.PermissiveRoot). os.DirFS/MapFS reject absolute names via
// fs.ValidPath, so a raw os.Open-backed FS is required to exercise the
// path-containment guard against real outside files.
type osFS struct{}

func (osFS) Open(name string) (fs.File, error) { return os.Open(name) }

// A grammar that loads fine on its own, used as the "outside" target the
// guard must refuse to read.
const escapeTargetRNG = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <define name="root"><element name="root"><text/></element></define>
  <start><ref name="root"/></start>
</grammar>`

const innerRNG = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <define name="root"><element name="root"><text/></element></define>
</grammar>`

// resolveHref applied its base-directory containment check to only one of
// its three return paths. An absolute href, and an href resolved through an
// escaping xml:base, both returned before the check and could read files
// outside the configured BaseDir. These cases must be rejected and must
// never reach the filesystem; a normal in-base relative href must still load.
func TestCompile_BaseDirContainsAllResolvedHrefs(t *testing.T) {
	t.Parallel()

	// Lay out an "outside" directory holding the secret target and a
	// separate "base" directory the compiler is confined to.
	root := t.TempDir()
	outsideDir := filepath.Join(root, "outside")
	baseDir := filepath.Join(root, "base")
	require.NoError(t, os.MkdirAll(outsideDir, 0o755))
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	outsidePath := filepath.Join(outsideDir, "escape.rng")
	require.NoError(t, os.WriteFile(outsidePath, []byte(escapeTargetRNG), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "inner.rng"), []byte(innerRNG), 0o644))

	compile := func(t *testing.T, schema string) (*recordingFS, string) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		// Anchor the document inside baseDir so NodeGetBase can resolve
		// xml:base against a real in-base location.
		doc.SetURL(filepath.Join(baseDir, "main.rng"))

		rec := &recordingFS{fsys: osFS{}}
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().FS(rec).BaseDir(baseDir).ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		return rec, compileErrors
	}

	assertNoOutsideOpen := func(t *testing.T, rec *recordingFS) {
		t.Helper()
		for _, p := range rec.openedPaths() {
			require.False(t, strings.Contains(filepath.Clean(p), outsideDir),
				"outside path must never reach Open; got %q", p)
		}
	}

	t.Run("absolute href outside base", func(t *testing.T) {
		t.Parallel()
		schema := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <include href="` + outsidePath + `"/>
  <start><ref name="root"/></start>
</grammar>`
		rec, errs := compile(t, schema)
		require.Contains(t, errs, "escapes base directory",
			"absolute outside href must be rejected; got: %s", errs)
		assertNoOutsideOpen(t, rec)
	})

	t.Run("escaping xml:base with relative href", func(t *testing.T) {
		t.Parallel()
		// xml:base climbs out of baseDir, then the relative href dives back
		// into the outside directory. NodeGetBase resolves against the
		// in-base document URL, producing a path above baseDir.
		schema := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0" xmlns:xml="http://www.w3.org/XML/1998/namespace">
  <div xml:base="../outside/">
    <include href="escape.rng"/>
  </div>
  <start><ref name="root"/></start>
</grammar>`
		rec, errs := compile(t, schema)
		require.Contains(t, errs, "escapes base directory",
			"escaping xml:base must be rejected; got: %s", errs)
		assertNoOutsideOpen(t, rec)
	})

	t.Run("in-base relative href still loads", func(t *testing.T) {
		t.Parallel()
		schema := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <include href="inner.rng"/>
  <start><ref name="root"/></start>
</grammar>`
		rec, errs := compile(t, schema)
		require.Empty(t, errs, "in-base relative href must compile cleanly; got: %s", errs)

		var sawInner bool
		for _, p := range rec.openedPaths() {
			if strings.HasSuffix(filepath.Clean(p), filepath.Join("base", "inner.rng")) {
				sawInner = true
			}
		}
		require.True(t, sawInner, "in-base include should have been opened; opened: %v", rec.openedPaths())
	})
}
