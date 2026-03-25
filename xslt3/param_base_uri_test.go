package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestGlobalParamStaticBaseURI verifies that static-base-uri() inside a
// global param's select resolves against the declaration-site xml:base,
// not the stylesheet's base URI.
func TestGlobalParamStaticBaseURI(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="base" xml:base="http://example.com/override/"
      select="static-base-uri()"/>
  <xsl:template match="/">
    <out><xsl:value-of select="$base"/></out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "http://example.com/override/")
}
