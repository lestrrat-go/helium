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
