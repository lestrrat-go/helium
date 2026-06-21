package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestStripSpaceEmptyRemappedSelectionProducesNoOutput verifies that an initial
// match selection that is ENTIRELY removed by strip-space remaps to an empty
// sequence and produces NO output — it must not fall through to applying
// templates to the source document.
//
// The selection /*/text() picks the single whitespace-only text node under
// <root>. With xsl:strip-space that node has no copy, so the remapped selection
// becomes empty (length 0). apply-templates over an empty sequence emits
// nothing. Before the fix, the zero-length remapped selection was treated as
// "no selection supplied", so the transform fell through to the source document
// and wrongly invoked the "/" root template, emitting <wrong/>. See finding
// 664-1.
func TestStripSpaceEmptyRemappedSelectionProducesNoOutput(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/"><wrong/></xsl:template>
  <xsl:template match="text()"><kept/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	// <root> contains only a whitespace-only text node, which strip-space removes.
	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root>`+"\n  \n"+`</root>`))
	require.NoError(t, err)

	sel := evalSelection(t, "/*/text()", source)
	require.Equal(t, 1, sel.Len(), "fixture must select the single whitespace-only text node")

	out, err := ss.ApplyTemplates(source).
		Selection(sel).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Empty(t, out,
		"a fully-stripped initial selection must produce no output; got %q", out)
	require.NotContains(t, out, "<wrong/>",
		"must not fall through to the source-document root template; got %q", out)
}
