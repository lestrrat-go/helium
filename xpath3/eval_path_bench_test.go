package xpath3_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// runPathBench evaluates expr against doc repeatedly. It covers the location
// step path: axis traversal, node testing, and the document ordering every
// step ends with.
func runPathBench(b *testing.B, doc helium.Node, expr string) {
	b.Helper()
	compiled := xpath3.NewCompiler().MustCompile(expr)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	ctx := b.Context()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := eval.Evaluate(ctx, compiled, doc); err != nil {
			b.Fatal(err)
		}
	}
}

// runPerNodeBench evaluates expr once per node selected by selectExpr, the
// shape of callers that drive one short relative expression per context node.
func runPerNodeBench(b *testing.B, selectExpr, expr string) {
	b.Helper()
	doc := buildLargeDoc(b, 1000)
	r, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(b.Context(), xpath3.NewCompiler().MustCompile(selectExpr), doc)
	require.NoError(b, err)
	nodes, err := r.Nodes()
	require.NoError(b, err)
	require.NotEmpty(b, nodes)

	compiled := xpath3.NewCompiler().MustCompile(expr)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	ctx := b.Context()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, n := range nodes {
			if _, err := eval.Evaluate(ctx, compiled, n); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPathDescendant(b *testing.B) {
	runPathBench(b, buildLargeDoc(b, 1000), "//item")
}

func BenchmarkPathChild(b *testing.B) {
	runPathBench(b, buildLargeDoc(b, 1000), "/root/item/val")
}

func BenchmarkPathAttribute(b *testing.B) {
	runPathBench(b, buildLargeDoc(b, 1000), "/root/item/@id")
}

func BenchmarkPathUnion(b *testing.B) {
	runPathBench(b, buildLargeDoc(b, 1000), "//item | //val")
}

// BenchmarkPathPositionalPredicate reads the context position in a predicate
// after a `//` step, so the step runs from 1000 context nodes.
func BenchmarkPathPositionalPredicate(b *testing.B) {
	runPathBench(b, buildLargeDoc(b, 1000), "//item[1]")
}

// BenchmarkPathNonAxisStep ends in a parenthesized step, which is a path
// step expression rather than an axis step.
func BenchmarkPathNonAxisStep(b *testing.B) {
	runPathBench(b, buildLargeDoc(b, 1000), "/root/item/(val)")
}

// BenchmarkPathNestedChild covers a child step whose context nodes sit at
// different depths, which needs the document-order sort.
func BenchmarkPathNestedChild(b *testing.B) {
	var buf strings.Builder
	buf.WriteString("<root>")
	for range 250 {
		buf.WriteString("<a><b/><a><b/><b/></a></a>")
	}
	buf.WriteString("</root>")
	doc, err := helium.NewParser().Parse(b.Context(), []byte(buf.String()))
	require.NoError(b, err)
	runPathBench(b, doc, "//a/b")
}

func BenchmarkPathPerNode(b *testing.B) {
	runPerNodeBench(b, "//item", "val")
}

func BenchmarkPathReverseAxisPerNode(b *testing.B) {
	runPerNodeBench(b, "//val", "ancestor::*")
}

// BenchmarkPathUnionPerNode evaluates the identity-transform select
// expression once per item: its operands are the attributes and the
// children of one element, already in document order when concatenated.
func BenchmarkPathUnionPerNode(b *testing.B) {
	runPerNodeBench(b, "//item", "@*|node()")
}

// BenchmarkPathUnionSwappedPerNode swaps the operands of
// BenchmarkPathUnionPerNode, so the union has to merge them.
func BenchmarkPathUnionSwappedPerNode(b *testing.B) {
	runPerNodeBench(b, "//item", "node()|@*")
}

func benchmarkPathDocument(b *testing.B, path string) *helium.Document {
	b.Helper()
	data, err := os.ReadFile(path)
	require.NoError(b, err)
	doc, err := helium.NewParser().Parse(b.Context(), data)
	require.NoError(b, err)
	return doc
}

func BenchmarkPathChapters(b *testing.B) {
	runPathBench(b, benchmarkPathDocument(b, "../testdata/libxml2-compat/xpath/docs/chapters"), "count(/EXAMPLE/chapter[p])")
}

func BenchmarkPathNVDEntries(b *testing.B) {
	runPathBench(b, benchmarkPathDocument(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml"), "count(//*[local-name()='entry'])")
}

// The NVD string-value benchmarks compute element and document string values:
// a predicate on each entry's whole subtree text, a predicate on many
// text-only elements, and the whole document's text.
func BenchmarkPathNVDEntryStrings(b *testing.B) {
	runPathBench(b, benchmarkPathDocument(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml"), "count(//*[local-name()='entry'][contains(., 'remote')])")
}

func BenchmarkPathNVDRefStrings(b *testing.B) {
	runPathBench(b, benchmarkPathDocument(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml"), "count(//*[local-name()='ref'][. != ''])")
}

func BenchmarkPathNVDDocumentString(b *testing.B) {
	runPathBench(b, benchmarkPathDocument(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml"), "string-length(string(/))")
}

// descendantBenchNVDNamespace is the default namespace of the NVD feed, bound
// to the prefix n so the descendant benchmarks can name its elements.
const descendantBenchNVDNamespace = "http://nvd.nist.gov/feeds/cve/1.2"

// descendantBenchCase is one expression of BenchmarkDescendantPaths. The
// xpath1 package runs the same names and expressions, so the two engines can
// be compared with benchstat.
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
	{"exists", "exists(//n:ref)"},
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
	{"exists", "exists(//val)"},
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
// feed and on a synthetic 40,000-element document. The xpath1 package has a
// benchmark of the same name and cases.
func BenchmarkDescendantPaths(b *testing.B) {
	nvd := benchmarkPathDocument(b, "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml")
	synthetic := buildDescendantBenchDoc(b)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Namespaces(map[string]string{"n": descendantBenchNVDNamespace})
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

func runDescendantBench(b *testing.B, eval xpath3.Evaluator, doc *helium.Document, expr string) {
	compiled := xpath3.NewCompiler().MustCompile(expr)
	ctx := b.Context()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := eval.Evaluate(ctx, compiled, doc); err != nil {
			b.Fatal(err)
		}
	}
}
