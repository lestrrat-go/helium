package xslt3_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestSerializeResultXML(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><root>hello</root></xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, ss.DefaultOutputDef())
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<root>hello</root>")
}

func TestSerializeResultNilOutputDef(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><root>hello</root></xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	// nil OutputDef should use defaults.
	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, nil)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<root>hello</root>")
}

func TestSerializeResultText(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/">hello world</xsl:template>
</xsl:stylesheet>`)

	doc, err := ss.Transform(parseTransformSource(t)).Do(t.Context())
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeResult(&buf, doc, ss.DefaultOutputDef())
	require.NoError(t, err)
	require.Equal(t, "hello world", strings.TrimSpace(buf.String()))
}

func TestSerializeItemsAtomics(t *testing.T) {
	items := xpath3.ItemSlice{
		xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "alpha"},
		xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: "bravo"},
	}

	var buf bytes.Buffer
	err := xslt3.SerializeItems(&buf, items, nil, nil)
	require.NoError(t, err)
	result := buf.String()
	require.Contains(t, result, "alpha")
	require.Contains(t, result, "bravo")
}

func TestSerializeItemsWithDocument(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<data>content</data>`))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = xslt3.SerializeItems(&buf, nil, doc, nil)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "content")
}

func TestDefaultOutputDef(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text" encoding="UTF-8"/>
  <xsl:template match="/">hello</xsl:template>
</xsl:stylesheet>`)

	outDef := ss.DefaultOutputDef()
	require.NotNil(t, outDef)
}

func TestDefaultOutputDefNilStylesheet(t *testing.T) {
	var ss *xslt3.Stylesheet
	outDef := ss.DefaultOutputDef()
	require.Nil(t, outDef)
}
