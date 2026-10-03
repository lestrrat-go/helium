package xslt3_test

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// runNumberStylesheet compiles a stylesheet with top-level declarations decls
// and a root template whose body is body, applies it to source, and returns
// the text output. Documents in docs are served by URI suffix.
func runNumberStylesheet(t *testing.T, source, decls, body string, docs numberDocResolver) string {
	t.Helper()
	text := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="urn:f" exclude-result-prefixes="xs f">
<xsl:output method="text"/>` + decls + `
<xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
	ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(text))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), ssDoc)
	require.NoError(t, err)
	src, err := helium.NewParser().Parse(t.Context(), []byte(source))
	require.NoError(t, err)
	got, err := ss.Transform(src).URIResolver(docs).Serialize(t.Context())
	require.NoError(t, err)
	return got
}

// numberDocResolver serves in-memory documents keyed by URI suffix.
type numberDocResolver map[string]string

func (r numberDocResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	for suffix, content := range r {
		if strings.HasSuffix(uri, suffix) {
			return io.NopCloser(strings.NewReader(content)), nil
		}
	}
	return nil, fmt.Errorf("no document for %q", uri)
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
		decls  string
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
		{
			// The pattern "." also matches r and the document node, which
			// level="any" counts as ancestors.
			name:   "count dot",
			source: `<r><a/><b/><a/></r>`,
			body: `<xsl:for-each select="r/*">[<xsl:number count="."/>,` +
				`<xsl:number level="any" count="."/>]</xsl:for-each>`,
			want: `[1,3][2,4][3,5]`,
		},
		{
			name:   "attributes as the selected node",
			source: `<r><x a="1" b="2"/><x a="3"/></r>`,
			body: `<xsl:for-each select="r/x/@*">[<xsl:number/>,` +
				`<xsl:number level="any" count="x"/>]</xsl:for-each>`,
			want: `[1,1][1,1][1,2]`,
		},
		{
			// f:n numbers the preceding x with the same two instructions while
			// they are counting, so the walks interleave.
			name:   "count pattern runs the same instruction again",
			source: `<r><x k="y"/><x/><x k="y"/><x k="y"/><x/><x k="y"/></r>`,
			decls: `<xsl:function name="f:n"><xsl:param name="n"/><xsl:variable name="s">` +
				`<xsl:for-each select="$n/preceding-sibling::x[1]"><xsl:call-template name="num"/>` +
				`<xsl:call-template name="numa"/></xsl:for-each></xsl:variable>` +
				`<xsl:sequence select="string-length($s) ge 0 and $n/@k = 'y'"/></xsl:function>` +
				`<xsl:template name="num"><xsl:number count="x[f:n(.)]"/></xsl:template>` +
				`<xsl:template name="numa"><xsl:number level="any" count="x[f:n(.)]"/></xsl:template>`,
			body: `<xsl:for-each select="r/x">[<xsl:call-template name="num"/>,` +
				`<xsl:call-template name="numa"/>]</xsl:for-each>`,
			want: `[1,1][,1][2,2][3,3][,3][4,4]`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, runNumberStylesheet(t, testCase.source, testCase.decls, testCase.body, nil))
		})
	}
}

