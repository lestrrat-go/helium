package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestSelfRecursiveKeyUseReturnsEmptyDuringBuild(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="self" match="root" use="string(count(key('self', '0')))"/>
  <xsl:template match="/">
    <out><xsl:value-of select="count(key('self', '0'))"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	result, err := xslt3.TransformString(t.Context(), parseTransformSource(t), ss)
	require.NoError(t, err)
	require.Contains(t, result, "<out>1</out>")
}

func TestKeyBasicLookup(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="items" match="item" use="@id"/>
  <xsl:template match="/">
    <out><xsl:value-of select="key('items', 'a')/@val"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item id="a" val="hello"/><item id="b" val="world"/></root>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), src, ss)
	require.NoError(t, err)
	t.Logf("result: %s", result)
	require.Contains(t, result, "<out>hello</out>")
}

func TestKeyInForEachSelect(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="items" match="item" use="@cat"/>
  <xsl:template match="root">
    <out><xsl:value-of select="count(key('items','a'))"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item cat="a"/><item cat="b"/><item cat="a"/></root>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), src, ss)
	require.NoError(t, err)
	t.Logf("result: %s", result)
	require.Contains(t, result, "<out>2</out>")
}

func TestKeyWithGenerateId(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="items" match="item" use="@cat"/>
  <xsl:template match="root">
    <out>
      <xsl:for-each select="item[generate-id() = generate-id(key('items',@cat)[1])]">
        <g cat="{@cat}"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item cat="a"/><item cat="b"/><item cat="a"/></root>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), src, ss)
	require.NoError(t, err)
	t.Logf("result: %s", result)
	require.Contains(t, result, `cat="a"`)
	require.Contains(t, result, `cat="b"`)
}

func TestKeyInPredicate(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="items" match="item" use="@cat"/>
  <xsl:template match="root">
    <out><xsl:for-each select="item[key('items',@cat)]"><x/></xsl:for-each></out>
  </xsl:template>
</xsl:stylesheet>`)

	src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item cat="a"/><item cat="b"/><item cat="a"/></root>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), src, ss)
	require.NoError(t, err)
	t.Logf("result: %s", result)
	// All 3 items should match since key('items', @cat) returns non-empty for all
	require.Contains(t, result, "<x/><x/><x/>")
}

func TestMutuallyRecursiveKeysDoNotOverflow(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="a" match="root" use="string(count(key('b', '0')))"/>
  <xsl:key name="b" match="root" use="string(count(key('a', '0')))"/>
  <xsl:template match="/">
    <out><xsl:value-of select="count(key('a', '1'))"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	result, err := xslt3.TransformString(t.Context(), parseTransformSource(t), ss)
	require.NoError(t, err)
	require.Contains(t, result, "<out>1</out>")
}
