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

// XSLT3-ADV-003: a NON-EMPTY atomic xsl:for-each (atomic-sequence sort path)
// must validate its sort-key data-type and raise XTDE0030, not silently sort
// the atomic items as text.
func TestForEachAtomicInvalidDataTypeRaisesXTDE0030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="('b','a','c')">
        <xsl:sort select="." data-type="bogus"/>
        <xsl:value-of select="."/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
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

// XSLT3-102: in a multi-key xsl:sort, a SECONDARY sort key whose atomic values
// are of mutually incomparable types across the sorted sequence (here xs:date
// vs xs:integer) must raise XTDE1030, just as the single-key path does. The
// primary key is uniform so the failure can only come from the second key.
func TestMultiKeySortIncompatibleSecondKeyRaisesXTDE1030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="@g"/>
        <xsl:sort select="if (@t='d') then xs:date(@v) else xs:integer(@v)"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item g="x" t="d" v="2020-01-01"/><item g="x" t="n" v="5"/></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE1030")
}

// XSLT3-102 r2: a SECONDARY sort key with explicit data-type="text" stringifies
// every value before comparison, so mutually incomparable original atomic types
// (here xs:date vs xs:integer) are perfectly valid and must NOT raise XTDE1030.
// The per-level type-consistency check only applies to default-data-type levels.
func TestMultiKeySortTextSecondKeyMixedTypesNoError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="@g"/>
        <xsl:sort select="if (@t='d') then xs:date(@v) else xs:integer(@v)" data-type="text"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item g="x" t="d" v="2020-01-01"/><item g="x" t="n" v="5"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out")
}

// XSLT3-102 r3: a SINGLE-key sort with explicit data-type="text" stringifies
// every value, so mutually incomparable original atomic types (xs:date vs
// xs:integer) are perfectly valid and must NOT raise XTDE1030. The single-key
// path must skip the type-consistency check for explicit text levels exactly
// like the multi-key path does.
func TestSingleKeyTextSortMixedTypesNoError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="if (@t='d') then xs:date(@v) else xs:integer(@v)" data-type="text"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item t="d" v="2020-01-01"/><item t="n" v="5"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out")
}

// XSLT3-102 r3: a default-data-type sort mixing xs:dateTimeStamp and xs:dateTime
// must NOT raise XTDE1030. In the repo's XSD 1.1 model xs:dateTimeStamp is a
// subtype of xs:dateTime and the two are mutually comparable with the value
// comparison `lt`/`eq` operators, so the consistency check must accept them
// rather than rejecting on raw type-name inequality.
func TestSingleKeySortDateTimeStampDateTimeNoError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="if (@t='s') then xs:dateTimeStamp(@v) else xs:dateTime(@v)"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item t="s" v="2020-01-01T00:00:00Z"/><item t="d" v="2019-06-01T12:00:00"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "2019-06-01T12:00:00")
}

// XSLT3-102 r3: a default-data-type sort mixing xs:anyURI and xs:string must NOT
// raise XTDE1030. Both belong to the string comparison family and are mutually
// comparable, so the consistency check must accept them.
func TestSingleKeySortAnyURIStringNoError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="if (@t='u') then xs:anyURI(@v) else string(@v)"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item t="s" v="zebra"/><item t="u" v="apple"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out")
}

// XSLT3-102 r3: a default-data-type single-key sort mixing genuinely
// incomparable atomic types (xs:date vs xs:integer, different orderability
// families) must still raise XTDE1030.
func TestSingleKeySortIncompatibleTypesRaisesXTDE1030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="if (@t='d') then xs:date(@v) else xs:integer(@v)"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item t="d" v="2020-01-01"/><item t="n" v="5"/></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE1030")
}

// XSLT3-102 r2: likewise, a SECONDARY sort key with explicit data-type="number"
// casts every value to xs:double, so mixed original atomic types (xs:date vs
// xs:integer here, which belong to different orderability families) are
// comparable and must NOT raise XTDE1030.
func TestMultiKeySortNumberSecondKeyMixedTypesNoError(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="@g"/>
        <xsl:sort select="if (@t='d') then xs:date(@v) else xs:integer(@v)" data-type="number"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item g="x" t="d" v="2020-01-01"/><item g="x" t="i" v="5"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out")
}
