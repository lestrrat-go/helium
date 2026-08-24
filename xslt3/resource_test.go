package xslt3_test

import (
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// These tests probe the engine's OPERATIONAL ENVELOPE under pathological but
// VALID stylesheets and source documents — behavior the W3C conformance suites
// do not exercise. Each test constructs input that would run unbounded (or read
// unbounded) if the corresponding guard were absent, then asserts BOUNDED
// behavior: a honored context cancellation, or a configured resource-limit
// error. The design principle is that every test FAILS if the guard is removed
// (the transform would run to completion returning a nil error / full result)
// and PASSES only when the guard fires.
//
// The guards under test are the real engine knobs:
//   - context.Context cancellation checked inside every hot instruction loop
//     (xsl:for-each, xsl:for-each-group, xsl:apply-templates, xsl:iterate,
//     xsl:analyze-string) and inside the xpath3 evaluator (per-op countOps);
//   - Invocation.MaxResourceBytes / xslt3.MaxResourceBytes + ErrResourceTooLarge
//     bounding external-resource reads and xsl:analyze-string match enumeration;
//   - the default-deny resource loader (no URIResolver / HTTPClient => retrieval
//     is refused, never a host fetch);
//   - the maxRecursionDepth guard turning unbounded template recursion into a
//     bounded XTDE0820 error, and never a Go stack overflow.

// ---------------------------------------------------------------------------
// Cancellation: an already-cancelled context must be honored promptly by every
// hot instruction loop. A non-cancelling engine returns a full result with a
// nil error, so asserting the returned error IS the context error is a
// timing-independent proof that the guard fired.
// ---------------------------------------------------------------------------

// feCancelStylesheet drives a large xsl:for-each over the range 1..N. Without
// the per-item ctx.Err() check in execForEach the loop would emit N elements.
const feCancelStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="1 to 5000000">
        <n><xsl:value-of select="."/></n>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`

// iterateCancelStylesheet drives a large xsl:iterate over 1..N.
const iterateCancelStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:iterate select="1 to 5000000">
        <n><xsl:value-of select="."/></n>
      </xsl:iterate>
    </out>
  </xsl:template>
</xsl:stylesheet>`

// fegCancelStylesheet drives a large xsl:for-each-group group-by over 1..N.
const fegCancelStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each-group select="1 to 200000" group-by=". mod 7">
        <g><xsl:value-of select="current-grouping-key()"/></g>
      </xsl:for-each-group>
    </out>
  </xsl:template>
</xsl:stylesheet>`

// analyzeCancelStylesheet drives an xsl:analyze-string whose regex matches at
// every position over a long input. The per-match ctx.Err() check must bail.
const analyzeCancelStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="s" select="string-join(for $i in 1 to 100000 return 'a', '')"/>
    <out>
      <xsl:analyze-string select="$s" regex=".">
        <xsl:matching-substring><m/></xsl:matching-substring>
      </xsl:analyze-string>
    </out>
  </xsl:template>
</xsl:stylesheet>`

// xpathSeqCancelStylesheet forces a large pure-XPath sequence computation. The
// xpath3 evaluator's per-op countOps ctx check must bail even though no XSLT
// instruction loop iterates (the whole cost is inside one select expression).
const xpathSeqCancelStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="sum(for $i in 1 to 100000000 return $i)"/></out>
  </xsl:template>
</xsl:stylesheet>`

// applyTemplatesCancelStylesheet applies templates to a large node-set.
const applyTemplatesCancelStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:apply-templates select="root/item"/></out>
  </xsl:template>
  <xsl:template match="item"><n><xsl:value-of select="@i"/></n></xsl:template>
</xsl:stylesheet>`

// buildWideSource returns a source document with n <item> children, used to
// give xsl:apply-templates a large node-set to iterate.
func buildWideSource(t *testing.T, n int) *helium.Document {
	t.Helper()
	var b strings.Builder
	b.WriteString("<root>")
	for i := range n {
		b.WriteString("<item i=\"")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("\"/>")
	}
	b.WriteString("</root>")
	doc, err := helium.NewParser().Parse(t.Context(), []byte(b.String()))
	require.NoError(t, err)
	return doc
}

// TestResourceBudgetCancellationHonored proves that an already-cancelled
// context is honored promptly by each hot loop (xsl:for-each, xsl:iterate,
// xsl:for-each-group, xsl:analyze-string, xsl:apply-templates) and by the
// xpath3 evaluator. Each stylesheet would run unbounded work if cancellation
// were ignored; asserting the returned error IS context.Canceled proves the
// guard fired (a non-cancelling engine would return a full result, nil error).
func TestResourceBudgetCancellationHonored(t *testing.T) {
	tests := []struct {
		name  string
		xsl   string
		wide  bool // use the wide node-set source instead of <root/>
		guard string
	}{
		{name: "for-each", xsl: feCancelStylesheet, guard: "execForEach per-item ctx.Err()"},
		{name: "iterate", xsl: iterateCancelStylesheet, guard: "execIterate per-item ctx.Err()"},
		{name: "for-each-group", xsl: fegCancelStylesheet, guard: "execForEachGroup per-group ctx.Err()"},
		{name: "analyze-string", xsl: analyzeCancelStylesheet, guard: "execAnalyzeString per-match ctx.Err()"},
		{name: "xpath-sequence", xsl: xpathSeqCancelStylesheet, guard: "xpath3 countOps per-op ctx.Err()"},
		{name: "apply-templates", xsl: applyTemplatesCancelStylesheet, wide: true, guard: "execApplyTemplates per-node ctx.Err()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := compileStylesheetString(t, tt.xsl)

			var source *helium.Document
			if tt.wide {
				source = buildWideSource(t, 20000)
			} else {
				source = parseTransformSource(t)
			}

			// Cancel BEFORE running: the transform runs no early transform-level
			// ctx check, so the only way it can surface context.Canceled is by a
			// hot loop / evaluator guard actually consulting the context.
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			_, err := ss.Transform(source).Serialize(ctx)
			require.Error(t, err, "%s: cancelled transform must error, not complete", tt.guard)
			require.ErrorIs(t, err, context.Canceled,
				"%s: error must wrap context.Canceled, got %v", tt.guard, err)
		})
	}
}

