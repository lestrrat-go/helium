package xslt3_test

import (
	"context"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
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
//     bounded XTDE0820 error rather than a Go stack overflow.

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
// rather than running the whole 1..N range to completion.
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
// / availability=false rather than ErrResourceTooLarge) instead of draining
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
// refused (errors) rather than fetched. This proves the loader never reaches
// out to an attacker-controlled endpoint on its own.
// ---------------------------------------------------------------------------

// TestResourceBudgetDefaultDenyHostileDoc proves doc() against a hostile
// http(s) URL is refused (FODC0002) when no HTTPClient / URIResolver is
// configured, rather than performing an SSRF-style fetch.
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
// against a hostile URL reports unavailable (rather than fetching) when no
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
// promptly rather than producing every secondary result.
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
// binary would crash with a fatal "stack overflow" runtime error rather than
// returning an ordinary error.
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
