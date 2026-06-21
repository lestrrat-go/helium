package relaxng_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUnresolvedRefIsCompileError covers 661-2 finding #1: per RELAX NG §4.18
// every <ref>/<parentRef> must refer to a <define>. A <parentRef> at the top
// level (no parent grammar scope) and a <ref> naming a define that does not
// exist in its scope must both be fatal compile errors, not silently-unresolved
// nodes that only fail at validation time.
func TestUnresolvedRefIsCompileError(t *testing.T) {
	t.Parallel()

	const ns = `xmlns="http://relaxng.org/ns/structure/1.0"`

	t.Run("top-level parentRef has no parent scope", func(t *testing.T) {
		t.Parallel()
		// <parentRef> in the outermost grammar has no parent grammar scope, so
		// it can never resolve. It must be a fatal compile error.
		schema := `<grammar ` + ns + `>
  <start>
    <element name="root">
      <parentRef name="x"/>
    </element>
  </start>
  <define name="x">
    <text/>
  </define>
</grammar>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"top-level <parentRef> (no parent scope) must be a fatal compile error")
	})

	t.Run("ref to missing name in scope", func(t *testing.T) {
		t.Parallel()
		// <ref name="missing"/> names a define that does not exist anywhere in
		// its grammar scope. It must be a fatal compile error.
		schema := `<grammar ` + ns + `>
  <start>
    <element name="root">
      <ref name="missing"/>
    </element>
  </start>
  <define name="x">
    <text/>
  </define>
</grammar>`
		require.NotEmpty(t, compileErrorsFor(t, schema),
			"<ref> to a name absent from its scope must be a fatal compile error")
	})

	t.Run("resolvable ref compiles cleanly", func(t *testing.T) {
		t.Parallel()
		schema := `<grammar ` + ns + `>
  <start>
    <element name="root">
      <ref name="x"/>
    </element>
  </start>
  <define name="x">
    <text/>
  </define>
</grammar>`
		require.Empty(t, compileErrorsFor(t, schema),
			"a <ref> resolving to an existing define must compile cleanly")
	})
}

// TestForbiddenNestingPerFlagContext covers 661-2 finding #2: forbidden-nesting
// checks depend on the ancestor ruleFlags context, but the visited-define cache
// was keyed by *pattern only. When the same define is reached first in an
// allowed context and again under <list>, the second check was skipped and the
// list//element error was missed. The check must fire per flag context.
func TestForbiddenNestingPerFlagContext(t *testing.T) {
	t.Parallel()

	const ns = `xmlns="http://relaxng.org/ns/structure/1.0"`

	// The same <define name="x"> (which contains an <element>) is referenced
	// twice: once in a normal content position (allowed) and once inside a
	// <list> (forbidden: list//element). The list-context reference must still
	// report the forbidden nesting even though the define was already visited in
	// the allowed context.
	schema := `<grammar ` + ns + `>
  <start>
    <element name="root">
      <group>
        <ref name="x"/>
        <list>
          <ref name="x"/>
        </list>
      </group>
    </element>
  </start>
  <define name="x">
    <element name="child"><text/></element>
  </define>
</grammar>`

	require.Contains(t, compileErrorsFor(t, schema), "list//element",
		"a define reached under <list> must report list//element even if already visited in an allowed context")
}
