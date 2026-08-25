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
//
// The attribute prefix "a" is what drives that scan deep. declareNS returns
// without recording anything when the prefix is already bound to the same
// URI, so a -> urn:attr is declared once, in the outermost scope, and every
// WriteAttributeNS below it scans every open scope to reach that one entry.
// The element prefix "p" on its own would not show the cost: each level
// rebinds p to a fresh URI in its own scope, which a scan from the innermost
// scope outward finds in constant work.
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
