package xslt3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrimaryResultDocumentAdaptiveMarkupTemporaryFrames(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		attributes string
	}{
		{
			name:       "ValidationLax",
			attributes: `validation="lax" method="adaptive" item-separator=" | "`,
		},
		{
			name:       "TypeNamedFormat",
			output:     `<xsl:output name="adaptive" method="adaptive" item-separator=" | "/>`,
			attributes: `type="xs:untyped" format="adaptive"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  `+tt.output+`
  <xsl:template match="/">
    <xsl:result-document `+tt.attributes+`>
      <xsl:comment select="'first'"/>
      <xsl:processing-instruction name="target" select="'data'"/>
      <xsl:comment select="'last'"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

			out, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			require.NoError(t, err)
			require.Equal(t, "<!--first--> | <?target data?> | <!--last-->", out)
			require.NotContains(t, out, "<?xml")
		})
	}
}

func TestPrimaryResultDocumentAdaptiveMarkupTemporaryFramesInvalidCharacter(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		attributes string
	}{
		{
			name:       "ValidationLax",
			attributes: `validation="lax" method="adaptive" item-separator=" | "`,
		},
		{
			name:       "TypeNamedFormat",
			output:     `<xsl:output name="adaptive" method="adaptive" item-separator=" | "/>`,
			attributes: `type="xs:untyped" format="adaptive"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  `+tt.output+`
  <xsl:template match="/">
    <xsl:result-document `+tt.attributes+`>
      <xsl:comment select="'first'"/>
      <xsl:processing-instruction name="target" select="codepoints-to-string(1)"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

			_, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
			requireSERE0006(t, err)
		})
	}
}