// TestResourceBudgetDeadlinePreemption proves mid-execution preemption: a short
// deadline set on a genuinely long-running xsl:for-each must trip DURING the
// loop (after transform setup completes), returning context.DeadlineExceeded
// well short of running the whole 1..N range to completion.
func TestResourceBudgetDeadlinePreemption(t *testing.T) {
	// Nested loops keep every individual range well under the xpath3 node-set
	// length cap (10M) so the cost is in the ITERATION COUNT (~25M inner bodies),
	// not one giant materialization. A non-preempting engine would run for
	// seconds; a preempting one returns ~at the deadline.
	const deadlineStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:for-each select="1 to 5000">
        <xsl:for-each select="1 to 5000">
          <n><xsl:value-of select=". * 2"/></n>
        </xsl:for-each>
      </xsl:for-each>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	ss := compileStylesheetString(t, deadlineStylesheet)
	source := parseTransformSource(t)

	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := ss.Transform(source).Serialize(ctx)
		done <- err
	}()

	select {
	case err := <-done:
		require.Error(t, err, "deadline transform must error, not complete")
		require.ErrorIs(t, err, context.DeadlineExceeded,
			"error must wrap context.DeadlineExceeded, got %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("transform did not honor the deadline: it kept running long past it (unbounded)")
	}
}

// ---------------------------------------------------------------------------
// Resource limits: MaxResourceBytes must bound external-resource reads. An
// oversized (effectively infinite) resolver stream must be rejected after at
// most the cap, not read to exhaustion.
// ---------------------------------------------------------------------------

// oversizedResolver serves an effectively infinite byte stream for every URI it
// is asked to resolve, recording the TOTAL bytes actually read. A read that is
// not bounded would never terminate; asserting the recorded total is at most
// the configured cap proves the read was bounded.
type oversizedResolver struct {
	totalRead atomic.Int64
}

func (r *oversizedResolver) ResolveURI(_ string) (io.ReadCloser, error) {
	return &infiniteStream{parent: r}, nil
}

type infiniteStream struct {
	parent *oversizedResolver
}

func (s *infiniteStream) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	s.parent.totalRead.Add(int64(len(p)))
	return len(p), nil
}

func (s *infiniteStream) Close() error { return nil }

const docResourceStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="doc('mem://big.xml')/*"/></out>
  </xsl:template>
</xsl:stylesheet>`

const sourceDocResourceStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:source-document href="mem://big.xml">
      <out><xsl:value-of select="/*"/></out>
    </xsl:source-document>
  </xsl:template>
</xsl:stylesheet>`

const unparsedTextResourceStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="unparsed-text-available('mem://big.txt')"/></out>
  </xsl:template>
