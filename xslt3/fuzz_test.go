package xslt3_test

import (
	"io"
	"os"
	"testing"
	"time"

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

// slowInputThreshold bounds how long a single fuzz input may spend in
// parse+compile (or transform) before the harness flags it. Go's fuzzing worker
// already turns a genuine hang into a crasher — it wraps each fuzz call in a 10s
// deadlock detector (internal/fuzz worker.go: panic("deadlocked!")), and its
// coordinator records the offending input when the worker panics or dies. What
// that 10s net misses is the slow-but-finite input (say 3-8s): it completes, so
// no deadlock fires, and it silently drags the run's throughput toward the
// overall fuzztime deadline — the aggregate slowdown that surfaces only as an
// unactionable "context deadline exceeded" with no reproducer. Timing each input
// inline and failing via t.Errorf when it crosses this threshold makes the
// fuzzing engine persist those exact bytes as a crasher (CI's existing
// "Failing input written to" collection then uploads them). The threshold MUST
// stay below Go's 10s worker deadline to fire first; it defaults to 5s (normal
// compiles finish in well under a second even under load, so this leaves ample
// headroom for CI scheduler jitter) and is overridable via HELIUM_FUZZ_SLOW_INPUT
// (a Go duration, e.g. "8s" or "500ms").
func slowInputThreshold() time.Duration {
	if v := os.Getenv("HELIUM_FUZZ_SLOW_INPUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 5 * time.Second
}

// flagIfSlow fails the current fuzz input when it ran past the slow-input
// threshold, so the fuzzing engine captures its bytes as a reproducer. It runs
// via defer in the fuzz goroutine, so a panic in the code under test still
// unwinds through testing's normal recovery (an ordinary minimizable crasher),
// and the elapsed check simply does not fire on that path.
func flagIfSlow(t *testing.T, start time.Time, stage string) {
	if d := time.Since(start); d >= slowInputThreshold() {
		t.Errorf("xslt3 %s took %s (>= %s) on this input; captured as a slow-input crasher", stage, d, slowInputThreshold())
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

		defer flagIfSlow(t, time.Now(), "parse+compile")

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

		defer flagIfSlow(t, time.Now(), "compile+transform")

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
