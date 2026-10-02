package xpath1_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

const benchChapterExpr = "count(/EXAMPLE/chapter[p])"
const benchNVDEntryExpr = "count(//*[local-name()='entry'])"

// String-value workloads: a predicate on each entry's whole subtree text, a
// predicate on many text-only elements, and the whole document's text.
const (
	benchNVDEntryStringExpr = "count(//*[local-name()='entry'][contains(., 'remote')])"
	benchNVDRefStringExpr   = "count(//*[local-name()='ref'][. != ''])"
	benchNVDDocStringExpr   = "string-length(string(/))"
)

func benchmarkXPathDocument(b *testing.B, path string) *helium.Document {
	b.Helper()
	data, err := os.ReadFile(path)
	require.NoError(b, err)
	doc, err := helium.NewParser().Parse(b.Context(), data)
	require.NoError(b, err)
	return doc
}

func benchmarkCompiledXPath(b *testing.B, path, query string, expected float64) {
	doc := benchmarkXPathDocument(b, path)
	expr, err := xpath1.Compile(query)
	require.NoError(b, err)
	ctx := b.Context()
	var result *xpath1.Result

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result, err = expr.Evaluate(ctx, doc)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	require.Equal(b, xpath1.NumberResult, result.Type)
	require.Equal(b, expected, result.Number)
}

func benchmarkConvenienceXPath(b *testing.B, path, query string, expected float64) {
	doc := benchmarkXPathDocument(b, path)
	ctx := b.Context()
	var result *xpath1.Result
	var err error

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result, err = xpath1.Evaluate(ctx, doc, query)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	require.Equal(b, xpath1.NumberResult, result.Type)
	require.Equal(b, expected, result.Number)
}

func BenchmarkEvaluateCompiledChapters(b *testing.B) {
	benchmarkCompiledXPath(b, "../testdata/libxml2-compat/xpath/docs/chapters", benchChapterExpr, 5)
}

func BenchmarkEvaluateConvenienceChapters(b *testing.B) {
	benchmarkConvenienceXPath(b, "../testdata/libxml2-compat/xpath/docs/chapters", benchChapterExpr, 5)
}

func BenchmarkEvaluateCompiledNVDEntries(b *testing.B) {
	benchmarkCompiledXPath(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml", benchNVDEntryExpr, 176)
}

func BenchmarkEvaluateConvenienceNVDEntries(b *testing.B) {
	benchmarkConvenienceXPath(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml", benchNVDEntryExpr, 176)
}

func BenchmarkEvaluateCompiledNVDEntryStrings(b *testing.B) {
	benchmarkCompiledXPath(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml", benchNVDEntryStringExpr, 147)
}

func BenchmarkEvaluateCompiledNVDRefStrings(b *testing.B) {
	benchmarkCompiledXPath(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml", benchNVDRefStringExpr, 484)
}

func BenchmarkEvaluateCompiledNVDDocumentString(b *testing.B) {
	benchmarkCompiledXPath(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml", benchNVDDocStringExpr, 121849)
}

// descendantBenchNVDNamespace is the default namespace of the NVD feed, bound
// to the prefix n so the descendant benchmarks can name its elements.
const descendantBenchNVDNamespace = "http://nvd.nist.gov/feeds/cve/1.2"

// descendantBenchCase is one expression of BenchmarkDescendantPaths. The
// xpath3 package runs the same names and expressions, so the two engines can
// be compared with benchstat. XPath 1.0 has no exists(), so the "exists" case
// uses boolean(), which also stops at the first node.
type descendantBenchCase struct {
	name string
	expr string
}

var descendantBenchNVDCases = []descendantBenchCase{
	{"count_all", "count(//*)"},
	{"all", "//*"},
	{"named", "//n:ref"},
	{"named_attr", "/n:nvd//n:entry/@name"},
	{"count_pred", "count(//n:ref[@url])"},
	{"last", "//*[last()]"},
	{"exists", "boolean(//n:ref)"},
	{"first", "(//n:ref)[1]"},
	{"multi", "/n:nvd/n:entry//n:ref"},
	{"nested", "//*//n:ref"},
}

var descendantBenchSyntheticCases = []descendantBenchCase{
	{"count_all", "count(//*)"},
	{"all", "//*"},
	{"named", "//item"},
	{"named_attr", "/root//item/@id"},
	{"count_pred", "count(//item[@cat])"},
	{"last", "//*[last()]"},
	{"exists", "boolean(//val)"},
	{"first", "(//val)[1]"},
	{"multi", "/root/group//val"},
	{"nested", "//*//val"},
}

// buildDescendantBenchDoc generates 100 <group> elements of 100 <item>
// elements each; every item holds a <val> and a <note> with mixed content.
func buildDescendantBenchDoc(b *testing.B) *helium.Document {
	b.Helper()
	var buf strings.Builder
	buf.WriteString("<root>")
	for g := range 100 {
		buf.WriteString("<group>")
		for i := range 100 {
			cat := "a"
			if i%2 == 1 {
				cat = "b"
			}
			fmt.Fprintf(&buf, `<item cat="%s" id="%d"><val>%d</val><note>text <b>x</b> tail</note></item>`, cat, g*100+i, i)
		}
		buf.WriteString("</group>")
	}
	buf.WriteString("</root>")
	doc, err := helium.NewParser().Parse(b.Context(), []byte(buf.String()))
	require.NoError(b, err)
	return doc
}

// BenchmarkDescendantPaths evaluates descendant-path expressions on the NVD
// feed and on a synthetic 40,000-element document. The xpath3 package has a
// benchmark of the same name and cases.
func BenchmarkDescendantPaths(b *testing.B) {
	nvd := benchmarkXPathDocument(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml")
	synthetic := buildDescendantBenchDoc(b)
	eval := xpath1.NewEvaluator().Namespaces(map[string]string{"n": descendantBenchNVDNamespace})
	for _, tc := range descendantBenchNVDCases {
		b.Run("nvd/"+tc.name, func(b *testing.B) {
			runDescendantBench(b, eval, nvd, tc.expr)
		})
	}
	for _, tc := range descendantBenchSyntheticCases {
		b.Run("synthetic/"+tc.name, func(b *testing.B) {
			runDescendantBench(b, eval, synthetic, tc.expr)
		})
	}
}

func runDescendantBench(b *testing.B, eval xpath1.Evaluator, doc *helium.Document, query string) {
	expr, err := xpath1.Compile(query)
	require.NoError(b, err)
	ctx := b.Context()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := eval.Evaluate(ctx, expr, doc); err != nil {
			b.Fatal(err)
		}
	}
}