</xsl:stylesheet>`

// TestResourceBudgetMaxResourceBytesDoc proves Invocation.MaxResourceBytes
// bounds a doc()/document() read: the oversized resolver stream is rejected
// with ErrResourceTooLarge after reading at most cap+1 bytes, not read to
// exhaustion (which would never return).
func TestResourceBudgetMaxResourceBytesDoc(t *testing.T) {
	const limitBytes = int64(1024)
	ss := compileStylesheetString(t, docResourceStylesheet)
	source := parseTransformSource(t)
	res := &oversizedResolver{}

	_, err := ss.Transform(source).
		URIResolver(res).
		MaxResourceBytes(limitBytes).
		Serialize(t.Context())

	require.Error(t, err, "doc() over an oversized stream must error")
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"doc() over-cap read must surface ErrResourceTooLarge, got %v", err)
	require.LessOrEqual(t, res.totalRead.Load(), limitBytes+64,
		"read must stop at the cap, not drain the stream (read %d bytes)", res.totalRead.Load())
}

// TestResourceBudgetMaxResourceBytesSourceDocument proves the same cap bounds
// xsl:source-document retrieval.
func TestResourceBudgetMaxResourceBytesSourceDocument(t *testing.T) {
	const limitBytes = int64(1024)
	ss := compileStylesheetString(t, sourceDocResourceStylesheet)
	source := parseTransformSource(t)
	res := &oversizedResolver{}

	_, err := ss.Transform(source).
		URIResolver(res).
		MaxResourceBytes(limitBytes).
		Serialize(t.Context())

	require.Error(t, err, "xsl:source-document over an oversized stream must error")
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"source-document over-cap read must surface ErrResourceTooLarge, got %v", err)
	require.LessOrEqual(t, res.totalRead.Load(), limitBytes+64,
		"read must stop at the cap (read %d bytes)", res.totalRead.Load())
}

// TestResourceBudgetMaxResourceBytesUnparsedText proves the same cap bounds an
// fn:unparsed-text read: an over-cap resource makes unparsed-text-available
// report false (per the documented contract, the built-in surfaces FOUT1170
// / availability=false, and never ErrResourceTooLarge) instead of draining
// the stream.
func TestResourceBudgetMaxResourceBytesUnparsedText(t *testing.T) {
	const limitBytes = int64(1024)
	ss := compileStylesheetString(t, unparsedTextResourceStylesheet)
	source := parseTransformSource(t)
	res := &oversizedResolver{}

	result, err := ss.Transform(source).
		URIResolver(res).
		MaxResourceBytes(limitBytes).
		Serialize(t.Context())

	require.NoError(t, err)
	require.Contains(t, result, "<out>false</out>",
		"unparsed-text-available must report false for an over-cap resource")
	require.LessOrEqual(t, res.totalRead.Load(), limitBytes+64,
		"read must stop at the cap (read %d bytes)", res.totalRead.Load())
}

// TestResourceBudgetAnalyzeStringMatchCap proves the resource cap doubles as
// the xsl:analyze-string match-count ceiling: a match-everywhere regex over an
// input longer than the (tiny) cap is rejected with ErrResourceTooLarge rather
// than enumerating a match per character.
func TestResourceBudgetAnalyzeStringMatchCap(t *testing.T) {
	const analyzeStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:variable name="s" select="string-join(for $i in 1 to 5000 return 'a', '')"/>
    <out>
      <xsl:analyze-string select="$s" regex=".">
        <xsl:matching-substring><m/></xsl:matching-substring>
      </xsl:analyze-string>
    </out>
  </xsl:template>
</xsl:stylesheet>`

	ss := compileStylesheetString(t, analyzeStylesheet)
	source := parseTransformSource(t)

	// A tiny cap (100) makes the 5000-character input trip the match ceiling.
	_, err := ss.Transform(source).
		MaxResourceBytes(100).
		Serialize(t.Context())

	require.Error(t, err, "analyze-string over the match cap must error")
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"over-cap match enumeration must surface ErrResourceTooLarge, got %v", err)
}

// ---------------------------------------------------------------------------
// Default-deny: with NO resolver / HTTPClient, a hostile-URL retrieval must be
// refused (errors), and never fetched. This proves the loader never reaches
// out to an attacker-controlled endpoint on its own.
// ---------------------------------------------------------------------------

