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
//
// XSLT3-102 r5: with a uniform primary key the secondary text level must drive
// the order by STRING value. Document order is date-then-integer; correct text
// order is "1" < "2020-01-01", so the integer item must come first. Pre-r5 the
// date key's str was blanked to "" by the numeric rewrite and sorted first.
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
		[]byte(`<root><item g="x" t="d" v="2020-01-01"/><item g="x" t="n" v="1"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">12020-01-01<")
}

// XSLT3-102 r3: a SINGLE-key sort with explicit data-type="text" stringifies
// every value, so mutually incomparable original atomic types (xs:date vs
// xs:integer) are perfectly valid and must NOT raise XTDE1030. The single-key
// path must skip the type-consistency check for explicit text levels exactly
// like the multi-key path does.
//
// XSLT3-102 r5: the result must also be ordered by the STRING value of each
// key. Pre-r5 fillSingletonSortValue rewrote the xs:date key into a numeric
// sortValue with an empty str, so under text comparison the date key compared
// as "" (sorting before the integer's "1") and the order was wrong. Document
// order is date-then-integer; correct text order is "1" < "2020-01-01", so the
// integer must come first.
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
		[]byte(`<root><item t="d" v="2020-01-01"/><item t="n" v="1"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">12020-01-01<")
}

// XSLT3-102 r5: a default single-key sort over xs:date keys with explicit
// data-type="text" must order strictly by the canonical string value. Pre-r5
// fillSingletonSortValue rewrote every date key into a numeric sortValue and
// blanked str, so all keys compared equal ("") and the sort degenerated to
// document order. Document order here is 2020,2019,2021; correct text order is
// 2019,2020,2021.
func TestSingleKeyTextSortDateKeysOrdersByString(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="xs:date(@v)" data-type="text"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item v="2020-01-01"/><item v="2019-06-01"/><item v="2021-03-03"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">2019-06-012020-01-012021-03-03<")
}

// XSLT3-102 r5: same as the date case but for xs:dayTimeDuration keys, which
// fillSingletonSortValue rewrites through a separate duration branch. Under
// data-type="text" the canonical string value must drive ordering. value-of
// outputs the canonical duration string (the same value the sort compares) so
// the assertion is independent of any lexical-vs-canonical mismatch. Document
// order is PT3H,PT1H,PT20H; correct text order is PT1H < PT20H < PT3H.
func TestSingleKeyTextSortDurationKeysOrdersByString(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="xs:dayTimeDuration(@v)" data-type="text"/>
        <xsl:value-of select="xs:dayTimeDuration(@v)"/><xsl:text>|</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item v="PT3H"/><item v="PT1H"/><item v="PT20H"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">PT1H|PT20H|PT3H|<")
}

// XSLT3-102 r5: a default multi-key sort whose secondary level is flipped to
// auto-number mode by an earlier item (an xs:integer key) and whose later item
// supplies the key as a NODE with an incompatible atomized type (untypedAtomic
// "zzz", not castable to a number) must raise XTDE1030. Pre-r5 the
// dataTypeNumberAuto branch only recorded an atom when the singleton item was
// already an xpath3.AtomicValue, so the node key was silently skipped by
// validateSortLevelTypes and the inconsistency bypassed the type gate.
func TestMultiKeySortNumberAutoNodeKeyBypassRaisesXTDE1030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="@g"/>
        <xsl:sort select="if (@t='i') then xs:integer(@v) else @v"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item g="x" t="i" v="5"/><item g="x" t="n" v="zzz"/></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE1030")
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

// XSLT3-102 r4: a default-data-type SINGLE-key sort mixing xs:yearMonthDuration
// and xs:dayTimeDuration must raise XTDE1030. The two duration subtypes define
// `eq` but NOT `lt` (ordering between them is XPTY0004), so the orderability
// gate must reject them. Pre-fix this slipped through an equality-based oracle
// and the values were silently mashed into one numeric domain (months vs
// seconds) and sorted nonsensically.
func TestSingleKeySortMixedDurationsRaisesXTDE1030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="if (@t='y') then xs:yearMonthDuration(@v) else xs:dayTimeDuration(@v)"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item t="y" v="P1Y"/><item t="d" v="P400D"/></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE1030")
}

