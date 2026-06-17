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

// TestApplyTemplatesMixedSortPosition verifies that within an xsl:sort over a
// mixed atomic+node selection, position()/last() in the sort key reflect the
// full mixed sequence (size = number of selected items, position = 1-based
// index in the unsorted sequence) rather than a stale size of 1.
func TestApplyTemplatesMixedSortPosition(t *testing.T) {
	ctx := t.Context()

	// Selection is ('x', /root/a, 'y'), three items. Sort key = position()
	// in descending order, so the processing order must be reversed:
	// 'y' (pos 3), /root/a (pos 2), 'x' (pos 1).
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out><xsl:apply-templates select="('x', /root/a, 'y')">
      <xsl:sort select="position()" data-type="number" order="descending"/>
    </xsl:apply-templates></out>
  </xsl:template>
  <xsl:template match=".[. instance of xs:string]">[str:<xsl:value-of select="."/>]</xsl:template>
  <xsl:template match="a">[node:<xsl:value-of select="."/>]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root><a>A</a></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Contains(t, out, `[str:y][node:A][str:x]`)
}

// TestApplyTemplatesMixedSortLast verifies that last() in the sort key over a
// mixed atomic+node selection reports the full mixed sequence size.
func TestApplyTemplatesMixedSortLast(t *testing.T) {
	ctx := t.Context()

	// last() must be 3 for every item. Sort key = (last() - position()) so the
	// order is reversed: 'y' (3-3=0), /root/a (3-2=1), 'x' (3-1=2) ascending.
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out><xsl:apply-templates select="('x', /root/a, 'y')">
      <xsl:sort select="last() - position()" data-type="number"/>
    </xsl:apply-templates></out>
  </xsl:template>
  <xsl:template match=".[. instance of xs:string]">[str:<xsl:value-of select="."/>]</xsl:template>
  <xsl:template match="a">[node:<xsl:value-of select="."/>]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root><a>A</a></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Contains(t, out, `[str:y][node:A][str:x]`)
}

// TestApplyTemplatesMixedSortCurrent verifies that current() in the sort key
// resolves to the node being sorted for NODE items in a mixed selection, while
// non-node items still atomize via the context item.
func TestApplyTemplatesMixedSortCurrent(t *testing.T) {
	ctx := t.Context()

	// Selection mixes nodes and a string. Sort by current() string value so
	// node items sort by their own value via current(). Nodes a/b/c have
	// values "3","1","2"; string "0" sorts first. Expected ascending order:
	// '0', b(1), c(2), a(3).
	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out><xsl:apply-templates select="(/root/a, '0', /root/b, /root/c)">
      <xsl:sort select="current()" data-type="number"/>
    </xsl:apply-templates></out>
  </xsl:template>
  <xsl:template match=".[. instance of xs:string]">[str:<xsl:value-of select="."/>]</xsl:template>
  <xsl:template match="*">[node:<xsl:value-of select="."/>]</xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root><a>3</a><b>1</b><c>2</c></root>`))
	require.NoError(t, err)

	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)

	require.Contains(t, out, `[str:0][node:1][node:2][node:3]`)
}
