package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestCallTemplateCoercesParamsToDeclaredTypes(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:call-template name="show">
        <xsl:with-param name="a" select="xs:untypedAtomic('FOO')"/>
        <xsl:with-param name="c" select="xs:untypedAtomic('50')"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="show">
    <xsl:param name="a" as="xs:string"/>
    <xsl:param name="c" as="xs:double"/>
    <q a="{$a instance of xs:string}" c="{$c instance of xs:double}"/>
  </xsl:template>
</xsl:stylesheet>`)

	source := parseTransformSource(t)
	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, result, `a="true"`)
	require.Contains(t, result, `c="true"`)
}

// TestGlobalParamStaticBaseURI verifies that static-base-uri() inside a
// global param's select resolves against the declaration-site xml:base,
// not the stylesheet's base URI.
func TestGlobalParamStaticBaseURI(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="base" xml:base="http://example.com/override/"
      select="static-base-uri()"/>
  <xsl:template match="/">
    <out><xsl:value-of select="$base"/></out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "http://example.com/override/")
}

// TestCommentBodyNoStraySpace verifies that xsl:comment body construction
// does not produce a stray leading space when an empty TVT precedes text.
func TestCommentBodyNoStraySpace(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="empty" select="''"/>
  <xsl:template match="/">
    <out>
      <xsl:comment>
        <xsl:value-of select="$empty"/>
        <xsl:text>hello</xsl:text>
      </xsl:comment>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	// The comment content should be "hello" with no leading space.
	require.Contains(t, out, "<!--hello-->")
}

// TestPIBodyNoStraySpace verifies that xsl:processing-instruction body
// does not produce a stray leading space when an empty TVT precedes text.
func TestPIBodyNoStraySpace(t *testing.T) {
	ctx := t.Context()
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="empty" select="''"/>
  <xsl:template match="/">
    <out>
      <xsl:processing-instruction name="target">
        <xsl:value-of select="$empty"/>
        <xsl:text>data</xsl:text>
      </xsl:processing-instruction>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)
	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	// The PI content should be "data" with no leading space.
	require.Contains(t, out, "<?target data?>")
}

func TestAnnotateAttrRegistersIDSubtype(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
      <xs:simpleType name="myID">
        <xs:restriction base="xs:ID"/>
      </xs:simpleType>
      <xs:complexType name="rootType">
        <xs:attribute name="id" type="myID"/>
      </xs:complexType>
      <xs:element name="root" type="rootType"/>
    </xs:schema>
  </xsl:import-schema>

  <xsl:template match="/">
    <result>
      <found><xsl:value-of select="boolean(id('alpha'))"/></found>
      <name><xsl:value-of select="id('alpha')/name()"/></name>
    </result>
  </xsl:template>
</xsl:stylesheet>`)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<root id="alpha"/>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	require.Contains(t, result, "<found>true</found>")
	require.Contains(t, result, "<name>root</name>")
}
