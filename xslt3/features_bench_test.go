package xslt3_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// featureBenchRecords is the number of <rec> elements in the source document
// every instruction-level benchmark in this file transforms.
const featureBenchRecords = 5000

// buildFeatureSource builds a flat <recs> document of nRecs records. Record i
// carries:
//
//   - id="r{i}" and n="{i mod 37}" (a small repeating number);
//   - cat="c{i mod nCats}", so the categories interleave and group-by
//     collects nCats groups out of document order;
//   - run="u{i div 100}", so equal values sit next to each other in runs of
//     100 for group-adjacent;
//   - head="yes" on every 100th record, the group starts for
//     group-starting-with;
//   - a <v>{i}</v> child to sum.
func buildFeatureSource(nRecs, nCats int) []byte {
	var b strings.Builder
	b.WriteString(`<recs>`)
	for i := range nRecs {
		fmt.Fprintf(&b, `<rec id="r%d" cat="c%d" run="u%d" n="%d"`, i, i%nCats, i/100, i%37)
		if i%100 == 0 {
			b.WriteString(` head="yes"`)
		}
		fmt.Fprintf(&b, `><v>%d</v></rec>`, i)
	}
	b.WriteString(`</recs>`)
	return []byte(b.String())
}

// buildCompileBenchStylesheet generates a stylesheet with nTemplates match
// templates in the unnamed mode, plus the declarations a real mid-size
// stylesheet carries alongside them:
//
//   - two xsl:keys, a typed global xsl:param and a map-valued global
//     xsl:variable;
//   - nTemplates/10 xsl:functions (at least 2) and nTemplates/20 named
//     templates (at least 2), which the match templates call;
//   - a named "summary" mode declared with xsl:mode, holding one template for
//     every fifth match template.
//
// Every match template binds a typed local variable, branches with
// xsl:choose, calls a function or a named template with a parameter, builds
// literal result elements with attribute value templates, and iterates a key
// lookup with xsl:for-each.
func buildCompileBenchStylesheet(nTemplates int) string {
	nFuncs := max(nTemplates/10, 2)
	nNamed := max(nTemplates/20, 2)

	var b strings.Builder
	b.WriteString(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"` +
		` xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="urn:bench" exclude-result-prefixes="xs f">`)
	b.WriteString(`<xsl:output method="xml" indent="no"/>`)
	b.WriteString(`<xsl:mode name="summary" on-no-match="shallow-skip"/>`)
	b.WriteString(`<xsl:key name="by-cat" match="rec" use="@cat"/>`)
	b.WriteString(`<xsl:key name="by-id" match="rec" use="@id"/>`)
	b.WriteString(`<xsl:param name="threshold" as="xs:integer" select="10"/>`)
	b.WriteString(`<xsl:variable name="labels" as="map(xs:string, xs:string)" select="map{'big':'Big','row':'Row'}"/>`)
	for i := range nFuncs {
		fmt.Fprintf(&b, `<xsl:function name="f:scale%d" as="xs:double">`+
			`<xsl:param name="v" as="xs:double"/>`+
			`<xsl:sequence select="$v * %d + $threshold"/>`+
			`</xsl:function>`, i, i+1)
	}
	for i := range nNamed {
		fmt.Fprintf(&b, `<xsl:template name="row%d">`+
			`<xsl:param name="node" as="element()"/>`+
			`<row id="{$node/@id}" label="{$labels?row}"><xsl:value-of select="upper-case(string($node/@cat))"/></row>`+
			`</xsl:template>`, i)
	}
	b.WriteString(`<xsl:template match="/">` +
		`<out><xsl:apply-templates select="//rec"/>` +
		`<summary><xsl:apply-templates select="//rec" mode="summary"/></summary></out>` +
		`</xsl:template>`)
	for i := range nTemplates {
		fmt.Fprintf(&b, `<xsl:template match="rec[@cat='c%d']">`+
			`<xsl:variable name="v" as="xs:double" select="xs:double(@n)"/>`+
			`<xsl:choose>`+
			`<xsl:when test="$v gt $threshold"><big id="{@id}" s="{f:scale%d($v)}" label="{$labels?big}"/></xsl:when>`+
			`<xsl:otherwise><xsl:call-template name="row%d"><xsl:with-param name="node" select="."/></xsl:call-template></xsl:otherwise>`+
			`</xsl:choose>`+
			`<xsl:for-each select="key('by-cat', @cat)[position() le 2]"><peer ref="{@id}"/></xsl:for-each>`+
			`</xsl:template>`, i, i%nFuncs, i%nNamed)
		if i%5 == 0 {
			fmt.Fprintf(&b, `<xsl:template match="rec[@cat='c%d']" mode="summary">`+
				`<s c="{@cat}"><xsl:value-of select="count(key('by-id', @id))"/></s>`+
				`</xsl:template>`, i)
		}
	}
	b.WriteString(`</xsl:stylesheet>`)
	return b.String()
}