// TestResourceBudgetDefaultDenyHostileDoc proves doc() against a hostile
// http(s) URL is refused (FODC0002) when no HTTPClient / URIResolver is
// configured, performing no SSRF-style fetch.
func TestResourceBudgetDefaultDenyHostileDoc(t *testing.T) {
	const hostileDocStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="doc('http://169.254.169.254/latest/meta-data/')/*"/></out>
  </xsl:template>
</xsl:stylesheet>`

	ss := compileStylesheetString(t, hostileDocStylesheet)
	source := parseTransformSource(t)

	_, err := ss.Transform(source).Serialize(t.Context())
	require.Error(t, err, "doc() of a hostile URL must be denied without a resolver")
	require.Contains(t, err.Error(), "FODC0002",
		"default-deny doc() must surface FODC0002, got %v", err)
}

// TestResourceBudgetDefaultDenyHostileUnparsedText proves fn:unparsed-text
// against a hostile URL reports unavailable, and fetches nothing, when no
// resolver is configured.
func TestResourceBudgetDefaultDenyHostileUnparsedText(t *testing.T) {
	const hostileTextStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="unparsed-text-available('http://169.254.169.254/latest/meta-data/')"/></out>
  </xsl:template>
</xsl:stylesheet>`

	ss := compileStylesheetString(t, hostileTextStylesheet)
	source := parseTransformSource(t)

	result, err := ss.Transform(source).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>false</out>",
		"unparsed-text-available of a hostile URL must be false without a resolver")
}

// ---------------------------------------------------------------------------
// Fanout: xsl:result-document fanout over a large loop must honor cancellation
// promptly, producing no more secondary results.
// ---------------------------------------------------------------------------

// countingResultDocHandler counts the secondary result documents it receives.
type countingResultDocHandler struct {
	count atomic.Int64
}

func (h *countingResultDocHandler) HandleResultDocument(_ string, _ *helium.Document, _ *xslt3.OutputDef) error {
	h.count.Add(1)
	return nil
}

// TestResourceBudgetResultDocumentFanoutCancellation proves that a huge
// xsl:result-document fanout is bounded by cancellation: with an already-
// cancelled context the enclosing xsl:for-each bails before it can emit the
// N secondary documents, so the handler receives far fewer than N.
func TestResourceBudgetResultDocumentFanoutCancellation(t *testing.T) {
	const fanoutStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:for-each select="1 to 1000000">
      <xsl:result-document href="out{.}.xml">
        <n><xsl:value-of select="."/></n>
      </xsl:result-document>
    </xsl:for-each>
  </xsl:template>
</xsl:stylesheet>`

	ss := compileStylesheetString(t, fanoutStylesheet)
	source := parseTransformSource(t)
	handler := &countingResultDocHandler{}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := ss.Transform(source).
		ResultDocumentHandler(handler).
		Do(ctx)

	require.Error(t, err, "cancelled fanout must error, not complete")
	require.ErrorIs(t, err, context.Canceled,
		"fanout cancellation must wrap context.Canceled, got %v", err)
	require.Less(t, handler.count.Load(), int64(1000),
		"cancelled fanout must not produce the full result set (got %d)", handler.count.Load())
}

// ---------------------------------------------------------------------------
// Deep recursion: unbounded template recursion must degrade to a bounded
// XTDE0820 error, never a Go stack overflow that crashes the process.
// ---------------------------------------------------------------------------

// TestResourceBudgetUnboundedRecursionIsBounded proves that a template that
// recurses without a base case hits the maxRecursionDepth guard and returns
// XTDE0820 instead of overflowing the stack. If the guard were absent the test
// binary would crash with a fatal "stack overflow" runtime error, returning no
// ordinary error.
func TestResourceBudgetUnboundedRecursionIsBounded(t *testing.T) {
	const recursiveStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="deep">
    <xsl:call-template name="deep"/>
  </xsl:template>
  <xsl:template match="/">
    <out><xsl:call-template name="deep"/></out>
  </xsl:template>
</xsl:stylesheet>`

	ss := compileStylesheetString(t, recursiveStylesheet)
	source := parseTransformSource(t)

	_, err := ss.Transform(source).Serialize(t.Context())
	require.Error(t, err, "unbounded recursion must return an error, not run forever / crash")
	require.Contains(t, err.Error(), "XTDE0820",
		"unbounded recursion must surface the recursion-depth error XTDE0820, got %v", err)
}

// These regressions pin the invariant that a non-default per-resource cap
// configured on the Compiler (or effective Invocation) is threaded into every
// nested compile / stylesheet construction site, so resource reads never
// silently fall back to the default MaxResourceBytes cap.

// resolverFunc adapts a function to the xslt3.URIResolver interface used at
// compile time for xsl:import / xsl:include / simplified-stylesheet loads.
type resolverFunc func(uri string) (io.ReadCloser, error)

func (f resolverFunc) Resolve(uri string) (io.ReadCloser, error) { return f(uri) }

// capPackageResolver serves a single fixed package source for any name, plus
// satisfies any nested xsl:include the package issues via a companion
// URIResolver supplied on the Compiler.
type capPackageResolver struct {
	source  []byte
	baseURI string
}

