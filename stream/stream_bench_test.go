package stream_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/lestrrat-go/helium/stream"
)

// BenchmarkStartElementNSDepth measures StartElementNS/EndElement cost as a
// function of nesting depth, with each level declaring its own namespace URI.
// lookupNS and hasDefaultNSInScope walk the entire open-scope stack on every
// namespaced write, so this workload is quadratic in depth unless namespace
// lookups are backed by a current-binding map instead of a stack scan.
func BenchmarkStartElementNSDepth(b *testing.B) {
	for _, depth := range []int{500, 1000, 2000, 4000} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			for range b.N {
				w := stream.NewWriter(io.Discard)
				for d := range depth {
					uri := fmt.Sprintf("urn:level-%d", d)
					if err := w.StartElementNS("p", "e", uri); err != nil {
						b.Fatal(err)
					}
					if err := w.WriteAttributeNS("a", "k", "urn:attr", "v"); err != nil {
						b.Fatal(err)
					}
				}
				for range depth {
					if err := w.EndElement(); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
