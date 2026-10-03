package xslt3_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// userFunctionSheet wraps the given declarations (xsl:function elements) and a
// root template body into a stylesheet that binds the prefix f to urn:f.
func userFunctionSheet(functions, body string) string {
	return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="urn:f" exclude-result-prefixes="xs f">
  <xsl:output method="xml" omit-xml-declaration="yes"/>
` + functions + `
  <xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
}

const userFunctionNodeBuilders = `
  <xsl:function name="f:double" as="xs:integer">
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="$n * 2"/>
  </xsl:function>
  <xsl:function name="f:make" as="element()">
    <xsl:param name="v" as="xs:string"/>
    <item v="{$v}" d="{f:double(string-length($v))}"><xsl:value-of select="upper-case($v)"/></item>
  </xsl:function>
  <xsl:function name="f:build" as="element()+">
    <xsl:param name="v" as="xs:string"/>
    <xsl:element name="built">
      <xsl:attribute name="v" select="$v"/>
      <xsl:sequence select="f:make($v)"/>
    </xsl:element>
    <xsl:sequence select="f:make(concat($v, $v))"/>
  </xsl:function>`

func TestUserFunctionCall(t *testing.T) {
	t.Run("select-only body", func(t *testing.T) {
		out, err := transformStr(t, userFunctionSheet(`
  <xsl:function name="f:label" as="xs:string">
    <xsl:param name="cat" as="xs:string"/>
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="concat(upper-case($cat), '-', $n * 2)"/>
  </xsl:function>
  <xsl:function name="f:pair">
    <xsl:param name="n"/>
    <xsl:sequence select="$n"/>
    <xsl:sequence select="$n + 1"/>
  </xsl:function>
  <xsl:function name="f:none"/>`,
			`<out><xsl:for-each select="1 to 3"><r><xsl:value-of select="f:label('c', .)"/></r></xsl:for-each>`+
				`<p><xsl:value-of select="f:pair(4)"/></p><e><xsl:value-of select="count(f:none())"/></e></out>`))
		require.NoError(t, err)
		require.Equal(t, `<out><r>C-2</r><r>C-4</r><r>C-6</r><p>4 5</p><e>0</e></out>`, out)
	})

	t.Run("recursive select-only body", func(t *testing.T) {
		out, err := transformStr(t, userFunctionSheet(`
  <xsl:function name="f:fact" as="xs:integer">
    <xsl:param name="k" as="xs:integer"/>
    <xsl:sequence select="if ($k le 1) then 1 else $k * f:fact($k - 1)"/>
  </xsl:function>
  <xsl:function name="f:countdown" as="xs:integer*">
    <xsl:param name="k" as="xs:integer"/>
    <xsl:sequence select="if ($k lt 1) then () else ($k, f:countdown($k - 1))"/>
  </xsl:function>`,
			`<out><a><xsl:value-of select="f:fact(10)"/></a><b><xsl:value-of select="f:countdown(5)"/></b></out>`))
		require.NoError(t, err)
		require.Equal(t, `<out><a>3628800</a><b>5 4 3 2 1</b></out>`, out)
	})

	t.Run("select-only body returns argument nodes", func(t *testing.T) {
		out, err := transformStr(t, userFunctionSheet(`
  <xsl:function name="f:same" as="node()*">
    <xsl:param name="n" as="node()*"/>
    <xsl:sequence select="$n"/>
  </xsl:function>
  <xsl:function name="f:attr" as="attribute()?">
    <xsl:param name="e" as="element()"/>
    <xsl:sequence select="$e/@a"/>
  </xsl:function>`,
			`<xsl:variable name="t"><e a="1"><c/></e></xsl:variable>`+
				`<out><xsl:value-of select="f:same($t/e) is $t/e"/><x><xsl:sequence select="f:attr($t/e)"/></x>`+
				`<xsl:copy-of select="f:same($t/e/c)"/></out>`))
		require.NoError(t, err)
		require.Equal(t, `<out>true<x a="1"/><c/></out>`, out)
	})

	t.Run("node-building body returns live nodes", func(t *testing.T) {
		out, err := transformStr(t, userFunctionSheet(userFunctionNodeBuilders,
			`<xsl:variable name="a" select="f:make('a')"/>`+
				`<xsl:variable name="b" select="f:build('b')"/>`+
				`<xsl:variable name="later" select="(for $i in 1 to 50 return f:double($i), f:make('zz'))"/>`+
				`<out><xsl:copy-of select="$a"/><xsl:copy-of select="$b"/>`+
				`<n><xsl:value-of select="$a/@v, $a/@d, count($b), name($b[1]), $b[1]/item/@v, $b[2]/@v"/></n>`+
				`<p><xsl:value-of select="count($a/..), count($b[1]/..)"/></p></out>`))
		require.NoError(t, err)
		require.Equal(t, `<out><item v="a" d="2">A</item>`+
			`<built v="b"><item v="b" d="2">B</item></built><item v="bb" d="4">BB</item>`+
			`<n>a 2 2 built b bb</n><p>0 0</p></out>`, out)
	})

	t.Run("call inside another function's node-building body", func(t *testing.T) {
		out, err := transformStr(t, userFunctionSheet(userFunctionNodeBuilders+`
  <xsl:function name="f:wrap" as="element()">
    <xsl:param name="v" as="xs:string"/>
    <wrap n="{f:double(3)}">
      <xsl:value-of select="f:double(string-length($v))"/>
      <xsl:sequence select="f:build($v)"/>
      <xsl:copy-of select="f:make('in')"/>
    </wrap>
  </xsl:function>
  <xsl:function name="f:outer" as="element()">
    <xsl:param name="v" as="xs:string"/>
    <xsl:sequence select="f:wrap($v)"/>
  </xsl:function>`,
			`<out><xsl:copy-of select="f:outer('q')"/><c><xsl:value-of select="count(f:outer('r')//item)"/></c></out>`))
		require.NoError(t, err)
		require.Equal(t, `<out><wrap n="6">2<built v="q"><item v="q" d="2">Q</item></built>`+
			`<item v="qq" d="4">QQ</item><item v="in" d="4">IN</item></wrap><c>3</c></out>`, out)
	})

	t.Run("caught error leaves later calls intact", func(t *testing.T) {
		out, err := transformStr(t, userFunctionSheet(`
  <xsl:function name="f:check" as="xs:integer">
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="if ($n gt 2) then error(xs:QName('f:too-big')) else $n"/>
  </xsl:function>
  <xsl:function name="f:sum" as="xs:integer">
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="if ($n le 0) then 0 else f:check($n) + f:sum($n - 1)"/>
  </xsl:function>`,
			`<out><xsl:try><xsl:value-of select="f:sum(5)"/><xsl:catch>caught</xsl:catch></xsl:try>`+
				`<s><xsl:value-of select="f:sum(2)"/></s></out>`))
		require.NoError(t, err)
		require.Equal(t, `<out>caught<s>3</s></out>`, out)
	})
}