func (r capPackageResolver) ResolvePackage(_ string, _ string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader(string(r.source))), r.baseURI, nil
}

// A simplified (literal-result-element) stylesheet must honor the Compiler's
// MaxResourceBytes cap on its own runtime resource reads. Before the fix the
// cap was dropped during compileSimplified, so an over-cap fn:doc read used the
// default 10 MiB bound instead of the configured one.
func TestSimplifiedStylesheetHonorsCompilerCap(t *testing.T) {
	t.Parallel()

	const u = "http://example.invalid/big.xml"
	body := "<root>" + strings.Repeat("a", 4096) + "</root>"

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != u {
			return nil, &xpath3.XPathError{Code: "FOUT1170", Message: "not found: " + uri}
		}
		return io.NopCloser(strings.NewReader(body)), nil
	})

	// Simplified stylesheet: root is a literal result element with xsl:version.
	simplified := `<out xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xsl:version="3.0">` +
		`<xsl:copy-of select="doc('` + u + `')"/></out>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(simplified))
	require.NoError(t, err)

	// Compile with a cap far below the resource size.
	ss, err := xslt3.NewCompiler().MaxResourceBytes(64).Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"simplified stylesheet must honor the Compiler MaxResourceBytes cap")
}

// xsl:use-package bounds the package FILE read, but the cap must also flow into
// the COMPILE of that package so its own xsl:include loads are bounded. Before
// the fix the nested include fell back to the default cap.
func TestUsePackageNestedIncludeHonorsCap(t *testing.T) {
	t.Parallel()

	const includeURI = "http://example.invalid/included.xsl"
	// An included module larger than the configured cap.
	includedBody := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<!-- ` + strings.Repeat("x", 4096) + ` -->` +
		`<xsl:template name="helper"><h/></xsl:template>` +
		`</xsl:stylesheet>`

	pkgBody := `<?xml version="1.0"?>` +
		`<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"` +
		` xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:include href="` + includeURI + `"/>` +
		`<xsl:template name="main" visibility="public"><out/></xsl:template>` +
		`</xsl:package>`

	includeResolver := resolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != includeURI {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(includedBody)), nil
	})

	mainSheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:use-package name="http://example.com/pkg"/>` +
		`<xsl:template match="/"><out/></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(mainSheet))
	require.NoError(t, err)

	// Cap below the included module size but above the package file size.
	_, err = xslt3.NewCompiler().
		MaxResourceBytes(512).
		PackageResolver(capPackageResolver{source: []byte(pkgBody)}).
		URIResolver(includeResolver).
		Compile(t.Context(), doc)
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"xsl:use-package nested include must honor the Compiler MaxResourceBytes cap")
}

