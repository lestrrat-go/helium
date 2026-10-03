package bench_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"golang.org/x/text/encoding/htmlindex"
)

var (
	smallXML  []byte // ~118 KB
	mediumXML []byte // ~287 KB
	largeXML  []byte // ~608 KB
	loadOnce  sync.Once
	repoRoot  string
)

func loadCorpus(b *testing.B) {
	b.Helper()
	loadOnce.Do(func() {
		repoRoot = heliumtest.RepoRoot()
		var err error
		smallXML, err = os.ReadFile(filepath.Join(repoRoot, "testdata/libxml2-compat/relaxng/test/spec_0.xml"))
		if err != nil {
			b.Fatal(err)
		}
		mediumXML, err = os.ReadFile(filepath.Join(repoRoot, "testdata/libxml2-compat/schemas/test/nvdcve_0.xml"))
		if err != nil {
			b.Fatal(err)
		}
		largeXML, err = os.ReadFile(filepath.Join(repoRoot, "testdata/libxml2-compat/relaxng/test/comps_0.xml"))
		if err != nil {
			b.Fatal(err)
		}
	})
}

var corpus = []struct {
	name string
	data *[]byte
}{
	{"118KB", &smallXML},
	{"287KB", &mediumXML},
	{"608KB", &largeXML},
}

func BenchmarkHeliumParse(b *testing.B) {
	loadCorpus(b)
	for _, tc := range corpus {
		data := *tc.data
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				doc, err := helium.NewParser().Parse(context.Background(), data)
				if err != nil {
					b.Fatal(err)
				}
				doc.Free()
			}
		})
	}
}

// smallDocs are documents small enough that the fixed per-parse setup cost,
// not the bytes, decides the parse time.
var smallDocs = []struct {
	name string
	data []byte
}{
	{"tiny", []byte(`<a/>`)},
	{"70B", []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<root><item>hello</item></root>")},
	{"1KB", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<catalog xmlns="urn:example:catalog" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:x="urn:example:extra">
  <book id="b1" lang="en" x:rating="4">
    <dc:title>The Art of Parsing</dc:title>
    <dc:creator>A. Author</dc:creator>
    <price currency="USD">29.99</price>
  </book>
  <book id="b2" lang="fr" x:rating="5">
    <dc:title>Arbres et Branches</dc:title>
    <dc:creator>B. Auteur</dc:creator>
    <price currency="EUR">24.50</price>
  </book>
  <book id="b3" lang="de" x:rating="3">
    <dc:title>Knoten und Kanten</dc:title>
    <dc:creator>C. Verfasser</dc:creator>
    <price currency="EUR">19.00</price>
  </book>
  <book id="b4" lang="ja" x:rating="4">
    <dc:title>Namespaces in Practice</dc:title>
    <dc:creator>D. Writer</dc:creator>
    <price currency="JPY">3200</price>
  </book>
  <x:note x:kind="summary">Four books &amp; one note.</x:note>
  <book id="b5" lang="en" x:rating="2">
    <dc:title>Attributes Everywhere</dc:title>
    <dc:creator>E. Scribe</dc:creator>
    <price currency="USD">9.99</price>
  </book>
</catalog>
`)},
}

// BenchmarkHeliumParseSmall times parsing documents small enough that the
// per-parse setup dominates: through Parse with a new Parser per call, through
// ParseReader, and through Parse with one Parser value reused across calls.
func BenchmarkHeliumParseSmall(b *testing.B) {
	for _, tc := range smallDocs {
		data := tc.data
		b.Run(tc.name+"/Parse", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				doc, err := helium.NewParser().Parse(context.Background(), data)
				if err != nil {
					b.Fatal(err)
				}
				doc.Free()
			}
		})
		b.Run(tc.name+"/ParseReader", func(b *testing.B) {
			p := helium.NewParser()
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				doc, err := p.ParseReader(context.Background(), bytes.NewReader(data))
				if err != nil {
					b.Fatal(err)
				}
				doc.Free()
			}
		})
		b.Run(tc.name+"/ReusedParser", func(b *testing.B) {
			p := helium.NewParser()
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				doc, err := p.Parse(context.Background(), data)
				if err != nil {
					b.Fatal(err)
				}
				doc.Free()
			}
		})
	}
}

func BenchmarkStdlibXMLDecode(b *testing.B) {
	loadCorpus(b)
	for _, tc := range corpus {
		data := *tc.data
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				dec := xml.NewDecoder(bytes.NewReader(data))
				// The 118KB fixture declares iso-8859-1, which encoding/xml
				// rejects on the first token without a CharsetReader.
				dec.CharsetReader = stdlibCharsetReader
				for {
					_, err := dec.Token()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func stdlibCharsetReader(label string, input io.Reader) (io.Reader, error) {
	enc, err := htmlindex.Get(label)
	if err != nil {
		return nil, err
	}
	return enc.NewDecoder().Reader(input), nil
}
