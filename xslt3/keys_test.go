package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestKey(t *testing.T) {
	t.Run("basic lookup", func(t *testing.T) {
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
	})

	t.Run("in a for-each select", func(t *testing.T) {
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
	})

	t.Run("with generate-id()", func(t *testing.T) {
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
	})

	t.Run("in predicate", func(t *testing.T) {
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
	})

	t.Run("empty name arg returns error", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="items" match="item" use="@id"/>
  <xsl:template match="/">
    <out><xsl:value-of select="key((), 'v')"/></out>
  </xsl:template>
</xsl:stylesheet>`)

		src, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item id="v"/></root>`))
		require.NoError(t, err)

		// An empty sequence for the key name must produce a dynamic type error,
		// not an index-out-of-range panic.
		_, err = xslt3.TransformString(t.Context(), src, ss)
		require.Error(t, err)
	})

	t.Run("a self-recursive key use returns empty during the build", func(t *testing.T) {
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
	})

	t.Run("a canonical key uses the QName value space", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:key name="byType" match="item" use="resolve-QName(@type, .)"/>
  <xsl:template match="/">
    <result>
      <count><xsl:value-of select="count(key('byType', resolve-QName('one:mp3', /root/item[1])))"/></count>
    </result>
  </xsl:template>
</xsl:stylesheet>`)

		source, err := helium.NewParser().Parse(t.Context(), []byte(
			`<root>`+
				`<item xmlns:one="urn:test" type="one:mp3"/>`+
				`<item xmlns:two="urn:test" type="two:mp3"/>`+
				`</root>`))
		require.NoError(t, err)

		result, err := xslt3.TransformString(t.Context(), source, ss)
		require.NoError(t, err)

		// Both items use the same QName (urn:test, mp3), so key() must return 2.
		require.Contains(t, result, "<count>2</count>")
	})

	t.Run("mutually recursive keys do not overflow", func(t *testing.T) {
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
	})
}

// TestKeyNamespaceNodeMatching pins buildKeyTable's namespace-node indexing
// behavior against every pattern shape that decides whether a key needs to
// enumerate in-scope namespace nodes. A guard computed from the pattern
// source text (mirroring matchesAttributes) would wrongly skip "." and
// ".[pred]", which match every node including namespace nodes despite their
// source containing neither "namespace" nor "@". These cases must stay
// identical whether or not buildKeyTable skips the namespace-node walk.
func TestKeyNamespaceNodeMatching(t *testing.T) {
	const srcXML = `<root xmlns:p="urn:p"><a id="x"><b/></a></root>`

	buildStylesheet := func(match string) string {
		return `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="k" match="` + match + `" use="name()"/>
  <xsl:template match="/">
    <out p="{count(key('k','p'))}" xml="{count(key('k','xml'))}" a="{count(key('k','a'))}"/>
  </xsl:template>
</xsl:stylesheet>`
	}

	run := func(t *testing.T, match string) string {
		t.Helper()
		ss := compileStylesheetString(t, buildStylesheet(match))
		src, err := helium.NewParser().Parse(t.Context(), []byte(srcXML))
		require.NoError(t, err)
		result, err := xslt3.TransformString(t.Context(), src, ss)
		require.NoError(t, err)
		return result
	}

	t.Run("namespace-node() matches every in-scope prefix on every element", func(t *testing.T) {
		result := run(t, "namespace-node()")
		require.Contains(t, result, `p="3"`)
		require.Contains(t, result, `xml="3"`)
	})

	t.Run("*/namespace::* matches every in-scope prefix on non-root elements", func(t *testing.T) {
		result := run(t, "*/namespace::*")
		require.Contains(t, result, `p="3"`)
		require.Contains(t, result, `xml="3"`)
	})

	t.Run("dot pattern matches namespace nodes too, the case a source-text guard breaks", func(t *testing.T) {
		result := run(t, ".")
		require.Contains(t, result, `p="3"`)
		require.Contains(t, result, `xml="3"`)
	})

	t.Run("union of element and namespace-node matches both kinds", func(t *testing.T) {
		result := run(t, "element()|namespace-node()")
		require.Contains(t, result, `p="3"`)
		require.Contains(t, result, `a="1"`)
	})

	t.Run("element-only pattern never matches a namespace node", func(t *testing.T) {
		result := run(t, "item")
		require.Contains(t, result, `p="0"`)
		require.Contains(t, result, `xml="0"`)
	})
}