// TestNumberAttributeAndNamespace checks xsl:number when the selected node is
// an attribute or namespace node. XDM gives those nodes no siblings and keeps
// them off the preceding axis (XSLT 3.0 §12.3 counts preceding-sibling::node()
// for level="single"/"multiple" and preceding::node()|ancestor-or-self::node()
// for level="any"), so the other attributes of the same element are never
// counted. The repeated cases number several attributes of one element with
// one instruction, so a count carried over from the previous attribute would
// give a wrong number.
func TestNumberAttributeAndNamespace(t *testing.T) {
	const twoAttrs = `<r b="1" a="2"/>`
	testCases := []struct {
		name   string
		source string
		body   string
		want   string
	}{
		{
			name:   "single count any attribute",
			source: twoAttrs,
			body:   `<xsl:for-each select="r/@a"><xsl:number count="@*"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "single count named attribute",
			source: twoAttrs,
			body:   `<xsl:for-each select="r/@a"><xsl:number count="@a"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "single default count",
			source: twoAttrs,
			body:   `<xsl:for-each select="r/@a"><xsl:number/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "single select attribute",
			source: twoAttrs,
			body:   `<xsl:number select="r/@a" count="@*"/>`,
			want:   `1`,
		},
		{
			name:   "single count attribute or element",
			source: `<r><x/><y b="1" a="2"/></r>`,
			body:   `<xsl:for-each select="r/y/@a"><xsl:number count="@*|*"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "single repeated over attributes",
			source: `<r a="1" b="2" c="3"/>`,
			body:   `<xsl:for-each select="r/@*">[<xsl:number count="@*"/>]</xsl:for-each>`,
			want:   `[1][1][1]`,
		},
		{
			name:   "any count any attribute",
			source: twoAttrs,
			body:   `<xsl:for-each select="r/@a"><xsl:number level="any" count="@*"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "any count named attribute",
			source: `<r><x a="1"/><y b="1" a="2"/></r>`,
			body:   `<xsl:for-each select="r/y/@a"><xsl:number level="any" count="@a"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "any default count",
			source: `<r><x a="1"/><y b="1" a="2"/></r>`,
			body:   `<xsl:for-each select="r/y/@a"><xsl:number level="any"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "any counts preceding elements but not sibling attributes",
			source: `<r><x/><y b="1" a="2"/></r>`,
			body: `<xsl:for-each select="r/y/@a">` +
				`<xsl:number level="any" count="node()|@*"/></xsl:for-each>`,
			want: `4`,
		},
		{
			name:   "any repeated over attributes",
			source: `<r a="1" b="2" c="3"/>`,
			body:   `<xsl:for-each select="r/@*">[<xsl:number level="any" count="@*"/>]</xsl:for-each>`,
			want:   `[1][1][1]`,
		},
		{
			name:   "any repeated over elements then attributes",
			source: `<r><x/><y a="1" b="2"/></r>`,
			body: `<xsl:for-each select="//*|//@*">` +
				`[<xsl:number level="any" count="node()|@*"/>]</xsl:for-each>`,
			want: `[1][2][3][4][4]`,
		},
		{
			name:   "multiple count any attribute",
			source: twoAttrs,
			body:   `<xsl:for-each select="r/@a"><xsl:number level="multiple" count="@*"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "multiple count named attribute",
			source: twoAttrs,
			body:   `<xsl:for-each select="r/@a"><xsl:number level="multiple" count="@a"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "multiple default count",
			source: twoAttrs,
			body:   `<xsl:for-each select="r/@a"><xsl:number level="multiple"/></xsl:for-each>`,
			want:   `1`,
		},
		{
			name:   "multiple count elements and attributes",
			source: `<r><x/><y b="1" a="2"/></r>`,
			body: `<xsl:for-each select="r/y/@a">` +
				`<xsl:number level="multiple" count="*|@*"/></xsl:for-each>`,
			want: `1.2.1`,
		},
		{
			name:   "multiple repeated over attributes",
			source: `<r a="1" b="2" c="3"/>`,
			body:   `<xsl:for-each select="r/@*">[<xsl:number level="multiple" count="@*"/>]</xsl:for-each>`,
			want:   `[1][1][1]`,
		},
		{
			name:   "namespace node default count",
			source: `<r xmlns:p="urn:p" xmlns:q="urn:q"/>`,
			body: `<xsl:for-each select="r/namespace::q">[<xsl:number/>,` +
				`<xsl:number level="any"/>,<xsl:number level="multiple"/>]</xsl:for-each>`,
			want: `[1,1,1]`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, runNumberStylesheet(t, testCase.source, "", testCase.body, nil))
		})
	}
}

// TestNumberSingleFrom checks level="single" with a from pattern (XSLT 3.0
// §12.3): the counted node must lie inside the subtree of the innermost
// ancestor-or-self of the selected node that matches from, where the root of
// the tree always matches. Otherwise the place marker is empty.
func TestNumberSingleFrom(t *testing.T) {
	testCases := []struct {
		name   string
		source string
		body   string
		want   string
	}{
		{
			name:   "from match between the selected node and the counted node",
			source: `<r><x/><x><y><z/></y></x></r>`,
			body:   `<xsl:for-each select="//z">[<xsl:number count="x" from="y"/>]</xsl:for-each>`,
			want:   `[]`,
		},
		{
			// The first x has no y ancestor, so the root is its from node and
			// it is numbered among its siblings.
			name:   "root counts as the from node when no ancestor matches",
			source: `<r><x/><y><x/><x/></y></r>`,
			body:   `<xsl:for-each select="//x">[<xsl:number count="x" from="y"/>]</xsl:for-each>`,
			want:   `[1][1][2]`,
		},
		{
			name:   "counted node that matches from",
			source: `<r><x/><x/></r>`,
			body:   `<xsl:for-each select="r/x">[<xsl:number count="x" from="x"/>]</xsl:for-each>`,
			want:   `[1][2]`,
		},
		{
			name:   "counted node inside the from subtree",
			source: `<r><y><x><z/></x><x><z/></x><x><z/></x></y></r>`,
			body:   `<xsl:for-each select="//z">[<xsl:number count="x" from="y"/>]</xsl:for-each>`,
			want:   `[1][2][3]`,
		},
		{
			// One instruction numbers z elements in document order, so each
			// count can reuse the previous one; those behind a y between z and
			// its x must still come out empty, and the others still count
			// every earlier x sibling.
			name: "counts in a row alternate across from boundaries",
			source: `<r><x><z/></x><x><y><z/></y></x><x><z/></x><x><y><z/></y></x>` +
				`<x><z/></x></r>`,
			body: `<xsl:for-each select="//z">[<xsl:number count="x" from="y"/>]</xsl:for-each>`,
			want: `[1][][3][][5]`,
		},
		{
			name:   "default count with a from match above the selected node",
			source: `<r><z/><y><a><z/></a></y><z/></r>`,
			body:   `<xsl:for-each select="//z">[<xsl:number from="a"/>]</xsl:for-each>`,
			want:   `[1][1][2]`,
		},
		{
			// from is tested on the ancestors of the selected node, not of
			// the context node.
			name:   "select names a node outside the context node's from subtree",
			source: `<r><x/><x><z/></x><y><w/></y></r>`,
			body:   `<xsl:for-each select="//w">[<xsl:number select="/r/x[2]/z" count="x" from="y"/>]</xsl:for-each>`,
			want:   `[2]`,
		},
		{
			name:   "select names a node behind a from match",
			source: `<r><x/><x><y><z/></y></x></r>`,
			body:   `<xsl:for-each select="r/x[1]">[<xsl:number select="../x[2]/y/z" count="x" from="y"/>]</xsl:for-each>`,
			want:   `[]`,
		},
		{
			name:   "parentless element tree",
			source: `<r/>`,
			body: `<xsl:variable name="e" as="element()"><x><x/><x><y><z/></y></x></x></xsl:variable>` +
				`[<xsl:number select="$e/x[2]/y/z" count="x" from="y"/>]` +
				`[<xsl:number select="$e/x[2]" count="x" from="y"/>]`,
			want: `[][2]`,
		},
		{
			name:   "attribute as the selected node",
			source: `<r><x/><x a="1"/></r>`,
			body: `<xsl:for-each select="r/x[2]/@a">[<xsl:number count="x" from="@a"/>]` +
				`[<xsl:number count="x" from="y"/>]</xsl:for-each>`,
			want: `[][2]`,
		},
		{
			name:   "multiple stops at the innermost from match",
			source: `<r><x/><x><y><z/></y></x></r>`,
			body:   `<xsl:for-each select="//z">[<xsl:number level="multiple" count="x|z" from="y"/>]</xsl:for-each>`,
			want:   `[1]`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, runNumberStylesheet(t, testCase.source, "", testCase.body, nil))
		})
	}
}

// numberSchema types r as a sequence of x elements with element-only
// content, so validation drops the whitespace between them and annotates
// each x as xT.
const numberSchema = `<xsl:import-schema><xs:schema>
<xs:element name="r" type="rT"/>
<xs:complexType name="rT"><xs:sequence><xs:element name="x" maxOccurs="unbounded" type="xT"/></xs:sequence></xs:complexType>
<xs:complexType name="xT"/>
</xs:schema></xsl:import-schema>`

// TestNumberCountingAfterValidation numbers nodes of a document before and
// after xsl:source-document validates that same cached document in place,
// which removes whitespace text nodes and adds type annotations. Each number
// must reflect the tree as it is when xsl:number runs.
func TestNumberCountingAfterValidation(t *testing.T) {
	docs := numberDocResolver{"a.xml": "<r>\n <x/>\n <x/>\n</r>"}
	testCases := []struct {
		name  string
		decls string
		body  string
		want  string
	}{
		{
			name:  "single after whitespace is stripped",
			decls: numberSchema + `<xsl:template name="n"><xsl:number count="node()"/></xsl:template>`,
			body: `<xsl:for-each select="doc('mem:a.xml')/r/x[1]">[<xsl:call-template name="n"/>]</xsl:for-each>` +
				`<xsl:source-document href="mem:a.xml" validation="strict" streamable="no">` +
				`<xsl:for-each select="r/x[2]">[<xsl:call-template name="n"/>]</xsl:for-each>` +
				`</xsl:source-document>`,
			want: `[2][2]`,
		},
		{
			name:  "any after annotations are added",
			decls: numberSchema + `<xsl:template name="n"><xsl:number level="any" count="element(*, xT)"/></xsl:template>`,
			body: `<xsl:source-document href="mem:a.xml" streamable="no">` +
				`<xsl:for-each select="r/x[1]">[<xsl:call-template name="n"/>]</xsl:for-each>` +
				`</xsl:source-document>` +
				`<xsl:source-document href="mem:a.xml" validation="strict" streamable="no">` +
				`<xsl:for-each select="r/x[2]">[<xsl:call-template name="n"/>]</xsl:for-each>` +
				`</xsl:source-document>`,
			want: `[][2]`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.want, runNumberStylesheet(t, `<z/>`, testCase.decls, testCase.body, docs))
		})
	}
}
