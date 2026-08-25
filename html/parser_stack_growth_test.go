package html

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// blockedEndTagInput builds the htmlAutoCloseOnClose worst case:
// <html><body><span><div> + n x <b> + n x </span>. hasOnStack finds <span> in
// O(1) (it is index 2, near the bottom), so the cost is entirely in the
// backward pre-scan: the naive version walks all n <b> elements (default
// priority 100) before reaching <div> (priority 150), which aborts the scan
// without popping anything. That abort repeats for every one of the n </span>
// tags, so the naive pre-scan costs O(n^2) stack visits while the indexed one
// costs O(n).
func blockedEndTagInput(n int) []byte {
	var sb strings.Builder
	sb.WriteString("<html><body><span><div>")
	for range n {
		sb.WriteString("<b>")
	}
	for range n {
		sb.WriteString("</span>")
	}
	return []byte(sb.String())
}

// parseStackScanSteps parses input with SAX events discarded and returns
// parser.stackScanSteps: the exact number of open-element-stack positions the
// htmlAutoCloseOnClose pre-scan examined. The count is a property of the input
// and the algorithm alone, so it is identical on every machine and under any
// load — unlike elapsed time, which is neither.
func parseStackScanSteps(tb testing.TB, input []byte) int64 {
	tb.Helper()
	cfg := NewParser().SuppressErrors(true).SuppressWarnings(true).parseConfig()
	ctx := context.Background()
	p := newParser(ctx, input, &SAXCallbacks{}, cfg)
	if err := p.parse(ctx); err != nil {
		tb.Fatalf("parse: %v", err)
	}
	return p.stackScanSteps
}

// TestHTMLAutoCloseOnCloseBlockedGrowth guards against the
// htmlAutoCloseOnClose pre-scan going quadratic again. At depth 4,000 and
// 8,000 the input length exactly doubles, so a pre-scan whose per-end-tag cost
// is independent of stack depth doubles its total stack-position count, while
// the naive O(depth) backward scan quadruples it. maxRatio sits between those
// two so the assertion is unambiguous either way.
//
// The measurement is a counter, not a clock: stackScanSteps is an exact
// integer determined by the input and the algorithm, so this test gives the
// same verdict on a fast laptop and on a heavily loaded CI runner. A
// wall-clock budget or ratio would not, and raising such a budget only widens
// the tolerance on the same defect.
func TestHTMLAutoCloseOnCloseBlockedGrowth(t *testing.T) {
	const small = 4000
	const large = 8000

	smallSteps := parseStackScanSteps(t, blockedEndTagInput(small))
	largeSteps := parseStackScanSteps(t, blockedEndTagInput(large))

	t.Logf("blocked end-tag pre-scan: n=%d -> %d stack positions, n=%d -> %d stack positions",
		small, smallSteps, large, largeSteps)

	// Lower bound: the pre-scan must examine at least one stack position per
	// blocked </span>. This is what keeps the ratio assertion honest — without
	// it, deleting the stackScanSteps increments would leave both counts at
	// zero and the guard would pass while measuring nothing.
	if smallSteps < small {
		t.Fatalf("pre-scan examined %d stack positions for %d blocked end tags: the stackScanSteps instrumentation is not counting the pre-scan's work",
			smallSteps, small)
	}
	if largeSteps < large {
		t.Fatalf("pre-scan examined %d stack positions for %d blocked end tags: the stackScanSteps instrumentation is not counting the pre-scan's work",
			largeSteps, large)
	}

	const maxRatio = 3.0
	ratio := float64(largeSteps) / float64(smallSteps)
	if ratio > maxRatio {
		t.Fatalf("htmlAutoCloseOnClose pre-scan looks quadratic again: n=%d..%d stack-position ratio=%.2f, want <= %.1f (linear input growth is 2x)",
			small, large, ratio, maxRatio)
	}
}

// BenchmarkParse and BenchmarkParseSAX measure a full pass over the
// libxml2-compat golden corpus (45 files, 118,757 bytes).
func loadCorpus(tb testing.TB) [][]byte {
	tb.Helper()
	dir := "../testdata/libxml2-compat/html"
	entries, err := os.ReadDir(dir)
	if err != nil {
		tb.Skipf("testdata/libxml2-compat/html not found: %v", err)
	}
	var names []string
	for _, fi := range entries {
		name := fi.Name()
		if fi.IsDir() || strings.HasSuffix(name, ".sax") ||
			strings.HasSuffix(name, ".err") || strings.HasSuffix(name, ".expected") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	data := make([][]byte, 0, len(names))
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			tb.Fatalf("reading %s: %v", name, err)
		}
		data = append(data, b)
	}
	return data
}

func BenchmarkParseSAX(b *testing.B) {
	data := loadCorpus(b)
	p := NewParser().SuppressErrors(true).SuppressWarnings(true)
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		for _, in := range data {
			h := SAXCallbacks{}
			_ = p.ParseWithSAX(ctx, in, &h)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	data := loadCorpus(b)
	// Matches the SuppressErrors/SuppressWarnings setting used to capture the
	// DOM baseline: without it, error-formatting cost on the golden corpus's
	// error-producing files would dominate the comparison instead of the
	// tree-construction cost this benchmark targets.
	p := NewParser().SuppressErrors(true).SuppressWarnings(true)
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		for _, in := range data {
			_, _ = p.Parse(ctx, in)
		}
	}
}
