package bench_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
)

var (
	smallXML  []byte // ~109 KB
	mediumXML []byte // ~196 KB
	largeXML  []byte // ~3.3 MB
	loadOnce  sync.Once
	repoRoot  string
)

func findRepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("cannot find repo root (go.mod)")
		}
		dir = parent
	}
}

func loadCorpus(b *testing.B) {
	b.Helper()
	loadOnce.Do(func() {
		repoRoot = findRepoRoot()
		var err error
		smallXML, err = os.ReadFile(filepath.Join(repoRoot, "testdata/qt3ts/testdata/fsx_NS.xml"))
		if err != nil {
			b.Fatal(err)
		}
		mediumXML, err = os.ReadFile(filepath.Join(repoRoot, "testdata/xslt30/testdata/tests/insn/merge/cities-SE.xml"))
		if err != nil {
			b.Fatal(err)
		}
		largeXML, err = os.ReadFile(filepath.Join(repoRoot, "testdata/xslt30/testdata/tests/strm/docs/ot.xml"))
		if err != nil {
			b.Fatal(err)
		}
	})
}

var corpus = []struct {
	name string
	data *[]byte
}{
	{"109KB", &smallXML},
	{"196KB", &mediumXML},
	{"3MB", &largeXML},
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
				for {
					_, err := dec.Token()
					if err != nil {
						break
					}
				}
			}
		})
	}
}
