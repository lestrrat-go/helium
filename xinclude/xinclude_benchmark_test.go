package xinclude_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xinclude"
)

// The measured operation resolves and parses a real local include, then splices
// it into a fresh parser-produced document. Input parsing happens outside time.
func BenchmarkProcessXML(b *testing.B) {
	dir := filepath.Join("..", "testdata", "libxml2-compat", "xinclude")
	for _, tc := range []struct {
		name  string
		files []string
		count int
	}{
		{"include.xml", []string{"something.xml"}, 1},
		{"issue733.xml", []string{"issue733-1.xml", "issue733-2.xml", "issue733.dtd"}, 3},
	} {
		b.Run(tc.name, func(b *testing.B) {
			input, err := os.ReadFile(filepath.Join(dir, "docs", tc.name))
			if err != nil {
				b.Fatal(err)
			}
			fsys := fstest.MapFS{}
			for _, name := range tc.files {
				target, err := os.ReadFile(filepath.Join(dir, "ents", name))
				if err != nil {
					b.Fatal(err)
				}
				fsys["ents/"+name] = &fstest.MapFile{Data: target}
			}
			processor := xinclude.NewProcessor().BaseURI("docs/" + tc.name).
				Resolver(xinclude.NewFSResolver(fsys)).NoXIncludeMarkers()
			parser := helium.NewParser()
			newDoc := func() *helium.Document {
				b.Helper()
				doc, err := parser.Parse(b.Context(), input)
				if err != nil {
					b.Fatal(err)
				}
				return doc
			}
			if count, err := processor.Process(b.Context(), newDoc()); err != nil || count != tc.count {
				b.Fatalf("initial include: count=%d, err=%v", count, err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				b.StopTimer()
				doc := newDoc()
				b.StartTimer()
				count, err := processor.Process(b.Context(), doc)
				if err != nil || count != tc.count {
					b.Fatalf("include: count=%d, err=%v", count, err)
				}
			}
		})
	}
}
