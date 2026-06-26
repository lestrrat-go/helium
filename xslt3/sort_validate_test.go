package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// ENG-003: an evaluated xsl:sort order that is neither "ascending" nor
// "descending" must raise XTDE0030 instead of silently sorting ascending.
func TestSortInvalidOrderRaisesXTDE0030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="." order="bogus"/>
        <xsl:value-of select="."/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item>b</item><item>a</item></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE0030")
}

// ENG-003: an evaluated xsl:sort case-order that is neither "upper-first"
// nor "lower-first" must raise XTDE0030.
func TestSortInvalidCaseOrderRaisesXTDE0030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="." case-order="bogus"/>
        <xsl:value-of select="."/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item>b</item><item>a</item></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE0030")
}

// ENG-003: a stable AVT must be parsed and evaluated without error, and a
// valid order must still produce a correctly sorted result.
func TestSortStableAVTEvaluates(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="." stable="{if (true()) then 'yes' else 'no'}" order="ascending"/>
        <xsl:value-of select="."/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item>b</item><item>a</item><item>c</item></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">abc<")
}

// ENG-003: an invalid evaluated stable value must raise XTDE0030.
func TestSortInvalidStableRaisesXTDE0030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="." stable="{'bogus'}"/>
        <xsl:value-of select="."/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item>b</item><item>a</item></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE0030")
}

// ENG-004: xsl:perform-sort over an EMPTY sequence must still validate its
// sort keys, so an invalid collation raises XTDE1035 even with no input.
func TestPerformSortEmptyValidatesCollation(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:perform-sort select="()">
        <xsl:sort select="." collation="http://example.com/does-not-exist"/>
      </xsl:perform-sort>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE1035")
}

// XSLT3-ADV-003: an xsl:sort data-type that is neither "text"/"number" nor a
// valid QName must raise XTDE0030 even when the input sequence is empty (the
// validation must not be skipped just because there is nothing to sort).
func TestSortInvalidDataTypeEmptyRaisesXTDE0030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:perform-sort select="()">
        <xsl:sort select="." data-type="bogus"/>
      </xsl:perform-sort>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE0030")
}

// XSLT3-ADV-003: an invalid evaluated data-type with non-empty input must also
// raise XTDE0030 rather than silently falling back to text sorting.
func TestSortInvalidDataTypeRaisesXTDE0030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="." data-type="{'bogus'}"/>
        <xsl:value-of select="."/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item>b</item><item>a</item></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE0030")
}
