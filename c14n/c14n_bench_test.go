package c14n_test

import (
	"os"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/stretchr/testify/require"
)

func benchmarkC14NDocument(b *testing.B, path string) *helium.Document {
	b.Helper()
	data, err := os.ReadFile(path)
	require.NoError(b, err)
	doc, err := helium.NewParser().Parse(b.Context(), data)
	require.NoError(b, err)
	return doc
}

func BenchmarkCanonicalize(b *testing.B) {
	for _, input := range []struct {
		name string
		path string
	}{
		{"C14NFixture", "../testdata/libxml2-compat/c14n/exc-without-comments/test/test-0.xml"},
		{"NVDEntries", "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml"},
	} {
		doc := benchmarkC14NDocument(b, input.path)
		for _, tc := range []struct {
			name string
			mode c14n.Mode
		}{
			{"Inclusive10", c14n.C14N10},
			{"Exclusive10", c14n.ExclusiveC14N10},
		} {
			b.Run(input.name+"/"+tc.name, func(b *testing.B) {
				canonicalizer := c14n.NewCanonicalizer(tc.mode)
				var output []byte
				var err error
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					output, err = canonicalizer.CanonicalizeTo(doc)
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				require.NotEmpty(b, output)
			})
		}
	}
}