// BenchmarkCompileStylesheet times Compiler.Compile alone on an already-parsed
// stylesheet document from buildCompileBenchStylesheet, at three template
// counts so the growth of compile cost with stylesheet size is visible.
//
// Before timing, the stylesheet is compiled once and run over a source with
// one record per match template, and the output is checked to hold one <big>
// or <row> element per record. The stylesheet document is also checked to
// serialize the same before and after that compile, so every timed iteration
// compiles the same parsed tree.
func BenchmarkCompileStylesheet(b *testing.B) {
	for _, n := range []int{20, 100, 500} {
		b.Run(fmt.Sprintf("templates=%d", n), func(b *testing.B) {
			runCompileBench(b, n)
		})
	}
}

func runCompileBench(b *testing.B, nTemplates int) {
	b.Helper()
	p := helium.NewParser()
	sdoc, err := p.Parse(b.Context(), []byte(buildCompileBenchStylesheet(nTemplates)))
	require.NoError(b, err)

	before, err := helium.WriteString(sdoc)
	require.NoError(b, err)
	compiler := xslt3.NewCompiler()
	ss, err := compiler.Compile(b.Context(), sdoc)
	require.NoError(b, err)
	after, err := helium.WriteString(sdoc)
	require.NoError(b, err)
	require.Equal(b, before, after, "Compile must leave the stylesheet document unchanged")

	src, err := p.Parse(b.Context(), buildFeatureSource(nTemplates, nTemplates))
	require.NoError(b, err)
	out, err := xslt3.TransformString(b.Context(), src, ss)
	require.NoError(b, err)
	require.Equal(b, nTemplates, strings.Count(out, "<big ")+strings.Count(out, "<row "), "one result element per record")
	require.Equal(b, (nTemplates+4)/5, strings.Count(out, "<s "), "one summary element per summary-mode template")

	b.ReportAllocs()
	for b.Loop() {
		if _, err := compiler.Compile(b.Context(), sdoc); err != nil {
			b.Fatal(err)
		}
	}
}

// featureBenchCase is one stylesheet in an instruction-level benchmark: its
// source text and a check run once on its serialized output before timing.
type featureBenchCase struct {
	name  string
	xsl   string
	check func(tb testing.TB, out string)
}

// runFeatureBenchCases compiles each case once, transforms the shared source
// (featureBenchRecords records in groupByCats categories) once to check the
// output, then times xslt3.Transform (result tree only, no serialization) per
// case.
func runFeatureBenchCases(b *testing.B, cases []featureBenchCase) {
	b.Helper()
	src, err := helium.NewParser().Parse(b.Context(), buildFeatureSource(featureBenchRecords, groupByCats))
	require.NoError(b, err)

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			ss := compileBenchStylesheet(b, tc.xsl)
			out, err := xslt3.TransformString(b.Context(), src, ss)
			require.NoError(b, err)
			tc.check(b, out)

			b.ReportAllocs()
			for b.Loop() {
				if _, err := xslt3.Transform(b.Context(), src, ss); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// groupByCats is the number of distinct @cat values in the source every
// instruction-level benchmark transforms.
const groupByCats = 50

const groupByStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/recs">
    <out>
      <xsl:for-each-group select="rec" group-by="@cat">
        <g k="{current-grouping-key()}" n="{count(current-group())}" s="{sum(current-group()/v)}"/>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

const groupAdjacentStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/recs">
    <out>
      <xsl:for-each-group select="rec" group-adjacent="@run">
        <g k="{current-grouping-key()}" n="{count(current-group())}" s="{sum(current-group()/v)}"/>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

const groupStartingWithStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/recs">
    <out>
      <xsl:for-each-group select="rec" group-starting-with="rec[@head]">
        <g k="{@id}" n="{count(current-group())}" s="{sum(current-group()/v)}"/>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

// checkGroupBy expects one group per category; category c0 holds records 0,
// 50, ..., 4950, so its count is 100 and its sum is 247500.
func checkGroupBy(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, groupByCats, strings.Count(out, "<g "))
	require.Contains(tb, out, `<g k="c0" n="100" s="247500"/>`)
}

// checkGroupAdjacent expects one group per run of 100 records; the first run
// holds records 0..99, summing to 4950.
func checkGroupAdjacent(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords/100, strings.Count(out, "<g "))
	require.Contains(tb, out, `<g k="u0" n="100" s="4950"/>`)
}

// checkGroupStartingWith expects one group per head record; the first group
// starts at r0 and holds records 0..99.
func checkGroupStartingWith(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords/100, strings.Count(out, "<g "))
	require.Contains(tb, out, `<g k="r0" n="100" s="4950"/>`)
}

