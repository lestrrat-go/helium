package xslt3_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestStripSpacePreservesExternalDTDIDs verifies that running a transform whose
// stylesheet declares xsl:strip-space keeps id()/GetElementByID working for IDs
// declared in an EXTERNAL DTD subset.
//
// The lazy GetElementByID fallback (document.GetElementByID) walks BOTH the
// internal AND external DTD subsets when resolving ID-typed attributes. Without
// strip-space the transform runs over the original source, whose external subset
// (extSubset) is present, so id('x') resolves to the <item> element. With
// strip-space the transform runs over copyAndStrip's private copy; that copy
// drops the source ID table, so id() must fall back to the DTD walk. Before the
// fix copyAndStrip only carried over the INTERNAL subset (via CopyDTDInfo) and
// lost extSubset, so the copy's id('x') resolved to nothing and the two paths
// disagreed. See finding codex 664-3.
func TestStripSpacePreservesExternalDTDIDs(t *testing.T) {
	t.Parallel()

	// The ID attribute is declared ONLY in the external DTD, so resolving id('x')
	// requires consulting extSubset.
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

	parseSource := func() *helium.Document {
		src, err := helium.NewParser().
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(source))
		require.NoError(t, err)
		return src
	}

	// Sanity: the external subset really is what carries the ID declaration, so
	// the source itself resolves id('x') to the <item> element.
	require.NotNil(t, parseSource().GetElementByID("x"),
		"external-DTD-declared ID must resolve on the source document")

	// The stylesheet emits the local name of id('x') (or "none"), revealing the
	// ID semantics of the (possibly copied) source the transform runs over.
	stylesheet := func(withStrip bool) string {
		strip := ""
		if withStrip {
			strip = `  <xsl:strip-space elements="*"/>` + "\n"
		}
		return `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
` + strip + `  <xsl:output method="text"/>
  <xsl:template match="/">
    <xsl:value-of select="if (id('x')) then local-name(id('x')) else 'none'"/>
  </xsl:template>
</xsl:stylesheet>`
	}

	// Baseline: without strip-space the transform runs over the original source,
	// whose extSubset resolves id('x') to <item>.
	noStripSS, err := xslt3.NewCompiler().Compile(t.Context(),
		mustParse(t, stylesheet(false)))
	require.NoError(t, err)
	noStripOut, err := xslt3.TransformString(t.Context(), parseSource(), noStripSS)
	require.NoError(t, err)
	require.Equal(t, "item", noStripOut,
		"baseline: external-DTD ID resolves without strip-space")

	// With strip-space the transform runs over copyAndStrip's copy, which must
	// carry over extSubset so id('x') resolves identically to the baseline.
	stripSS, err := xslt3.NewCompiler().Compile(t.Context(),
		mustParse(t, stylesheet(true)))
	require.NoError(t, err)
	stripOut, err := xslt3.TransformString(t.Context(), parseSource(), stripSS)
	require.NoError(t, err)
	require.Equal(t, "item", stripOut,
		"xsl:strip-space copy must preserve the source external DTD subset so id('x') still resolves")
}
