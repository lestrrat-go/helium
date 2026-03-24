package xslt3_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

const testSourceXML = `<root/>`

func compileStylesheetString(t *testing.T, src string) *xslt3.Stylesheet {
	t.Helper()

	doc, err := helium.Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	ss, err := xslt3.CompileStylesheet(t.Context(), doc)
	require.NoError(t, err)
	return ss
}

func parseTransformSource(t *testing.T) *helium.Document {
	t.Helper()

	doc, err := helium.Parse(t.Context(), []byte(testSourceXML))
	require.NoError(t, err)
	return doc
}

func TestCallTemplatePreservesParameters(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="x" select="'default-x'"/>
  <xsl:template match="/">
    <out>root</out>
  </xsl:template>
  <xsl:template name="t">
    <out><xsl:value-of select="$x"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	result, err := ss.CallTemplate("t").
		SourceDocument(source).
		SetParameter("x", xpath3.SingleString("one")).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>one</out>")
}

func TestInvocationCopyOnWriteParameters(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="x" select="'default-x'"/>
  <xsl:param name="y" select="'default-y'"/>
  <xsl:template match="/">
    <out><xsl:value-of select="concat($x, '|', $y)"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	base := ss.Transform(source).SetParameter("x", xpath3.SingleString("one"))
	derived := base.SetParameter("y", xpath3.SingleString("two"))

	baseResult, err := base.Serialize(t.Context())
	require.NoError(t, err)
	require.True(t, strings.Contains(baseResult, "<out>one|default-y</out>"), baseResult)

	derivedResult, err := derived.Serialize(t.Context())
	require.NoError(t, err)
	require.True(t, strings.Contains(derivedResult, "<out>one|two</out>"), derivedResult)
}
