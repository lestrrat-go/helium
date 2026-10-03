package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// runNumberStylesheet compiles a stylesheet whose root template body is body,
// applies it to source, and returns the text output.
func runNumberStylesheet(t *testing.T, source, body string) string {
	t.Helper()
	text := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:output method="text"/>
<xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
	ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(text))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), ssDoc)
	require.NoError(t, err)
	src, err := helium.NewParser().Parse(t.Context(), []byte(source))
	require.NoError(t, err)
	got, err := ss.Transform(src).Serialize(t.Context())
	require.NoError(t, err)
	return got
}

// TestNumberCounting checks xsl:number level="single" and level="any" when one
// instruction numbers many nodes in a row, as a list numbering does. Each case
// mixes nodes so that a count carried over from the previous call would give a
// wrong number: different names and kinds under the default count pattern,
// count patterns that read a variable, from boundaries, nodes visited out of
// document order, and nodes from more than one tree.
func TestNumberCounting(t *testing.T) {
	testCases := []struct {
		name   string
		source string
		body   string
		want   string
	}{
		{
			name:   "single default count with mixed names",
			source: `<r><a/><b/><a/><b/><a/></r>`,
			body:   `<xsl:for-each select="r/*">[<xsl:number/>]</xsl:for-each>`,
			want:   `[1][1][2][2][3]`,
		},
		{
			name:   "single default count with mixed node kinds",
			source: `<r><a/><!--c--><a/><?p x?><!--c--><a/></r>`,
			body:   `<xsl:for-each select="r/node()">[<xsl:number/>]</xsl:for-each>`,
			want:   `[1][1][2][1][2][3]`,
		},
		{
			name:   "single default count with one local name in two namespaces",
			source: `<r xmlns:p="urn:p"><a/><p:a/><a/><p:a/></r>`,
			body:   `<xsl:for-each select="r/*">[<xsl:number/>]</xsl:for-each>`,
			want:   `[1][1][2][2]`,
		},
		{
			name:   "single count pattern on an ancestor",
			source: `<r><s><x/><x/></s><t/><s><x/></s></r>`,
			body:   `<xsl:for-each select="//x">[<xsl:number count="s"/>]</xsl:for-each>`,
			want:   `[1][1][2]`,
		},
		{
			name:   "single in reverse document order",
			source: `<r><a/><a/><a/><a/></r>`,
			body: `<xsl:for-each select="r/a"><xsl:sort select="position()" order="descending"/>` +
				`[<xsl:number/>]</xsl:for-each>`,
			want: `[4][3][2][1]`,
		},
		{
			name:   "single count pattern reading a local variable",
			source: `<r><x k="1"/><x k="2"/><x k="1"/><x k="2"/></r>`,
			body: `<xsl:for-each select="r/x"><xsl:variable name="k" select="@k"/>` +
				`[<xsl:number count="x[@k = $k]"/>]</xsl:for-each>`,
			want: `[1][1][2][2]`,
		},
		{
			name:   "single numbering two trees in turn",
			source: `<r><a/><a/><a/></r>`,
			body: `<xsl:variable name="t"><r><a/><a/><a/></r></xsl:variable>` +
				`<xsl:for-each select="r/a"><xsl:variable name="i" select="position()"/>` +
				`[<xsl:number/>,<xsl:number select="$t/r/a[4 - $i]"/>]</xsl:for-each>`,
			want: `[1,3][2,2][3,1]`,
		},
		{
			name:   "any with count pattern",
			source: `<r><x/><y><x/><z><x/></z></y><x/></r>`,
			body:   `<xsl:for-each select="//x">[<xsl:number level="any" count="x"/>]</xsl:for-each>`,
			want:   `[1][2][3][4]`,
		},
		{
			name:   "any default count with mixed names",
			source: `<r><a/><b><a/></b><b/><a/></r>`,
			body:   `<xsl:for-each select="//*">[<xsl:number level="any"/>]</xsl:for-each>`,
			want:   `[1][1][1][2][2][3]`,
		},
		{
			name:   "any restarts at each from node",
			source: `<r><s><x/><x/></s><s><x/><x/><x/></s><x/></r>`,
			body:   `<xsl:for-each select="//x">[<xsl:number level="any" count="x" from="s"/>]</xsl:for-each>`,
			want:   `[1][2][1][2][3][4]`,
		},
		{
			name:   "any counts a from node that also matches count",
			source: `<r><x/><x f="1"/><x/><x f="1"/><x/></r>`,
			body:   `<xsl:for-each select="r/x">[<xsl:number level="any" count="x" from="x[@f]"/>]</xsl:for-each>`,
			want:   `[1][1][2][1][2]`,
		},
		{
			name:   "any count pattern reading a local variable",
			source: `<r><x k="1"/><x k="2"/><x k="1"/><x k="2"/></r>`,
			body: `<xsl:for-each select="r/x"><xsl:variable name="k" select="@k"/>` +
				`[<xsl:number level="any" count="x[@k = $k]"/>]</xsl:for-each>`,
			want: `[1][1][2][2]`,
		},
		{
			name:   "any from pattern reading a local variable",
			source: `<r><x k="1"/><x k="2"/><x k="3"/></r>`,
			body: `<xsl:for-each select="r/x"><xsl:variable name="k" select="@k"/>` +
				`[<xsl:number level="any" count="x" from="x[@k = $k - 1]"/>]</xsl:for-each>`,
			want: `[1][2][2]`,
		},
		{
			name:   "any numbering two trees in turn",
			source: `<r><a/><a/><a/></r>`,
			body: `<xsl:variable name="t"><r><a/><a/><a/></r></xsl:variable>` +
				`<xsl:for-each select="r/a"><xsl:variable name="i" select="position()"/>` +
				`[<xsl:number level="any"/>,<xsl:number level="any" select="$t/r/a[$i]"/>]</xsl:for-each>`,
			want: `[1,1][2,2][3,3]`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, runNumberStylesheet(t, testCase.source, testCase.body))
		})
	}
}
