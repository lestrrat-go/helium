package html_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium/html"
)

// The fixtures exercise HTML entity handling, implied elements, and tree building.
func BenchmarkParseDOM(b *testing.B) {
	for _, name := range []string{"fp40.htm", "wired.html"} {
		b.Run(name, func(b *testing.B) {
			data, err := os.ReadFile(filepath.Join("..", "testdata", "libxml2-compat", "html", name))
			if err != nil {
				b.Fatal(err)
			}
			parser := html.NewParser()
			if _, err := parser.Parse(b.Context(), data); err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				doc, err := parser.Parse(b.Context(), data)
				if err != nil {
					b.Fatal(err)
				}
				if doc == nil {
					b.Fatal("nil document")
				}
			}
		})
	}
}
