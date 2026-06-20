package relaxng_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRefParentRefScoping covers D-RNG-001: <ref> and <parentRef> must resolve
// against per-grammar lexical scopes, not a single flat global define map.
//
// Both the outer and the inner grammar define "x", but to DIFFERENT patterns.
// Inside the inner grammar:
//   - <ref name="x"/>        must select the INNER definition of x.
//   - <parentRef name="x"/>  must select the OUTER definition of x.
//
// With a flat global map keyed by name, one definition silently overwrites the
// other and the wrong scope is selected — accepting invalid documents or
// rejecting valid ones.
func TestRefParentRefScoping(t *testing.T) {
	t.Parallel()

	const ns = `xmlns="http://relaxng.org/ns/structure/1.0"`

	t.Run("ref resolves in current grammar scope", func(t *testing.T) {
		t.Parallel()
		// Outer x = <outerx>; inner x = <innerx>. The inner grammar's start
		// uses <ref name="x"/>, which must be the inner x (<innerx>).
		schema := `<grammar ` + ns + `>
  <start>
    <element name="root">
      <grammar>
        <start>
          <element name="wrap">
            <ref name="x"/>
          </element>
        </start>
        <define name="x">
          <element name="innerx"><text/></element>
        </define>
      </grammar>
    </element>
  </start>
  <define name="x">
    <element name="outerx"><text/></element>
  </define>
</grammar>`

		err := validateWith(t, schema, `<root><wrap><innerx>hi</innerx></wrap></root>`)
		require.NoError(t, err, "inner <ref name=x> must select the inner x (<innerx>)")

		err = validateWith(t, schema, `<root><wrap><outerx>hi</outerx></wrap></root>`)
		require.Error(t, err, "inner <ref name=x> must NOT select the outer x (<outerx>)")
	})

	t.Run("parentRef resolves in parent grammar scope", func(t *testing.T) {
		t.Parallel()
		// The inner grammar's start uses <parentRef name="x"/>, which must be
		// the OUTER x (<outerx>), not the inner x (<innerx>).
		schema := `<grammar ` + ns + `>
  <start>
    <element name="root">
      <grammar>
        <start>
          <element name="wrap">
            <parentRef name="x"/>
          </element>
        </start>
        <define name="x">
          <element name="innerx"><text/></element>
        </define>
      </grammar>
    </element>
  </start>
  <define name="x">
    <element name="outerx"><text/></element>
  </define>
</grammar>`

		err := validateWith(t, schema, `<root><wrap><outerx>hi</outerx></wrap></root>`)
		require.NoError(t, err, "inner <parentRef name=x> must select the outer x (<outerx>)")

		err = validateWith(t, schema, `<root><wrap><innerx>hi</innerx></wrap></root>`)
		require.Error(t, err, "inner <parentRef name=x> must NOT select the inner x (<innerx>)")
	})
}
