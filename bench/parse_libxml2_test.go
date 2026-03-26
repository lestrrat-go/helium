//go:build cgo && libxml2bench

package bench_test

import (
	"testing"

	"github.com/lestrrat-go/helium/bench"
)

func BenchmarkLibxml2Parse(b *testing.B) {
	loadCorpus(b)

	bench.Libxml2Init()
	b.Cleanup(bench.Libxml2Cleanup)

	for _, tc := range corpus {
		data := *tc.data
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if !bench.Libxml2ParseAndFree(data) {
					b.Fatal("xmlParseMemory failed")
				}
			}
		})
	}
}
