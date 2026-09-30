package helium_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
)

// deepContentDocument returns the source of a document whose elements nest
// depth levels deep, each holding the text "x" before its only child element,
// so the root element's Content is depth x's.
func deepContentDocument(depth int) string {
	var b strings.Builder
	b.Grow(depth * 8)
	for range depth {
		b.WriteString("<a>x")
	}
	for range depth {
		b.WriteString("</a>")
	}
	return b.String()
}

// parseDeepContentDocument parses deepContentDocument(depth) with the nesting
// cap lifted.
func parseDeepContentDocument(tb testing.TB, depth int) *helium.Document {
	tb.Helper()
	doc, err := helium.NewParser().MaxDepth(-1).Parse(tb.Context(), []byte(deepContentDocument(depth)))
	if err != nil {
		tb.Fatal(err)
	}
	return doc
}

// Content of a deep chain of elements visits each element once, so its cost
// grows with the number of nodes, not with the square of the depth.
func BenchmarkContentDeepTree(b *testing.B) {
	for _, depth := range []int{256, 4096, 50000} {
		b.Run("depth="+strconv.Itoa(depth), func(b *testing.B) {
			root := parseDeepContentDocument(b, depth).DocumentElement()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if got := len(root.Content()); got != depth {
					b.Fatalf("Content length = %d, want %d", got, depth)
				}
			}
		})
	}
}

// Content of the NVD fixture's root element, a wide and shallow real-world
// document.
func BenchmarkContentNVD(b *testing.B) {
	src, err := os.ReadFile("testdata/libxml2-compat/schemas/test/nvdcve_0.xml")
	if err != nil {
		b.Fatal(err)
	}
	doc, err := helium.NewParser().Parse(b.Context(), src)
	if err != nil {
		b.Fatal(err)
	}
	root := doc.DocumentElement()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if len(root.Content()) == 0 {
			b.Fatal("empty Content")
		}
	}
}
