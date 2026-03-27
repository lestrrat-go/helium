package xslt3_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func buildSortSource(b *testing.B, n int) *helium.Document {
	b.Helper()

	var sb strings.Builder
	sb.WriteString("<root>")
	for i := range n {
		sb.WriteString("<item>")
		sb.WriteString(strconv.Itoa(n - i))
		sb.WriteString("</item>")
	}
	sb.WriteString("</root>")

	doc, err := helium.NewParser().Parse(b.Context(), []byte(sb.String()))
	require.NoError(b, err)
	return doc
}

func BenchmarkSortNodes(b *testing.B) {
	textAscSS := compileStylesheetBench(b, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root">
    <sorted>
      <xsl:for-each select="item">
        <xsl:sort select="."/>
        <xsl:copy-of select="."/>
      </xsl:for-each>
    </sorted>
  </xsl:template>
</xsl:stylesheet>`)

	numericAscSS := compileStylesheetBench(b, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root">
    <sorted>
      <xsl:for-each select="item">
        <xsl:sort select="." data-type="number"/>
        <xsl:copy-of select="."/>
      </xsl:for-each>
    </sorted>
  </xsl:template>
</xsl:stylesheet>`)

	autoNumericSS := compileStylesheetBench(b, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root">
    <sorted>
      <xsl:for-each select="item">
        <xsl:sort select="number(.)"/>
        <xsl:copy-of select="."/>
      </xsl:for-each>
    </sorted>
  </xsl:template>
</xsl:stylesheet>`)

	for _, size := range []int{100, 1000, 10000} {
		source := buildSortSource(b, size)

		b.Run(fmt.Sprintf("TextAsc/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_, err := xslt3.Transform(b.Context(), source, textAscSS)
				require.NoError(b, err)
			}
		})

		b.Run(fmt.Sprintf("NumericAsc/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_, err := xslt3.Transform(b.Context(), source, numericAscSS)
				require.NoError(b, err)
			}
		})

		b.Run(fmt.Sprintf("AutoNumericAsc/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_, err := xslt3.Transform(b.Context(), source, autoNumericSS)
				require.NoError(b, err)
			}
		})
	}
}

func BenchmarkSortNodesMultiKey(b *testing.B) {
	threeKeySS := compileStylesheetBench(b, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root">
    <sorted>
      <xsl:for-each select="item">
        <xsl:sort select="."/>
        <xsl:sort select="." data-type="number"/>
        <xsl:sort select="." order="descending"/>
        <xsl:copy-of select="."/>
      </xsl:for-each>
    </sorted>
  </xsl:template>
</xsl:stylesheet>`)

	for _, size := range []int{100, 1000} {
		source := buildSortSource(b, size)

		b.Run(fmt.Sprintf("ThreeKey/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				_, err := xslt3.Transform(b.Context(), source, threeKeySS)
				require.NoError(b, err)
			}
		})
	}
}

func compileStylesheetBench(b *testing.B, src string) *xslt3.Stylesheet {
	b.Helper()

	doc, err := helium.NewParser().Parse(b.Context(), []byte(src))
	require.NoError(b, err)

	ss, err := xslt3.CompileStylesheet(b.Context(), doc)
	require.NoError(b, err)
	return ss
}
