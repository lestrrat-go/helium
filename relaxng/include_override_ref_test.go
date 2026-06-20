package relaxng_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// compileIncludeErrors compiles the named file from fsys (with baseDir
// "schemas") and returns the fatal compile errors.
func compileIncludeErrors(t *testing.T, fsys fstest.MapFS, file string) string {
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

// TestIncludeOverrideRemovesDanglingRef covers 661-3: per RELAX NG include
// semantics, a <define>/<start> overridden by an <include> body is REMOVED from
// the included grammar before ref resolution. A <ref> that lives only inside an
// overridden (removed) define must therefore never be resolved or errored —
// even if it names a define that does not exist. The round-3 fatal-ref machinery
// must not reject such a valid schema.
func TestIncludeOverrideRemovesDanglingRef(t *testing.T) {
	t.Parallel()

	const ns = `xmlns="http://relaxng.org/ns/structure/1.0"`

	t.Run("overridden define with dangling ref compiles", func(t *testing.T) {
		t.Parallel()
		// included.rng has <define name="x"> whose body references a
		// non-existent <ref name="missing"/>. The including grammar overrides
		// <define name="x"> in its <include> body, so the original define (and
		// its dangling ref) is removed and must not be resolved/errored.
		included := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <define name="x">
    <ref name="missing"/>
  </define>
</grammar>`

		main := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <include href="included.rng">
    <define name="x">
      <text/>
    </define>
  </include>
  <start>
    <element name="root">
      <ref name="x"/>
    </element>
  </start>
</grammar>`

		fsys := fstest.MapFS{
			"schemas/main.rng":     &fstest.MapFile{Data: []byte(main)},
			"schemas/included.rng": &fstest.MapFile{Data: []byte(included)},
		}
		got := compileIncludeErrors(t, fsys, "schemas/main.rng")
		require.Empty(t, got,
			"overriding the define that contained the dangling ref must remove it; got: %s", got)
	})

	t.Run("non-overridden dangling ref still errors", func(t *testing.T) {
		t.Parallel()
		// The included grammar has TWO defines: x (overridden) and y (NOT
		// overridden) whose body references a non-existent define. The ref
		// inside y survives and must still be a fatal compile error (round-3).
		included := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <define name="x">
    <text/>
  </define>
  <define name="y">
    <ref name="missing"/>
  </define>
</grammar>`

		main := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <include href="included.rng">
    <define name="x">
      <text/>
    </define>
  </include>
  <start>
    <element name="root">
      <ref name="y"/>
    </element>
  </start>
</grammar>`

		fsys := fstest.MapFS{
			"schemas/main.rng":     &fstest.MapFile{Data: []byte(main)},
			"schemas/included.rng": &fstest.MapFile{Data: []byte(included)},
		}
		got := compileIncludeErrors(t, fsys, "schemas/main.rng")
		require.True(t, strings.Contains(got, "missing"),
			"a dangling ref in a NON-overridden define must still be a fatal compile error; got: %s", got)
	})
}
