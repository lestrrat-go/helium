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

const validTargetRNG = `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <define name="root"><element name="root"><text/></element></define>
  <start><ref name="root"/></start>
</grammar>`

// A freshly constructed Compiler must NOT read host files referenced by an
// untrusted schema's externalRef/include href: the default FS is deny-all, so an
// absolute href that points at a real on-disk file fails to load. The same
// schema loads only when the caller explicitly opts into host access via
// Compiler.FS(helium.PermissiveFS()), proving the file is reachable and the
// deny-all default is what blocks it (RNG-007).
func TestCompile_DenyAllFSByDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "target.rng")
	require.NoError(t, os.WriteFile(target, []byte(validTargetRNG), 0o600))

	run := func(t *testing.T, schema string, c relaxng.Compiler) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = c.ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		return compileErrors
	}

	externalRefSchema := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><externalRef href="` + target + `"/></start>
</grammar>`

	includeSchema := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <include href="` + target + `"/>
  <start><ref name="root"/></start>
</grammar>`

	t.Run("externalRef denied by default", func(t *testing.T) {
		t.Parallel()
		got := run(t, externalRefSchema, relaxng.NewCompiler())
		require.Contains(t, got, "could not load",
			"untrusted externalRef must not read host files by default")
	})

	t.Run("include denied by default", func(t *testing.T) {
		t.Parallel()
		got := run(t, includeSchema, relaxng.NewCompiler())
		require.Contains(t, got, "could not load",
			"untrusted include must not read host files by default")
	})

	t.Run("externalRef loads with explicit PermissiveFS", func(t *testing.T) {
		t.Parallel()
		got := run(t, externalRefSchema, relaxng.NewCompiler().FS(helium.PermissiveFS()))
		require.Empty(t, got,
			"externalRef must load once host access is explicitly granted")
	})
}

// A single include/externalRef target larger than the configured
// MaxResourceBytes cap must be rejected with a clear "exceeds the maximum
// resource size" error rather than read in full (RNG-008).
func TestCompile_MaxResourceBytes(t *testing.T) {
	t.Parallel()

	// A grammar padded with a large comment so it exceeds a small cap. It
	// carries only a <define> (no <start>) so it can be pulled in via either
	// <include> (which supplies its own <start>) or <externalRef>.
	oversized := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <!-- ` + strings.Repeat("x", 4096) + ` -->
  <define name="root"><element name="root"><text/></element></define>
</grammar>`

	includeSchema := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <include href="target.rng"/>
  <start><ref name="root"/></start>
</grammar>`

	externalRefSchema := `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start><externalRef href="target.rng"/></start>
</grammar>`

	fsys := fstest.MapFS{
		"target.rng": &fstest.MapFile{Data: []byte(oversized)},
	}

	run := func(t *testing.T, schema string, limit int) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().
			FS(fsys).
			MaxResourceBytes(limit).
			ErrorHandler(collector).
			Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		return compileErrors
	}

	t.Run("oversized include rejected", func(t *testing.T) {
		t.Parallel()
		got := run(t, includeSchema, 1024)
		require.Contains(t, got, "exceeds the maximum resource size")
	})

	t.Run("oversized externalRef rejected", func(t *testing.T) {
		t.Parallel()
		got := run(t, externalRefSchema, 1024)
		require.Contains(t, got, "exceeds the maximum resource size")
	})

	t.Run("within-cap include accepted", func(t *testing.T) {
		t.Parallel()
		got := run(t, includeSchema, len(oversized))
		require.Empty(t, got, "a resource at or under the cap must load")
	})
}
