package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestApplyTemplatesMixedSelectionOrder verifies that xsl:apply-templates
// processes a mixed sequence of atomic values and nodes in sequence order,
// not by processing all nodes before all atomic values. Per XSLT 3.0, the
// selected sequence is processed in order.
func TestApplyTemplatesMixedSelectionOrder(t *testing.T) {
	ctx := t.Context()

	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out><xsl:apply-templates select="('a', /root/b, 'c')"/></out>
  </xsl:template>
  <xsl:template match=".[. instance of xs:string]">[str:<xsl:value-of select="."/>]</xsl:template>
  <xsl:template match="b">[node:<xsl:value-of select="."/>]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root><b>B</b></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	// Sequence order is: 'a', /root/b, 'c'. The buggy implementation emits
	// all nodes first, producing [node:B][str:a][str:c].
	require.Contains(t, out, `[str:a][node:B][str:c]`)
}
