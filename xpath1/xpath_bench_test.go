package xpath1_test

import (
	"os"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

const benchChapterExpr = "count(/EXAMPLE/chapter[p])"

func benchmarkChapters(b *testing.B) *helium.Document {
	b.Helper()
	data, err := os.ReadFile("../testdata/libxml2-compat/xpath/docs/chapters")
	require.NoError(b, err)
	doc, err := helium.NewParser().Parse(b.Context(), data)
	require.NoError(b, err)
	return doc
}

func BenchmarkEvaluateCompiledChapters(b *testing.B) {
	doc := benchmarkChapters(b)
	expr, err := xpath1.Compile(benchChapterExpr)
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
	require.Equal(b, float64(5), result.Number)
}

func BenchmarkEvaluateConvenienceChapters(b *testing.B) {
	doc := benchmarkChapters(b)
	ctx := b.Context()
	var result *xpath1.Result
	var err error

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result, err = xpath1.Evaluate(ctx, doc, benchChapterExpr)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	require.Equal(b, xpath1.NumberResult, result.Type)
	require.Equal(b, float64(5), result.Number)
}
