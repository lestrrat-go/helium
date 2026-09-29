package xpath1_test

import (
	"os"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

const benchChapterExpr = "count(/EXAMPLE/chapter[p])"
const benchCustomerExpr = "count(/Root/Customers[@CustomerID])"

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

func BenchmarkEvaluateCompiledCustomers(b *testing.B) {
	benchmarkCompiledXPath(b, "../testdata/qt3ts/testdata/nw_Customers.xml", benchCustomerExpr, 91)
}

func BenchmarkEvaluateConvenienceCustomers(b *testing.B) {
	benchmarkConvenienceXPath(b, "../testdata/qt3ts/testdata/nw_Customers.xml", benchCustomerExpr, 91)
}