// XSLT3-102 r4: a default-data-type MULTI-key sort whose SECONDARY key mixes
// xs:yearMonthDuration and xs:dayTimeDuration must raise XTDE1030 for the same
// reason as the single-key case: ordering across the two duration subtypes is
// undefined (XPTY0004), so the orderability gate must reject them.
func TestMultiKeySortMixedDurationsRaisesXTDE1030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort select="@g"/>
        <xsl:sort select="if (@t='y') then xs:yearMonthDuration(@v) else xs:dayTimeDuration(@v)"/>
        <xsl:value-of select="@v"/>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item g="x" t="y" v="P1Y"/><item g="x" t="d" v="P400D"/></root>`))
	require.NoError(t, err)

	_, err = ss.Transform(doc).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "XTDE1030")
}

// XSLT3-102 r4: a no-`select` BODY xsl:sort whose sequence-constructor key
// yields mutually incomparable atomic types across the sequence (xs:date for
// one item, xs:integer for another) must raise XTDE1030. Pre-fix the body path
// failed to record the original atomized value, so the per-level type gate
// (which skips values lacking an atom) silently bypassed the check.
func TestBodySortMixedIncomparableTypesRaisesXTDE1030(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="root/item">
        <xsl:sort>
          <xsl:choose>
            <xsl:when test="@t='d'"><xsl:sequence select="xs:date(@v)"/></xsl:when>
            <xsl:otherwise><xsl:sequence select="xs:integer(@v)"/></xsl:otherwise>
          </xsl:choose>
        </xsl:sort>
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

// XSLT3-102 r6: a default single-key sort whose keys are schema/type-annotated
// xs:dayTimeDuration NODES must order by the TYPED VALUE, not by the string. A
// node key is atomized for the XTDE1030 gate, but pre-r6 the atomized typed
// value was NOT fed through the same auto numeric/duration promotion that an
// atomic singleton receives: fillSingletonSortValue only recorded the node's
// type, leaving sv as text, and extractNumericSortValue (the NumberAuto branch)
// only converted already-atomic items. So a typed duration node sorted by its
// lexical string. Document order PT3H,PT1H,PT20H sorts lexically as
// PT1H<PT20H<PT3H but by value as PT1H<PT3H<PT20H.
func TestSingleKeySortDurationNodeOrdersByValue(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <xsl:variable name="ds" as="element()*">
      <xsl:for-each select="root/item">
        <xsl:element name="d" type="xs:dayTimeDuration"><xsl:value-of select="@v"/></xsl:element>
      </xsl:for-each>
    </xsl:variable>
    <out>
      <xsl:for-each select="$ds">
        <xsl:sort select="."/>
        <xsl:value-of select="."/><xsl:text>|</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item v="PT3H"/><item v="PT1H"/><item v="PT20H"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	// Value order, not lexical PT1H|PT20H|PT3H.
	require.Contains(t, result, ">PT1H|PT3H|PT20H|<")
}

// XSLT3-102 r6: same typed-node value-ordering requirement applied to a
// SECONDARY (multi-key) sort level. With a uniform primary key every record
// ties on level 1, so the typed xs:dayTimeDuration NODE secondary level must
// break the tie by typed value. Pre-r6 the secondary level compared the nodes
// by their lexical strings (PT1H<PT20H<PT3H) instead of by value
// (PT1H<PT3H<PT20H).
func TestMultiKeySortDurationNodeSecondaryOrdersByValue(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <xsl:variable name="ds" as="element()*">
      <xsl:for-each select="root/item">
        <xsl:element name="d" type="xs:dayTimeDuration"><xsl:value-of select="@v"/></xsl:element>
      </xsl:for-each>
    </xsl:variable>
    <out>
      <xsl:for-each select="$ds">
        <xsl:sort select="'k'"/>
        <xsl:sort select="."/>
        <xsl:value-of select="."/><xsl:text>|</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item v="PT3H"/><item v="PT1H"/><item v="PT20H"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">PT1H|PT3H|PT20H|<")
}

// XSLT3-102 r6: a default sort level flipped to auto-number mode by an earlier
// ATOMIC date item must still sort later schema-typed xs:date NODES by their
// typed value, not as NaN. Pre-r6 the dataTypeNumberAuto branch ran the node
// through extractNumericSortValue, which only converts already-atomic items, so
// the node fell back to parsing its lexical string ("2019-01-01") to a number →
// NaN, and every date node sorted ahead of the atomic key. The mixed sequence
// puts the atomic xs:date('2000-06-15') first (flipping the level to
// number-auto) followed by the typed date nodes; correct value order is
// 2000-06-15 < 2010 < 2019 < 2021.
func TestNumberAutoDateNodeOrdersByValue(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/">
    <xsl:variable name="dn" as="element()*">
      <xsl:for-each select="root/item">
        <xsl:element name="d" type="xs:date"><xsl:value-of select="@v"/></xsl:element>
      </xsl:for-each>
    </xsl:variable>
    <xsl:variable name="mixed" as="item()*" select="(xs:date('2000-06-15'), $dn)"/>
    <out>
      <xsl:for-each select="$mixed">
        <xsl:sort select="."/>
        <xsl:value-of select="."/><xsl:text>|</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item v="2019-01-01"/><item v="2021-01-01"/><item v="2010-01-01"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">2000-06-15|2010-01-01|2019-01-01|2021-01-01|<")
}

// XSLT3-102 r7: a default single-key sort whose keys are SCHEMA-DERIVED date
// NODES (a type derived from xs:date) must order by the TYPED date value, not as
// text. The XTDE1030 validation gate already accepts such derived types (it
// promotes via xpath3.ValueCompare/PromoteSchemaType), but pre-r7 the
// comparison-value conversion switched on the EXACT TypeName, so a derived date
// fell through to a text sort. BCE years discriminate text from value order:
// lexically "-0044-01-01" < "-0100-01-01" (3rd digit 0<1), but chronologically
// 100 BCE precedes 44 BCE, so the value order reverses the text order.
func TestSingleKeySortDerivedDateNodeOrdersByValue(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
      <xs:simpleType name="myDate">
        <xs:restriction base="xs:date"/>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>
  <xsl:template match="/">
    <xsl:variable name="ds" as="element()*">
      <xsl:for-each select="root/item">
        <xsl:element name="d" type="myDate">
          <xsl:attribute name="o" select="@v"/>
          <xsl:value-of select="@v"/>
        </xsl:element>
      </xsl:for-each>
    </xsl:variable>
    <out>
      <xsl:for-each select="$ds">
        <xsl:sort select="."/>
        <xsl:value-of select="@o"/><xsl:text>|</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item v="-0044-01-01"/><item v="-0100-01-01"/><item v="0050-01-01"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	// Chronological value order, NOT the lexical text order
	// (-0044-01-01|-0100-01-01|0050-01-01).
	require.Contains(t, result, ">-0100-01-01|-0044-01-01|0050-01-01|<")
}

// XSLT3-102 r7: same schema-derived xs:date value-ordering requirement applied to
// a SECONDARY (multi-key) sort level. With a uniform primary key every record ties
// on level 1, so the derived-date secondary level must break the tie by typed
// value. Pre-r7 it compared by text, giving the reversed BCE order.
func TestMultiKeySortDerivedDateNodeSecondaryOrdersByValue(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
      <xs:simpleType name="myDate">
        <xs:restriction base="xs:date"/>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>
  <xsl:template match="/">
    <xsl:variable name="ds" as="element()*">
      <xsl:for-each select="root/item">
        <xsl:element name="d" type="myDate">
          <xsl:attribute name="o" select="@v"/>
          <xsl:value-of select="@v"/>
        </xsl:element>
      </xsl:for-each>
    </xsl:variable>
    <out>
      <xsl:for-each select="$ds">
        <xsl:sort select="'k'"/>
        <xsl:sort select="."/>
        <xsl:value-of select="@o"/><xsl:text>|</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`)

	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<root><item v="-0044-01-01"/><item v="-0100-01-01"/><item v="0050-01-01"/></root>`))
	require.NoError(t, err)

	result, err := ss.Transform(doc).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, ">-0100-01-01|-0044-01-01|0050-01-01|<")
}