// Runtime fn:transform bounds the immediate stylesheet read, but the cap must
// also flow into the nested COMPILE so resources loaded while compiling the
// nested stylesheet (e.g. its xsl:include) honor the Invocation cap, and the
// resulting ErrResourceTooLarge stays observable via errors.Is.
func TestFnTransformNestedCompileHonorsInvocationCap(t *testing.T) {
	t.Parallel()

	const nestedLoc = "http://example.invalid/nested.xsl"
	const includeURI = "http://example.invalid/nested-include.xsl"

	// The included module is larger than the configured cap.
	includedBody := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<!-- ` + strings.Repeat("y", 8192) + ` -->` +
		`<xsl:template name="helper"><h/></xsl:template>` +
		`</xsl:stylesheet>`

	// The nested stylesheet itself is small, so its file read passes the cap;
	// only its xsl:include should trip the bound during the nested compile.
	nestedBody := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:include href="` + includeURI + `"/>` +
		`<xsl:template match="/"><nested/></xsl:template>` +
		`</xsl:stylesheet>`

	// fn:transform resolves the stylesheet-location (and the nested module's
	// xsl:include) through the COMPILE-TIME URIResolver carried on the
	// stylesheet, so it is configured on the Compiler. The cap, however, is set
	// only on the Invocation to prove the effective Invocation cap threads into
	// the nested compile.
	resolver := resolverFunc(func(uri string) (io.ReadCloser, error) {
		switch uri {
		case nestedLoc:
			return io.NopCloser(strings.NewReader(nestedBody)), nil
		case includeURI:
			return io.NopCloser(strings.NewReader(includedBody)), nil
		default:
			return nil, os.ErrNotExist
		}
	})

	outer := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:template match="/"><xsl:sequence select="transform(map{` +
		`'stylesheet-location':'` + nestedLoc + `',` +
		`'source-node':.})?output"/></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(outer))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().URIResolver(resolver).Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	// Cap above the nested file but below its include; only propagation into the
	// nested compile makes the include read trip ErrResourceTooLarge.
	_, err = ss.Transform(source).
		MaxResourceBytes(2048).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"fn:transform nested-compile resources must honor the Invocation MaxResourceBytes cap")
}

// xsl:source-document wraps an over-cap resource read in a dynamic error; the
// ErrResourceTooLarge sentinel must survive that wrapping so callers can match
// it via errors.Is, as the public API promises.
func TestSourceDocumentOverCapPreservesSentinel(t *testing.T) {
	t.Parallel()

	const u = "http://example.invalid/big-src.xml"
	body := "<root>" + strings.Repeat("a", 8192) + "</root>"

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != u {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(body)), nil
	})

	sheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:template match="/"><xsl:source-document href="` + u + `"><got/></xsl:source-document></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(sheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).
		MaxResourceBytes(64).
		URIResolver(resolver).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"xsl:source-document over-cap read must preserve ErrResourceTooLarge")
}

// xsl:merge wraps an over-cap resource read in a dynamic error; the
// ErrResourceTooLarge sentinel must survive that wrapping.
func TestMergeOverCapPreservesSentinel(t *testing.T) {
	t.Parallel()

	const u = "http://example.invalid/big-merge.xml"
	body := `<data>` + strings.Repeat(`<row><k>1</k></row>`, 1024) + `</data>`

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != u {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(body)), nil
	})

	sheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:template match="/"><out>` +
		`<xsl:merge>` +
		`<xsl:merge-source select="/" for-each-source="'` + u + `'">` +
		`<xsl:merge-key select="k"/>` +
		`</xsl:merge-source>` +
		`<xsl:merge-action><xsl:sequence select="."/></xsl:merge-action>` +
		`</xsl:merge>` +
		`</out></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(sheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).
		MaxResourceBytes(64).
		URIResolver(resolver).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"xsl:merge over-cap read must preserve ErrResourceTooLarge")
}

// A nested xs:import target loaded under xsl:import-schema that exceeds the
// Compiler MaxResourceBytes cap must FAIL compilation with ErrResourceTooLarge.
// The xsd import path normally demotes an xs:import load failure to a non-fatal
// warning ("Skipping the import."), which would silently defeat the cap for the
// imported schema. The resource-limit case must instead abort compilation, and
// the sentinel must survive across the xsd->xslt3 boundary so callers can
// errors.Is it.
func TestImportSchemaNestedImportOverCapIsFatal(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"
	const mainSchemaURI = "mem:/stylesheets/main.xsd"
	const importedSchemaURI = "mem:/stylesheets/imported.xsd"

	// main.xsd imports a second-namespace schema via xs:import.
	mainSchema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           xmlns:o="http://example.com/o"
           elementFormDefault="qualified">
  <xs:import namespace="http://example.com/o" schemaLocation="imported.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	// The imported schema is padded well past the configured cap.
	importedSchema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/o"
           elementFormDefault="qualified">
  <!-- ` + strings.Repeat("z", 8192) + ` -->
  <xs:element name="other" type="xs:string"/>
</xs:schema>`

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	resolver := fileMapResolver{files: map[string]string{
		mainSchemaURI:     mainSchema,
		importedSchemaURI: importedSchema,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)

	// Cap above the main schema but below the imported schema; only fatal
	// handling of the over-cap nested import surfaces ErrResourceTooLarge.
	_, err = xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		MaxResourceBytes(1024).
		Compile(ctx, doc)
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"over-cap nested xs:import under xsl:import-schema must fail with ErrResourceTooLarge, not be silently skipped")
}

