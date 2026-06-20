package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestStripSpacePreservesIDsSkip verifies that running a transform whose
// stylesheet declares xsl:strip-space preserves the source document's
// ID-skip state. A source parsed with SkipIDs(true) must NOT register its
// xml:id values, so fn:id('x') (and GetElementByID) returns nothing — both
// without strip-space (which transforms the original source directly) AND
// with strip-space (which transforms a private copy produced by copyAndStrip).
//
// Before the fix, copyAndStrip created a fresh document that dropped the
// source's idsSkip flag, so the copy fell back to an O(n) xml:id walk and
// fn:id('x') wrongly matched. See finding codex 664-2.
func TestStripSpacePreservesIDsSkip(t *testing.T) {
	t.Parallel()

	// The stylesheet emits "found" when id('x') resolves to an element and
	// "none" otherwise, so the transform output reveals the ID semantics of
	// the (possibly copied) source the transform actually runs over.
	idLookup := func(withStrip bool) string {
		strip := ""
		if withStrip {
			strip = `  <xsl:strip-space elements="*"/>` + "\n"
		}
		return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
` + strip + `  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="if (id('x')) then 'found' else 'none'"/>
  </xsl:template>
</xsl:stylesheet>`
	}

	parseSource := func() *helium.Document {
		// SkipIDs(true) means xml:id values are NOT interned, so id('x') must
		// not find the element.
		src, err := helium.NewParser().SkipIDs(true).Parse(t.Context(),
			[]byte(`<doc>
  <item xml:id="x">hello</item>
</doc>`))
		require.NoError(t, err)
		return src
	}

	// Baseline: without strip-space, the transform runs over the original
	// SkipIDs source directly, so id('x') finds nothing.
	noStripSS, err := xslt3.NewCompiler().Compile(t.Context(),
		mustParse(t, idLookup(false)))
	require.NoError(t, err)
	noStripOut, err := xslt3.TransformString(t.Context(), parseSource(), noStripSS)
	require.NoError(t, err)
	require.Equal(t, "none", noStripOut,
		"baseline: SkipIDs source without strip-space must not resolve id('x')")

	// With strip-space, the transform runs over copyAndStrip's copy. The copy
	// must inherit idsSkip so id('x') still finds nothing — matching baseline.
	stripSS, err := xslt3.NewCompiler().Compile(t.Context(),
		mustParse(t, idLookup(true)))
	require.NoError(t, err)
	stripOut, err := xslt3.TransformString(t.Context(), parseSource(), stripSS)
	require.NoError(t, err)
	require.Equal(t, "none", stripOut,
		"xsl:strip-space copy must preserve the source's SkipIDs state so id('x') stays unresolved")
}

func mustParse(t *testing.T, doc string) *helium.Document {
	t.Helper()
	d, err := helium.NewParser().Parse(t.Context(), []byte(doc))
	require.NoError(t, err)
	return d
}