// BenchmarkForEachGroup times xslt3.Transform for xsl:for-each-group over a
// flat document of 5000 records, emitting one element per group with its key,
// size and the sum of its members:
//
//   - group-by: 50 interleaved categories, so every group draws its members
//     from across the whole population;
//   - group-adjacent: 50 runs of 100 equal adjacent keys;
//   - group-starting-with: 50 groups, each opened by a record matching the
//     rec[@head] pattern.
func BenchmarkForEachGroup(b *testing.B) {
	runFeatureBenchCases(b, []featureBenchCase{
		{name: "group-by", xsl: groupByStylesheet, check: checkGroupBy},
		{name: "group-adjacent", xsl: groupAdjacentStylesheet, check: checkGroupAdjacent},
		{name: "group-starting-with", xsl: groupStartingWithStylesheet, check: checkGroupStartingWith},
	})
}

const predicateTemplateStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>
  <xsl:template match="rec[@head]"><h/></xsl:template>
  <xsl:template match="rec"><r/></xsl:template>
</xsl:stylesheet>`

// predicateTemplateCount is the number of match="rec[@cat='cN']" templates
// in the predicate-templates benchmark case.
const predicateTemplateCount = 20

// buildPredicateTemplatesStylesheet generates a stylesheet with one
// match="rec[@cat='cN']" template for each of the first n categories, plus a
// match="rec" fallback for the rest.
func buildPredicateTemplatesStylesheet(n int) string {
	var b strings.Builder
	b.WriteString(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">`)
	b.WriteString(`<xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>`)
	for i := range n {
		fmt.Fprintf(&b, `<xsl:template match="rec[@cat='c%d']"><c/></xsl:template>`, i)
	}
	b.WriteString(`<xsl:template match="rec"><r/></xsl:template>`)
	b.WriteString(`</xsl:stylesheet>`)
	return b.String()
}

// checkPredicateTemplate expects the 50 head records to match rec[@head] and
// every other record to fall through to rec.
func checkPredicateTemplate(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords/100, strings.Count(out, "<h/>"))
	require.Equal(tb, featureBenchRecords-featureBenchRecords/100, strings.Count(out, "<r/>"))
}

// checkPredicateTemplates expects the records in the first 20 of the 50
// categories to match a rec[@cat='cN'] template and the rest to fall through
// to rec.
func checkPredicateTemplates(tb testing.TB, out string) {
	tb.Helper()
	matched := featureBenchRecords / groupByCats * predicateTemplateCount
	require.Equal(tb, matched, strings.Count(out, "<c/>"))
	require.Equal(tb, featureBenchRecords-matched, strings.Count(out, "<r/>"))
}

// BenchmarkPredicatePatterns times xslt3.Transform for template rules whose
// match patterns carry a predicate, applied to each of the 5000 records:
//
//   - predicate-template: one match="rec[@head]" template beside a
//     match="rec" fallback;
//   - predicate-templates-20: twenty match="rec[@cat='cN']" templates beside a
//     match="rec" fallback, so every record is tested against each of them.
func BenchmarkPredicatePatterns(b *testing.B) {
	runFeatureBenchCases(b, []featureBenchCase{
		{name: "predicate-template", xsl: predicateTemplateStylesheet, check: checkPredicateTemplate},
		{
			name:  "predicate-templates-20",
			xsl:   buildPredicateTemplatesStylesheet(predicateTemplateCount),
			check: checkPredicateTemplates,
		},
	})
}

const numberSingleStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>
  <xsl:template match="rec"><r><xsl:number/></r></xsl:template>
</xsl:stylesheet>`

const numberAnyStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>
  <xsl:template match="rec"><r><xsl:number level="any" count="rec"/></r></xsl:template>
</xsl:stylesheet>`

const formatNumberStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>
  <xsl:template match="rec"><r><xsl:value-of select="format-number(@n * 1234.5, '#,##0.00')"/></r></xsl:template>
