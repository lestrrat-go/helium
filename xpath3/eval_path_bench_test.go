package xpath3_test

import (
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
