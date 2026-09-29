package xpath1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// buildBenchDoc generates a document with n <item> children, each carrying a
// @cat attribute alternating between "a" and "b" and a <val> element child.
func buildBenchDoc(b *testing.B, n int) *helium.Document {
	b.Helper()
	var buf strings.Builder
	buf.WriteString("<root>")
	for i := range n {
		cat := "a"
		if i%2 == 1 {
			cat = "b"
		}
		fmt.Fprintf(&buf, `<item cat="%s" id="%d"><val>%d</val></item>`, cat, i, i*10)
	}
	buf.WriteString("</root>")
	doc, err := helium.NewParser().Parse(b.Context(), []byte(buf.String()))
	require.NoError(b, err)
	return doc
}

// runEvalBench evaluates expr against the whole document repeatedly. It covers
// the location-step path: axis traversal, node testing, and the document-order
// dedup every step ends with.
func runEvalBench(b *testing.B, expr string) {
	b.Helper()
	doc := buildBenchDoc(b, 1000)
	compiled := xpath1.MustCompile(expr)
	eval := xpath1.NewEvaluator()
	ctx := b.Context()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := eval.Evaluate(ctx, compiled, doc); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEvaluateDescendantPath(b *testing.B) {
	runEvalBench(b, "//item")
}

func BenchmarkEvaluateDescendantPredicate(b *testing.B) {
	runEvalBench(b, `//item[@cat="a"]`)
}

func BenchmarkEvaluateChildPath(b *testing.B) {
	runEvalBench(b, "/root/item/val")
}

func BenchmarkEvaluateAttributePath(b *testing.B) {
	runEvalBench(b, "/root/item/@id")
}

func BenchmarkEvaluateUnion(b *testing.B) {
	runEvalBench(b, "//item | //val")
}

// BenchmarkEvaluatePerNode mirrors how schematron drives xpath1: one short
// relative expression evaluated once per context node.
func BenchmarkEvaluatePerNode(b *testing.B) {
	doc := buildBenchDoc(b, 1000)
	items, err := xpath1.NewEvaluator().Evaluate(b.Context(), xpath1.MustCompile("//item"), doc)
	require.NoError(b, err)
	nodes := items.NodeSet
	require.NotEmpty(b, nodes)

	compiled := xpath1.MustCompile("val")
	eval := xpath1.NewEvaluator()
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

// BenchmarkEvaluateDescendantChildPath covers a child step whose 1000 context
// nodes are siblings, so their results are already in document order.
func BenchmarkEvaluateDescendantChildPath(b *testing.B) {
	runEvalBench(b, "//item/val")
}

// BenchmarkEvaluateNestedChildPath covers a child step whose context nodes
// sit at different depths, which still needs the document-order sort.
func BenchmarkEvaluateNestedChildPath(b *testing.B) {
	var buf strings.Builder
	buf.WriteString("<root>")
	for range 250 {
		buf.WriteString("<a><b/><a><b/><b/></a></a>")
	}
	buf.WriteString("</root>")
	doc, err := helium.NewParser().Parse(b.Context(), []byte(buf.String()))
	require.NoError(b, err)
	compiled := xpath1.MustCompile("//a/b")
	eval := xpath1.NewEvaluator()
	ctx := b.Context()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := eval.Evaluate(ctx, compiled, doc); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEvaluateReverseAxisPerNode evaluates a reverse-axis step once per
// context node, the shape of per-node filters such as xmldsig1's XPath
// transform.
func BenchmarkEvaluateReverseAxisPerNode(b *testing.B) {
	doc := buildBenchDoc(b, 1000)
	vals, err := xpath1.NewEvaluator().Evaluate(b.Context(), xpath1.MustCompile("//val"), doc)
	require.NoError(b, err)
	nodes := vals.NodeSet
	require.NotEmpty(b, nodes)

	compiled := xpath1.MustCompile("ancestor::*")
	eval := xpath1.NewEvaluator()
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
