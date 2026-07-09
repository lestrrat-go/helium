package xslt3_test

import (
	"context"
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
// parse+compile (or transform) before the harness treats it as a stall. A stuck
// input otherwise pins the fuzz worker until the whole run's fuzztime deadline,
// surfacing only as an unactionable "context deadline exceeded" with no
// reproducer written. Flagging the input with t.Errorf instead makes the fuzzing
// engine persist its exact bytes under testdata/fuzz/<Target>/ (which CI's
// "Collect failing corpus" step then uploads), so a recurrence hands us the
// reproducing corpus entry directly. Normal compiles finish in well under a
// second even under load; the default leaves generous headroom for CI scheduler
// jitter and only fires on a genuine stall. Override with HELIUM_FUZZ_SLOW_INPUT
// (a Go duration, e.g. "45s" or "5s").
func slowInputThreshold() time.Duration {
	if v := os.Getenv("HELIUM_FUZZ_SLOW_INPUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 30 * time.Second
}

// finishesWithinBudget reports whether the watchdog-run work signalled done
// before the slow-input threshold elapsed. On a stall it returns false; the work
// goroutine is left to unwind on its own — the fuzz function's context is
// cancelled when the test ends, and the failing run then terminates the process,
// so the abandoned goroutine cannot accumulate across inputs.
func finishesWithinBudget(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	case <-time.After(slowInputThreshold()):
		return false
	}
}

func fuzzCompiler() xslt3.Compiler {
	return xslt3.NewCompiler().
		BaseURI("file:///fuzz/main.xsl").
		URIResolver(fuzzURIResolver{}).
		PackageResolver(fuzzPackageResolver{})
}

func parseAndCompile(ctx context.Context, data []byte, done chan<- struct{}) {
	defer close(done)

	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		return
	}

	_, _ = fuzzCompiler().Compile(ctx, doc)
}

func parseCompileTransform(ctx context.Context, data []byte, done chan<- struct{}) {
	defer close(done)

	styleDoc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		return
	}

	ss, err := fuzzCompiler().Compile(ctx, styleDoc)
	if err != nil {
		return
	}

	// fuzzSource is a fixed, valid document; a parse failure here would be a
	// regression caught by the ordinary tests, and t.Fatal is unsafe off the test
	// goroutine, so just drop out.
	sourceDoc, err := helium.NewParser().Parse(ctx, []byte(fuzzSource))
	if err != nil {
		return
	}

	_, _ = ss.Transform(sourceDoc).Serialize(ctx)
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

		done := make(chan struct{})
		go parseAndCompile(t.Context(), data, done)
		if !finishesWithinBudget(done) {
			t.Errorf("xslt3 parse+compile stalled past %s on this input; captured as a crasher", slowInputThreshold())
		}
	})
}

func FuzzTransform(f *testing.F) {
	f.Add([]byte(fuzzStylesheet))
	f.Add([]byte(`<?xml version="1.0"?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"><xsl:template match="/"><out>ok</out></xsl:template></xsl:stylesheet>`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		done := make(chan struct{})
		go parseCompileTransform(t.Context(), data, done)
		if !finishesWithinBudget(done) {
			t.Errorf("xslt3 compile+transform stalled past %s on this input; captured as a crasher", slowInputThreshold())
		}
	})
}
