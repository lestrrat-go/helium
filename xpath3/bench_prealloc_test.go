package xpath3_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// buildLargeDoc generates an XML document with n <item> children, each having
// a @cat attribute alternating between "a" and "b", and a <val> child.
func buildLargeDoc(b *testing.B, n int) *helium.Document { //nolint:unparam // n always 1000 but kept for flexibility
	b.Helper()
	var buf strings.Builder
	buf.WriteString("<root>")
	for i := 0; i < n; i++ {
		cat := "a"
		if i%2 == 1 {
			cat = "b"
		}
		fmt.Fprintf(&buf, `<item cat="%s" id="%d"><val>%d</val></item>`, cat, i, i*10)
	}
	buf.WriteString("</root>")
	doc, err := helium.NewParser().Parse(context.Background(), []byte(buf.String()))
	require.NoError(b, err)
	return doc
}

// BenchmarkLargePathPredicate evaluates //item[@cat="a"] on a 1000-element
// document, exercising slice pre-allocation in evalStepWithPredicates and
// applyVMAttributeEqualsStringPredicate.
func BenchmarkLargePathPredicate(b *testing.B) {
	doc := buildLargeDoc(b, 1000)
	expr := xpath3.NewCompiler().MustCompile(`//item[@cat="a"]`)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := eval.Evaluate(context.Background(), expr, doc)
		require.NoError(b, err)
		nodes, _ := result.Nodes()
		require.Equal(b, 500, len(nodes))
	}
}

// BenchmarkLargePathNoPredicate evaluates //item/val on a 1000-element
// document, exercising slice pre-allocation in evalStepNoPredicates.
func BenchmarkLargePathNoPredicate(b *testing.B) {
	doc := buildLargeDoc(b, 1000)
	expr := xpath3.NewCompiler().MustCompile(`//item/val`)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := eval.Evaluate(context.Background(), expr, doc)
		require.NoError(b, err)
		nodes, _ := result.Nodes()
		require.Equal(b, 1000, len(nodes))
	}
}

// BenchmarkLargeIntersect evaluates (//item[@cat="a"]) intersect (//item[@id<500])
// on a 1000-element document, exercising the intersect/except node-set path.
func BenchmarkLargeIntersect(b *testing.B) {
	doc := buildLargeDoc(b, 1000)
	expr := xpath3.NewCompiler().MustCompile(`//item[@cat="a"] intersect //item[@id < 500]`)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := eval.Evaluate(context.Background(), expr, doc)
		require.NoError(b, err)
		nodes, _ := result.Nodes()
		require.True(b, len(nodes) > 0)
	}
}

// BenchmarkLargeSimpleMap evaluates //item ! val on a 1000-element document,
// exercising the simple-map operator path.
func BenchmarkLargeSimpleMap(b *testing.B) {
	doc := buildLargeDoc(b, 1000)
	expr := xpath3.NewCompiler().MustCompile(`//item ! val`)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := eval.Evaluate(context.Background(), expr, doc)
		require.NoError(b, err)
		nodes, _ := result.Nodes()
		require.Equal(b, 1000, len(nodes))
	}
}

// BenchmarkLargeSequencePredicate evaluates (1 to 1000)[. mod 3 = 0],
// exercising applySequencePredicate with a large sequence.
func BenchmarkLargeSequencePredicate(b *testing.B) {
	doc, err := helium.NewParser().Parse(context.Background(), []byte("<r/>"))
	require.NoError(b, err)
	expr := xpath3.NewCompiler().MustCompile(`(1 to 1000)[. mod 3 = 0]`)
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eval.Evaluate(context.Background(), expr, doc)
		require.NoError(b, err)
	}
}
