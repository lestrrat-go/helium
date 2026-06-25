package xslt3_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// countNotations returns the number of notation-declaration children directly
// under the given DTD.
func countNotations(dtd *helium.DTD) int {
	if dtd == nil {
		return 0
	}
	count := 0
	for c := dtd.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.NotationNode {
			count++
		}
	}
	return count
}

// TestStripSpaceCopyExternalSubsetIsolated verifies that the strip-space copy
// owns an INDEPENDENT external DTD subset: mutating the copy's external subset
// (via a *DTD mutator) must NOT affect the source document's external subset.
//
// Before the fix, copyAndStrip shared the source's extSubset by pointer into the
// copy. Because the copy can be exposed to user code (raw-result capture) and
// *DTD has mutators, a handler mutating the copy's ExtSubset would corrupt the
// source. The fix deep-copies the external subset so the two are fully isolated.
// See finding codex 664-2 (extSubset aliasing).
func TestStripSpaceCopyExternalSubsetIsolated(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"ext.dtd": {Data: []byte(
			`<!ELEMENT doc (item*)>` + "\n" +
				`<!ELEMENT item (#PCDATA)>` + "\n" +
				`<!ATTLIST item eid ID #IMPLIED>`)},
	}
	const source = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc>
  <item eid="x">item</item>
</doc>`

	src, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(source))
	require.NoError(t, err)
	require.NotNil(t, src.ExtSubset(), "source must have an external subset")

	// Sanity: the source resolves the external-DTD-declared ID.
	require.NotNil(t, src.GetElementByID("x"))

	srcNotationsBefore := countNotations(src.ExtSubset())

	// Produce the strip-space copy and confirm it has its own external subset
	// that still resolves the external-DTD ID (round-3 behavior preserved).
	cp, err := xslt3.CopyAndStripForTest(src)
	require.NoError(t, err)
	require.NotNil(t, cp.ExtSubset(), "copy must carry over an external subset")
	require.NotSame(t, src.ExtSubset(), cp.ExtSubset(),
		"copy's external subset must be an independent *DTD, not the shared source pointer")
	require.NotNil(t, cp.GetElementByID("x"),
		"external-DTD ID must still resolve on the strip-space copy")

	// Mutate the COPY's external subset via a *DTD mutator. This must not touch
	// the source's external subset.
	_, err = cp.ExtSubset().AddNotation("injected", "", "injected.dtd")
	require.NoError(t, err)
	require.Positive(t, countNotations(cp.ExtSubset()),
		"mutation must register on the copy's external subset")

	require.Equal(t, srcNotationsBefore, countNotations(src.ExtSubset()),
		"mutating the copy's external subset must NOT change the source's external subset")
	_, found := src.ExtSubset().LookupNotation("injected")
	require.False(t, found,
		"notation added to the copy must not appear in the source's external subset")

	// And the deep copy must reproduce the ID-typed attribute declaration so id()
	// resolution truly comes from the copy's OWN subset (not residual sharing).
	adecls := cp.ExtSubset().AttributesForElement("item")
	require.NotEmpty(t, adecls, "copy's external subset must contain the item attribute decls")
	foundID := false
	for _, a := range adecls {
		if a.AType() == enum.AttrID {
			foundID = true
		}
	}
	require.True(t, foundID, "copy's external subset must contain the ID-typed attribute declaration")
}
