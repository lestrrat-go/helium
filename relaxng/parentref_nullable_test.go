package relaxng_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParentRefNullable covers 661-1: isNullable (and patternElementName) must
// follow <parentRef> through its scoped resolved pointer, exactly like <ref>.
//
// The inner grammar's content is an <interleave> of a <parentRef name="e"/> and
// an <element name="a">. The parent define "e" resolves to <empty/>, so the
// parentRef is nullable. The interleave can therefore complete by matching only
// <a/>. If isNullable does not follow parentRef, the empty branch is treated as
// non-nullable and the valid document is wrongly rejected.
func TestParentRefNullable(t *testing.T) {
	t.Parallel()

	const ns = `xmlns="http://relaxng.org/ns/structure/1.0"`

	schema := `<grammar ` + ns + `>
  <start>
    <element name="root">
      <grammar>
        <start>
          <element name="wrap">
            <interleave>
              <parentRef name="e"/>
              <element name="a"><text/></element>
            </interleave>
          </element>
        </start>
      </grammar>
    </element>
  </start>
  <define name="e">
    <empty/>
  </define>
</grammar>`

	err := validateWith(t, schema, `<root><wrap><a>hi</a></wrap></root>`)
	require.NoError(t, err, "parentRef resolving to <empty/> must be nullable")
}
