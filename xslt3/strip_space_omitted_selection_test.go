package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestStripSpaceDropsOmittedSelectionNodes verifies that when an initial match
// selection mixes a whitespace-only text node (which strip-space OMITS from the
// copy) with a real element child, the omitted node is DROPPED from the remapped
// selection rather than passed through pointing at the unstripped original. The
// apply loop must then compute position()/last() from the filtered sequence.
//
// The selection /*/text()[1] | /*/child selects the leading whitespace-only text
// node and the <child> element. After strip-space the whitespace text node has no
// copy, so the remapped selection should contain only <child>: position()=1,
// last()=1. Before the fix the omitted node was passed through, so the selection
// kept length 2 and <child> reported position()=2, last()=2. See finding 664-6.
func TestStripSpaceDropsOmittedSelectionNodes(t *testing.T) {
	t.Parallel()

	main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="child"><out><xsl:value-of select="position()"/>/<xsl:value-of select="last()"/></out></xsl:template>
  <xsl:template match="text()"/>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	// The root element has a leading whitespace-only text node (stripped) followed
	// by the <child> element.
	source, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root>`+"\n  <child/>\n"+`</root>`))
	require.NoError(t, err)

	sel := evalSelection(t, "/*/text()[1] | /*/child", source)
	require.Equal(t, 2, sel.Len(), "fixture must select the whitespace text node and the child")

	out, err := ss.ApplyTemplates(source).
		Selection(sel).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "<out>1/1</out>",
		"omitted whitespace node must be dropped so child is position 1 of 1; got %q", out)
	require.NotContains(t, out, "<out>2/2</out>",
		"the stripped selection must not retain the omitted whitespace node; got %q", out)
}
