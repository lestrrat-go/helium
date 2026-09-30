package xslt3_test

import (
	"io"
	"os"
	"runtime/metrics"
	"strconv"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

type fuzzURIResolver struct{}

func (fuzzURIResolver) Resolve(string) (io.ReadCloser, error) {
	return nil, os.ErrNotExist
}

type fuzzPackageResolver struct{}

func (fuzzPackageResolver) ResolvePackage(string, string) (io.ReadCloser, string, error) {
	return nil, "", os.ErrNotExist
}

const fuzzStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="name(/*)"/></out></xsl:template>
</xsl:stylesheet>`

const fuzzSource = `<?xml version="1.0"?><root><item>1</item></root>`

// defaultMaxInputAllocs is how many heap objects one fuzz input may allocate in
// parse+compile (or compile+transform) before the harness flags it. The
// allocation count measures the work an input causes, and the same input
// allocates the same count on every run and every machine, so a flagged input
// replays as a flagged input.
//
// Go's fuzzing worker already turns a genuine hang into a crasher — it wraps
// each fuzz call in a 10s deadlock detector (internal/fuzz worker.go:
// panic("deadlocked!")), and its coordinator records the offending input when
// the worker panics or dies. What that net misses is the heavy-but-finite
// input: it completes, so no deadlock fires, and it silently drags the run's
// throughput toward the overall fuzztime deadline — the aggregate slowdown that
// surfaces only as an unactionable "context deadline exceeded" with no
// reproducer. Counting each input's allocations inline and failing via
// t.Errorf when it crosses this bound makes the fuzzing engine persist those
// exact bytes as a crasher (CI's existing "Failing input written to"
// collection then uploads them).
//
// A transform running a million xsl:for-each bodies allocates about 56 million
// objects, so the default of 100 million is several seconds of work on any
// machine, and ordinary stylesheets stay orders of magnitude below it. It is
// overridable via HELIUM_FUZZ_MAX_ALLOCS (a decimal object count). Work that
// allocates nothing is not counted; the deadlock detector remains the net for
// such an input.
const defaultMaxInputAllocs = 100_000_000

// maxInputAllocs returns the allocation bound, honoring HELIUM_FUZZ_MAX_ALLOCS.
func maxInputAllocs() uint64 {
	if v := os.Getenv("HELIUM_FUZZ_MAX_ALLOCS"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return defaultMaxInputAllocs
}

// heapAllocs returns the process's cumulative count of heap allocations. A
// fuzz worker runs one input at a time, so the difference between two reads
// around an input is that input's work.
func heapAllocs() uint64 {
	sample := [1]metrics.Sample{{Name: "/gc/heap/allocs:objects"}}
	metrics.Read(sample[:])
	return sample[0].Value.Uint64()
}

// flagIfHeavy fails the current fuzz input when it allocated more than
// maxInputAllocs objects since start, so the fuzzing engine captures its bytes
// as a reproducer. It runs via defer in the fuzz goroutine, so a panic in the
// code under test still unwinds through testing's normal recovery (an ordinary
// minimizable crasher), and the check simply does not fire on that path.
func flagIfHeavy(t *testing.T, start uint64, stage string) {
	if n := heapAllocs() - start; n > maxInputAllocs() {
		t.Errorf("xslt3 %s allocated %d objects (> %d) on this input; captured as a heavy-input crasher", stage, n, maxInputAllocs())
	}
}

func fuzzCompiler() xslt3.Compiler {
	return xslt3.NewCompiler().
		BaseURI("file:///fuzz/main.xsl").
		URIResolver(fuzzURIResolver{}).
		PackageResolver(fuzzPackageResolver{})
}

func FuzzCompile(f *testing.F) {
	f.Add([]byte(fuzzStylesheet))
	f.Add([]byte(`<?xml version="1.0"?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"><xsl:template match="/"><out/></xsl:template></xsl:stylesheet>`))
	f.Add([]byte(``))
	f.Add([]byte(`not a stylesheet`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		defer flagIfHeavy(t, heapAllocs(), "parse+compile")

		doc, err := helium.NewParser().Parse(t.Context(), data)
		if err != nil {
			return
		}

		_, _ = fuzzCompiler().Compile(t.Context(), doc)
	})
}

func FuzzTransform(f *testing.F) {
	f.Add([]byte(fuzzStylesheet))
	f.Add([]byte(`<?xml version="1.0"?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"><xsl:template match="/"><out>ok</out></xsl:template></xsl:stylesheet>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		defer flagIfHeavy(t, heapAllocs(), "compile+transform")

		styleDoc, err := helium.NewParser().Parse(t.Context(), data)
		if err != nil {
			return
		}

		ss, err := fuzzCompiler().Compile(t.Context(), styleDoc)
		if err != nil {
			return
		}

		sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(fuzzSource))
		if err != nil {
			t.Fatalf("parse source doc: %v", err)
		}

		_, _ = ss.Transform(sourceDoc).Serialize(t.Context())
	})
}
