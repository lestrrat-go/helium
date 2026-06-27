package relaxng_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// TestIncludeOverrideContentTypeCheck covers the RELAX NG §7.2 content-type pass
// running over the LIVE grammar after include-override deletion/resolution, not
// over a parse-time append-only list of every parsed <element>. An <include>
// override that REPLACES a define changes which patterns are content-type
// checked: the overriding define is checked; the overridden one is not.
//
// The overridden define is contributed by a NESTED <include> (main -> b -> c),
// so it is actually parsed into the live grammar scope before the outer
// override deletes it — the case the older parse-time list mishandled by
// flagging now-dead schema content.
func TestIncludeOverrideContentTypeCheck(t *testing.T) {
	t.Parallel()

	const ns = `xmlns="http://relaxng.org/ns/structure/1.0"`

	// badElement has a content-type error (simple <data> grouped with complex
	// <element> content); goodElement is valid element-only content.
	const badElement = `<element name="x"><group><data type="string"/><element name="child"><text/></element></group></element>`
	const goodElement = `<element name="x"><text/></element>`

	t.Run("overridden content-type-bad define is not checked", func(t *testing.T) {
		t.Parallel()
		// c.rng defines x with a content-type-BAD element body. b.rng pulls it in
		// via a nested <include> (no override), so x is parsed into the live
		// scope. main.rng then OVERRIDES x with a valid element body, deleting the
		// nested define. The dead (overridden) bad element must NOT be checked.
		c := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <define name="x">` + badElement + `</define>
</grammar>`

		b := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <include href="c.rng"/>
</grammar>`

		main := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <include href="b.rng">
    <define name="x">` + goodElement + `</define>
  </include>
  <start>
    <element name="root"><ref name="x"/></element>
  </start>
</grammar>`

		fsys := fstest.MapFS{
			testMainRNGPath: &fstest.MapFile{Data: []byte(main)},
			"schemas/b.rng": &fstest.MapFile{Data: []byte(b)},
			"schemas/c.rng": &fstest.MapFile{Data: []byte(c)},
		}
		got := compileIncludeErrors(t, fsys)
		require.Empty(t, got,
			"an overridden (dead) define's content-type-bad element must not be checked; got: %s", got)
	})

	t.Run("overriding content-type-bad define is checked", func(t *testing.T) {
		t.Parallel()
		// Mirror image: the nested include contributes a VALID x, and the outer
		// override REPLACES it with a content-type-BAD element body. The define
		// that replaced it must be checked, so compilation must fail.
		c := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <define name="x">` + goodElement + `</define>
</grammar>`

		b := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <include href="c.rng"/>
</grammar>`

		main := `<?xml version="1.0"?>
<grammar ` + ns + `>
  <include href="b.rng">
    <define name="x">` + badElement + `</define>
  </include>
  <start>
    <element name="root"><ref name="x"/></element>
  </start>
</grammar>`

		fsys := fstest.MapFS{
			testMainRNGPath: &fstest.MapFile{Data: []byte(main)},
			"schemas/b.rng": &fstest.MapFile{Data: []byte(b)},
			"schemas/c.rng": &fstest.MapFile{Data: []byte(c)},
		}
		got := compileIncludeErrors(t, fsys)
		require.Contains(t, got, "content type error",
			"the overriding (live) define's content-type-bad element must be checked; got: %s", got)
	})
}
