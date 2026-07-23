package transform_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xmldsig1/transform"
	"github.com/stretchr/testify/require"
)

// TestXSLTAppliesStylesheet proves the transformer actually runs the stylesheet
// over the input and returns its output (the bytes xmldsig1 would then digest).
func TestXSLTAppliesStylesheet(t *testing.T) {
	stylesheet := []byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">` +
		`<xsl:output method="text"/>` +
		`<xsl:template match="/">seen:<xsl:value-of select="/a/b"/></xsl:template>` +
		`</xsl:stylesheet>`)
	input := []byte(`<a><b>hi</b></a>`)

	out, err := transform.XSLT{}.TransformXSLT(t.Context(), stylesheet, input)
	require.NoError(t, err)
	require.Equal(t, "seen:hi", string(out))
}

func TestXSLTTreeOutputDropsWriterTerminator(t *testing.T) {
	stylesheet := []byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">` +
		`<xsl:output method="xml" omit-xml-declaration="yes"/>` +
		`<xsl:template match="/"><out><xsl:value-of select="/a/b"/></out></xsl:template>` +
		`</xsl:stylesheet>`)

	out, err := transform.XSLT{}.TransformXSLT(t.Context(), stylesheet, []byte(`<a><b>hi</b></a>`))
	require.NoError(t, err)
	require.Equal(t, "<out>hi</out>", string(out))
}

func TestXSLTTextOutputPreservesTrailingNewline(t *testing.T) {
	stylesheet := []byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">` +
		`<xsl:output method="text"/>` +
		`<xsl:template match="/">seen:<xsl:value-of select="/a/b"/><xsl:text>&#10;</xsl:text></xsl:template>` +
		`</xsl:stylesheet>`)

	out, err := transform.XSLT{}.TransformXSLT(t.Context(), stylesheet, []byte(`<a><b>hi</b></a>`))
	require.NoError(t, err)
	require.Equal(t, "seen:hi\n", string(out))
}

// TestXSLTRejectsMalformedStylesheet proves a non-well-formed stylesheet is an
// error rather than a silent empty transform.
func TestXSLTRejectsMalformedStylesheet(t *testing.T) {
	_, err := transform.XSLT{}.TransformXSLT(t.Context(), []byte("not a stylesheet <"), []byte(`<a/>`))
	require.Error(t, err)
}

// TestXSLTRejectsMalformedInput proves a non-well-formed input document is an
// error, not a silent pass-through.
func TestXSLTRejectsMalformedInput(t *testing.T) {
	stylesheet := []byte(`<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">` +
		`<xsl:template match="/"><xsl:copy-of select="."/></xsl:template></xsl:stylesheet>`)
	_, err := transform.XSLT{}.TransformXSLT(t.Context(), stylesheet, []byte("not xml <"))
	require.Error(t, err)
}
