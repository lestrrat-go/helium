package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestCommentBodyNoStraySpace verifies that xsl:comment body construction
// does not produce a stray leading space when an empty TVT precedes text.
func TestCommentBodyNoStraySpace(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="empty" select="''"/>
  <xsl:template match="/">
    <out>
      <xsl:comment>
        <xsl:value-of select="$empty"/>
        <xsl:text>hello</xsl:text>
      </xsl:comment>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	// The comment content should be "hello" with no leading space.
	require.Contains(t, out, "<!--hello-->")
}

// TestPIBodyNoStraySpace verifies that xsl:processing-instruction body
// does not produce a stray leading space when an empty TVT precedes text.
func TestPIBodyNoStraySpace(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="empty" select="''"/>
  <xsl:template match="/">
    <out>
      <xsl:processing-instruction name="target">
        <xsl:value-of select="$empty"/>
        <xsl:text>data</xsl:text>
      </xsl:processing-instruction>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	// The PI content should be "data" with no leading space.
	require.Contains(t, out, "<?target data?>")
}