</xsl:stylesheet>`

// checkNumberSingle expects record i to be numbered i+1 among its siblings.
func checkNumberSingle(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords, strings.Count(out, "<r>"))
	require.Contains(tb, out, fmt.Sprintf("<r>%d</r></out>", featureBenchRecords))
}

// checkNumberAny expects record i to be numbered i+1 among the records
// before it in document order.
func checkNumberAny(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords, strings.Count(out, "<r>"))
	require.Contains(tb, out, fmt.Sprintf("<r>%d</r></out>", featureBenchRecords))
}

// checkFormatNumber expects record 1 (n=1) to format as 1,234.50.
func checkFormatNumber(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords, strings.Count(out, "<r>"))
	require.Contains(tb, out, "<r>0.00</r><r>1,234.50</r>")
}

// BenchmarkNumbering times xslt3.Transform for per-record number output over
// the 5000-record document, one result element per record:
//
//   - number-single: a bare xsl:number, which counts each record's preceding
//     siblings of the same name;
//   - number-any: xsl:number level="any" count="rec", which walks back
//     through every preceding node in document order;
//   - format-number: format-number() with a grouping-separator picture.
func BenchmarkNumbering(b *testing.B) {
	runFeatureBenchCases(b, []featureBenchCase{
		{name: "number-single", xsl: numberSingleStylesheet, check: checkNumberSingle},
		{name: "number-any", xsl: numberAnyStylesheet, check: checkNumberAny},
		{name: "format-number", xsl: formatNumberStylesheet, check: checkFormatNumber},
	})
}

const functionSimpleStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="urn:bench" exclude-result-prefixes="xs f">
  <xsl:function name="f:label" as="xs:string">
    <xsl:param name="cat" as="xs:string"/>
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="concat(upper-case($cat), '-', $n * 2)"/>
  </xsl:function>
  <xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>
  <xsl:template match="rec"><r><xsl:value-of select="f:label(@cat, xs:integer(@n))"/></r></xsl:template>
</xsl:stylesheet>`

const functionRecursiveStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="urn:bench" exclude-result-prefixes="xs f">
  <xsl:function name="f:fact" as="xs:integer">
    <xsl:param name="k" as="xs:integer"/>
    <xsl:sequence select="if ($k le 1) then 1 else $k * f:fact($k - 1)"/>
  </xsl:function>
  <xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>
  <xsl:template match="rec"><r><xsl:value-of select="f:fact(xs:integer(@n) mod 10)"/></r></xsl:template>
</xsl:stylesheet>`

// checkFunctionSimple expects record 1 (cat=c1, n=1) to label as C1-2.
func checkFunctionSimple(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords, strings.Count(out, "<r>"))
	require.Contains(tb, out, "<r>C0-0</r><r>C1-2</r>")
}

// checkFunctionRecursive expects record 9 (n=9) to compute 9! = 362880.
func checkFunctionRecursive(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords, strings.Count(out, "<r>"))
	require.Contains(tb, out, "<r>40320</r><r>362880</r>")
}

// BenchmarkFunctionCall times xslt3.Transform for user xsl:function calls,
// one call site per record over the 5000-record document:
//
//   - simple: a two-parameter function with typed parameters that builds a
//     string, one call per record;
//   - recursive: a recursive factorial of (n mod 10), up to nine nested calls
//     per record.
func BenchmarkFunctionCall(b *testing.B) {
	runFeatureBenchCases(b, []featureBenchCase{
		{name: "simple", xsl: functionSimpleStylesheet, check: checkFunctionSimple},
		{name: "recursive", xsl: functionRecursiveStylesheet, check: checkFunctionRecursive},
	})
}

const temporaryTreeVariableStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>
  <xsl:template match="rec">
    <xsl:variable name="t"><x><xsl:value-of select="@id"/></x></xsl:variable>
    <r><xsl:value-of select="$t"/></r>
  </xsl:template>
</xsl:stylesheet>`

const temporaryTreeFunctionStylesheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="urn:bench" exclude-result-prefixes="xs f">
  <xsl:function name="f:wrap" as="element()">
    <xsl:param name="id" as="xs:string"/>
    <w><xsl:value-of select="$id"/></w>
  </xsl:function>
  <xsl:template match="/recs"><out><xsl:apply-templates select="rec"/></out></xsl:template>
  <xsl:template match="rec"><xsl:copy-of select="f:wrap(@id)"/></xsl:template>
</xsl:stylesheet>`

// checkTemporaryTreeVariable expects record i to output the string value of
// its temporary tree, its own id.
func checkTemporaryTreeVariable(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords, strings.Count(out, "<r>"))
	require.Contains(tb, out, "<r>r0</r><r>r1</r>")
}

// checkTemporaryTreeFunction expects record i to output a copy of the element
// the function built for it.
func checkTemporaryTreeFunction(tb testing.TB, out string) {
	tb.Helper()
	require.Equal(tb, featureBenchRecords, strings.Count(out, "<w>"))
	require.Contains(tb, out, "<w>r0</w><w>r1</w>")
}

// BenchmarkTemporaryTree times xslt3.Transform for instructions that build a
// small temporary tree once per record over the 5000-record document, so the
// cost of setting up each new tree's document dominates:
//
//   - variable: an xsl:variable whose content is one element holding the
//     record id, read back through its string value;
//   - function: an xsl:function whose body builds one element holding the
//     record id, copied into the result.
func BenchmarkTemporaryTree(b *testing.B) {
	runFeatureBenchCases(b, []featureBenchCase{
		{name: "variable", xsl: temporaryTreeVariableStylesheet, check: checkTemporaryTreeVariable},
		{name: "function", xsl: temporaryTreeFunctionStylesheet, check: checkTemporaryTreeFunction},
	})
}
