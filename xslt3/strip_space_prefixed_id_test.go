package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestStripSpacePreservesPrefixedDTDIDs verifies that running a transform whose
// stylesheet declares xsl:strip-space keeps id() resolving an ID-typed attribute
// declared for a PREFIXED element (a:item) identically to the no-strip baseline.
//
// Without strip-space the transform runs over the original source, whose parser
// ID table maps "x" to the <a:item> element. With strip-space the transform runs
// over copyAndStrip's private copy. The copy used to drop the source ID table and
// rely on GetElementByID's lazy fallback, which looked up the DTD ATTLIST by the
// element's LocalName ("item") only and therefore missed the qualified ATTLIST
// for "a:item" — so id('x') resolved to <a:item> without strip-space but to
// nothing with strip-space. copyAndStrip now rebuilds the copy's ID table from
// the source's, so both paths agree. See finding codex 664-6.
func TestStripSpacePreservesPrefixedDTDIDs(t *testing.T) {
	t.Parallel()

	const source = `<?xml version="1.0"?>
<!DOCTYPE a:doc [
<!ELEMENT a:doc (a:item*)>
<!ELEMENT a:item (#PCDATA)>
<!ATTLIST a:item eid ID #IMPLIED>
<!ATTLIST a:doc xmlns:a CDATA #IMPLIED>
]>
<a:doc xmlns:a="urn:a">
  <a:item eid="x">item</a:item>
</a:doc>`

	parseSource := func() *helium.Document {
		src, err := helium.NewParser().Parse(t.Context(), []byte(source))
		require.NoError(t, err)
		return src
	}

	// Sanity: the source itself resolves the prefixed-element ID to <a:item>.
	require.NotNil(t, parseSource().GetElementByID("x"),
		"prefixed-element ID must resolve on the source document")

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

	noStripSS, err := xslt3.NewCompiler().Compile(t.Context(),
		mustParse(t, stylesheet(false)))
	require.NoError(t, err)
	noStripOut, err := xslt3.TransformString(t.Context(), parseSource(), noStripSS)
	require.NoError(t, err)
	require.Equal(t, "item", noStripOut,
		"baseline: prefixed-element ID resolves without strip-space")

	stripSS, err := xslt3.NewCompiler().Compile(t.Context(),
		mustParse(t, stylesheet(true)))
	require.NoError(t, err)
	stripOut, err := xslt3.TransformString(t.Context(), parseSource(), stripSS)
	require.NoError(t, err)
	require.Equal(t, "item", stripOut,
		"xsl:strip-space copy must resolve the prefixed-element ID identically to the baseline")
}