// An xsl:import-schema schema-location that exceeds the Compiler cap must FAIL
// compilation with ErrResourceTooLarge even when a pre-compiled schema for the
// SAME target namespace was registered via Compiler.ImportSchemas. The
// import-schema error branch normally falls back to that pre-compiled schema and
// reports success; a resource-limit breach (or any fatal schema-load) must NOT
// be papered over by that fallback or the byte cap is defeated for the primary
// schema-location load.
func TestImportSchemaOverCapNotMaskedByPrecompiledFallback(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"
	const schemaURI = "mem:/stylesheets/big.xsd"
	const ns = "http://example.com/s"

	// The schema-location target is padded well past the configured cap.
	bigSchema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="` + ns + `"
           elementFormDefault="qualified">
  <!-- ` + strings.Repeat("z", 8192) + ` -->
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	// A small, valid pre-compiled schema for the same namespace, registered as a
	// fallback via Compiler.ImportSchemas. Before the fix this satisfied the
	// import-schema declaration after the over-cap load failed, masking the cap.
	preSchema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="` + ns + `"
           elementFormDefault="qualified">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="` + ns + `">
  <xsl:import-schema namespace="` + ns + `" schema-location="big.xsd"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	preDoc, err := helium.NewParser().Parse(ctx, []byte(preSchema))
	require.NoError(t, err)
	precompiled, err := xsd.NewCompiler().Compile(ctx, preDoc)
	require.NoError(t, err)

	resolver := fileMapResolver{files: map[string]string{schemaURI: bigSchema}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)

	// Cap below the schema-location target. The fallback to the pre-compiled
	// schema must NOT mask the over-cap read.
	_, err = xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		ImportSchemas(precompiled).
		MaxResourceBytes(64).
		Compile(ctx, doc)
	require.Error(t, err,
		"over-cap import-schema schema-location must fail even with a pre-compiled fallback registered")
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"over-cap import-schema must surface ErrResourceTooLarge, not silently fall back to ImportSchemas")
}

// A top-level xsl:import-schema with a schema-location but NO Compiler.URIResolver
// must FAIL compilation with the default-deny denial even when a pre-compiled
// schema for the SAME target namespace was registered via Compiler.ImportSchemas.
// The denial ("no URIResolver configured") is a policy refusal, not a fetch miss,
// so it must NOT be papered over by the pre-compiled fallback — doing so would let
// a no-resolver schema-location silently compile via the registered schema,
// bypassing the secure-by-default policy.
func TestImportSchemaNoResolverNotMaskedByPrecompiledFallback(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"
	const ns = "http://example.com/s"

	// A small, valid pre-compiled schema for the same namespace, registered as a
	// fallback via Compiler.ImportSchemas.
	preSchema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="` + ns + `"
           elementFormDefault="qualified">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="` + ns + `">
  <xsl:import-schema namespace="` + ns + `" schema-location="s.xsd"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	preDoc, err := helium.NewParser().Parse(ctx, []byte(preSchema))
	require.NoError(t, err)
	precompiled, err := xsd.NewCompiler().Compile(ctx, preDoc)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)

	// No URIResolver: the schema-location load is a default-deny policy refusal.
	// The registered pre-compiled schema must NOT mask it.
	_, err = xslt3.NewCompiler().
		BaseURI(baseURI).
		ImportSchemas(precompiled).
		Compile(ctx, doc)
	require.Error(t, err,
		"no-resolver import-schema schema-location must fail even with a pre-compiled fallback registered")
	require.Contains(t, err.Error(), "no URIResolver configured",
		"the default-deny denial message must surface, not a silent fallback to ImportSchemas")
}

// An xsl:import-schema whose schema-location loads fine but whose NESTED
// xs:include escapes its base directory via "../" must FAIL compilation, even
// when a pre-compiled schema for the SAME target namespace is registered via
// Compiler.ImportSchemas. The path-escape sentinel is a plain unexported xsd
// error (not a FatalSchemaLoader interface value), so an interface-only fallback
// guard would miss it and silently substitute the pre-compiled schema — defeating
// the path-traversal guard. The single xsd.IsFatalSchemaLoad classifier must
// short-circuit the fallback. (This test FAILS before the centralized fix and
// PASSES after.)
func TestImportSchemaNestedPathEscapeNotMaskedByPrecompiledFallback(t *testing.T) {
	t.Parallel()

	// Local (non-URI) base so the xsd compiler's "../"-escape guard fires on the
	// nested include (the URI-base branch resolves per RFC 3986 instead).
	const baseURI = "/local/styles/main.xsl"
	const ns = "http://example.com/s"

	// Main schema loads cleanly but pulls in a nested include that climbs above
	// its own directory — a path-traversal escape the xsd compiler rejects with
	// the errSchemaPathEscape sentinel.
	mainSchema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="` + ns + `"
           elementFormDefault="qualified">
  <xs:include schemaLocation="../../../etc/escape.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	// A small, valid pre-compiled schema for the same namespace. Before the fix
	// the escape error fell through to this fallback and compilation reported OK.
	preSchema := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="` + ns + `"
           elementFormDefault="qualified">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="` + ns + `">
  <xsl:import-schema namespace="` + ns + `" schema-location="main.xsd"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	preDoc, err := helium.NewParser().Parse(ctx, []byte(preSchema))
	require.NoError(t, err)
	precompiled, err := xsd.NewCompiler().Compile(ctx, preDoc)
	require.NoError(t, err)

	// Resolver serves the main schema; the escaping nested include is never
	// reachable because the escape is rejected during resolution.
	resolver := fileMapResolver{files: map[string]string{
		"/local/styles/main.xsd": mainSchema,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		ImportSchemas(precompiled).
		Compile(ctx, doc)
	require.Error(t, err,
		"nested xs:include path-escape must fail compilation even with a pre-compiled fallback registered")
}

// A serialization parameter-document over-cap read wraps in a static error
// (XTSE0090) at compile time; the ErrResourceTooLarge sentinel must survive
// that wrapping so callers can match it via errors.Is.
func TestParameterDocumentOverCapPreservesSentinel(t *testing.T) {
	t.Parallel()

	const u = "http://example.invalid/big-params.xml"
	body := `<serialization-parameters xmlns="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<!-- ` + strings.Repeat("p", 8192) + ` -->` +
		`<indent value="yes"/>` +
		`</serialization-parameters>`

	resolver := resolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != u {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(body)), nil
	})

	sheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:output method="xml" parameter-document="` + u + `"/>` +
		`<xsl:template match="/"><out/></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(sheet))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().
		MaxResourceBytes(64).
		URIResolver(resolver).
		Compile(t.Context(), doc)
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"serialization parameter-document over-cap read must preserve ErrResourceTooLarge")
}

