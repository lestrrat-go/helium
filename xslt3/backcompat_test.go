package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// transformStr compiles the stylesheet and transforms <doc/>, returning the
// serialized result.
func transformStr(t *testing.T, xsltSrc string) (string, error) {
	t.Helper()
	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	if err != nil {
		return "", err
	}
	src, err := helium.NewParser().Parse(ctx, []byte(`<doc/>`))
	require.NoError(t, err)
	return ss.Transform(src).Serialize(ctx)
}

// TestBackCompatXPathArithmetic verifies that a version="1.0" stylesheet both
// compiles/runs (no XTDE0160) and evaluates XPath in 1.0 compatibility mode:
// '3' + 4 becomes 7 rather than a type error.
func TestBackCompatXPathArithmetic(t *testing.T) {
	ss := `<?xml version="1.0"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="'3' + 4"/></out></xsl:template>
</xsl:stylesheet>`
	out, err := transformStr(t, ss)
	require.NoError(t, err)
	require.Contains(t, out, "<out>7</out>")
}

// TestBackCompatVersionGated verifies the same expression is a type error under
// version="3.0" — compatibility mode is opt-in via the version attribute.
func TestBackCompatVersionGated(t *testing.T) {
	ss := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="'3' + 4"/></out></xsl:template>
</xsl:stylesheet>`
	_, err := transformStr(t, ss)
	require.Error(t, err)
}

// TestBackCompatPerSubtreeOverride verifies that an inner xsl:version on a
// literal result element enables compatibility mode for that subtree only.
func TestBackCompatPerSubtreeOverride(t *testing.T) {
	ss := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out xsl:version="1.0"><xsl:value-of select="'3' + 4"/></out>
  </xsl:template>
</xsl:stylesheet>`
	out, err := transformStr(t, ss)
	require.NoError(t, err)
	require.Contains(t, out, "7")
}

// TestBackCompatSupportsProperty verifies system-property reports support.
func TestBackCompatSupportsProperty(t *testing.T) {
	ss := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="system-property('xsl:supports-backwards-compatibility')"/></out></xsl:template>
</xsl:stylesheet>`
	out, err := transformStr(t, ss)
	require.NoError(t, err)
	require.Contains(t, out, "<out>yes</out>")
}
