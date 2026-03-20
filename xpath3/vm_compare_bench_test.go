package xpath3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

const (
	benchPathExpr  = `/library/book[@lang="en"]/title`
	benchCountExpr = `count(/library/book[@lang="en"]/title)`
	benchFLWORExpr = `string-join(for $b in /library/book[@lang="en"] return string($b/title), "|")`
)

func benchmarkDoc(b *testing.B) *helium.Document {
	b.Helper()
	doc, err := helium.Parse(b.Context(), []byte(testXML))
	require.NoError(b, err)
	return doc
}

func BenchmarkCompilePath(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		expr, err := xpath3.Compile(benchPathExpr)
		require.NoError(b, err)
		require.NotNil(b, expr)
	}
}

func BenchmarkEvaluateCompiledCount(b *testing.B) {
	doc := benchmarkDoc(b)
	expr := xpath3.MustCompile(benchCountExpr)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := expr.Evaluate(b.Context(), doc)
		require.NoError(b, err)
		value, ok := result.IsNumber()
		require.True(b, ok)
		require.Equal(b, 2.0, value)
	}
}

func BenchmarkEvaluateConvenienceCount(b *testing.B) {
	doc := benchmarkDoc(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := xpath3.Evaluate(b.Context(), doc, benchCountExpr)
		require.NoError(b, err)
		value, ok := result.IsNumber()
		require.True(b, ok)
		require.Equal(b, 2.0, value)
	}
}

func BenchmarkEvaluateCompiledFLWOR(b *testing.B) {
	doc := benchmarkDoc(b)
	expr := xpath3.MustCompile(benchFLWORExpr)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := expr.Evaluate(b.Context(), doc)
		require.NoError(b, err)
		value, ok := result.IsString()
		require.True(b, ok)
		require.Equal(b, "Go Programming|XPath Mastery", value)
	}
}