// A runtime xsl:result-document parameter-document (an AVT, so it is loaded
// during execution, and not at compile time) that exceeds the cap must FAIL
// the transformation, and must never be silently swallowed. Before the fix the
// runtime loader cached the OutputDef only on success and otherwise continued,
// so the over-cap read never surfaced and callers could not observe the
// ErrResourceTooLarge sentinel.
func TestResultDocumentParameterDocumentOverCapIsFatal(t *testing.T) {
	t.Parallel()

	const u = "http://example.invalid/big-rd-params.xml"
	body := `<serialization-parameters xmlns="http://www.w3.org/2010/xslt-xquery-serialization">` +
		`<!-- ` + strings.Repeat("p", 8192) + ` -->` +
		`<indent value="yes"/>` +
		`</serialization-parameters>`

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != u {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(body)), nil
	})

	// parameter-document is an AVT ({...}) so the load happens at runtime.
	sheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:output name="o" method="xml"/>` +
		`<xsl:template match="/">` +
		`<xsl:result-document href="out.xml" format="o" parameter-document="{'` + u + `'}">` +
		`<got/></xsl:result-document>` +
		`</xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(sheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).
		MaxResourceBytes(64).
		URIResolver(resolver).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"runtime xsl:result-document parameter-document over-cap read must fail with ErrResourceTooLarge")
	// A runtime failure is a dynamic error and must NOT also classify as a
	// static error: errors.Is must not match both taxonomies.
	require.ErrorIs(t, err, xslt3.ErrDynamicError,
		"runtime xsl:result-document parameter-document failure must be a dynamic error")
	require.NotErrorIs(t, err, xslt3.ErrStaticError,
		"runtime xsl:result-document parameter-document failure must NOT also be a static error")
}

// A source schema referenced via xsi:schemaLocation that exceeds the cap must
// FAIL the transformation even under lax (non-strict) validation. The
// schema-location load failure is normally non-fatal when validation is lax,
// but a resource-limit breach must not be demoted that way or the cap is
// defeated for the referenced schema; ErrResourceTooLarge must survive.
func TestSourceSchemaLocationOverCapIsFatal(t *testing.T) {
	t.Parallel()

	const schemaURI = "http://example.invalid/big-src-schema.xsd"
	schemaBody := `<?xml version="1.0"?>` +
		`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` +
		` targetNamespace="http://example.com/n"` +
		` elementFormDefault="qualified">` +
		`<!-- ` + strings.Repeat("z", 8192) + ` -->` +
		`<xs:element name="root" type="xs:string"/>` +
		`</xs:schema>`

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != schemaURI {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(schemaBody)), nil
	})

	// Stylesheet does not declare strict validation, so a schema-location load
	// failure would normally be non-fatal — except for the resource cap.
	sheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:template match="/"><out/></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(sheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	source := `<root xmlns="http://example.com/n"` +
		` xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"` +
		` xsi:schemaLocation="http://example.com/n ` + schemaURI + `">x</root>`
	src, err := helium.NewParser().Parse(t.Context(), []byte(source))
	require.NoError(t, err)

	_, err = ss.Transform(src).
		MaxResourceBytes(64).
		URIResolver(resolver).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"over-cap xsi:schemaLocation source-schema read must fail with ErrResourceTooLarge under lax validation")
}
