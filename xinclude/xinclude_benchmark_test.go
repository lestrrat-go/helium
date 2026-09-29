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
	input, err := os.ReadFile(filepath.Join(dir, "docs", "include.xml"))
	if err != nil {
		b.Fatal(err)
	}
	target, err := os.ReadFile(filepath.Join(dir, "ents", "something.xml"))
	if err != nil {
		b.Fatal(err)
	}
	fsys := fstest.MapFS{"ents/something.xml": &fstest.MapFile{Data: target}}
	processor := xinclude.NewProcessor().BaseURI("docs/include.xml").
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
	if count, err := processor.Process(b.Context(), newDoc()); err != nil || count != 1 {
		b.Fatalf("initial include: count=%d, err=%v", count, err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		doc := newDoc()
		b.StartTimer()
		count, err := processor.Process(b.Context(), doc)
		if err != nil || count != 1 {
			b.Fatalf("include: count=%d, err=%v", count, err)
		}
	}
}
