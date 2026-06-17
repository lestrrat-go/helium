package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestAttributeUndeclaredPrefixSequenceMode verifies that xsl:attribute with a
// computed name using an undeclared prefix raises XTDE0860 even when the
// attribute is constructed in sequence mode (xsl:variable/xsl:param with an
// "as" type), rather than being captured silently as a no-namespace attribute.
func TestAttributeUndeclaredPrefixSequenceMode(t *testing.T) {
	ctx := t.Context()

	// The variable has an "as" type, so xsl:attribute is constructed in
	// sequence mode. The computed name "p:a" uses prefix "p" which is not
	// declared anywhere in scope, so XTDE0860 must be raised.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="v" as="attribute()*">
      <xsl:attribute name="{'p:a'}" select="'x'"/>
    </xsl:variable>
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	_, err = ss.Transform(src).Serialize(ctx)
	require.Error(t, err, "undeclared prefix in computed attribute name must raise an error")
	require.True(t, strings.Contains(err.Error(), "XTDE0860"),
		"expected XTDE0860, got: %v", err)
}
