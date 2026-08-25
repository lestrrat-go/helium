package html_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/html"
)

// blockedEndTagInput builds the htmlAutoCloseOnClose worst case:
// <html><body><span><div> + n x <b> + n x </span>. hasOnStack finds <span> in
// O(1) (it is index 2, near the bottom), so the cost is entirely in the
// backward pre-scan: it walks all n <b> elements (default priority 100)
// before reaching <div> (priority 150), which aborts the scan without
// popping anything. That abort repeats for every one of the n </span> tags.
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

func timeParseSAX(tb testing.TB, input []byte) time.Duration {
	tb.Helper()
	p := html.NewParser().SuppressErrors(true).SuppressWarnings(true)
	ctx := context.Background()
	// One untimed warm-up run so allocator/GC state does not skew the first
	// timed sample.
	_ = p.ParseWithSAX(ctx, input, &html.SAXCallbacks{})
	start := time.Now()
	_ = p.ParseWithSAX(ctx, input, &html.SAXCallbacks{})
	return time.Since(start)
}

// TestHTMLAutoCloseOnCloseBlockedGrowth guards against the
// htmlAutoCloseOnClose pre-scan going quadratic again. At depth 4,000 and
// 8,000 the input length exactly doubles; a linear pre-scan should therefore
// roughly double in time (measured ~1.9x), while the naive O(depth) backward
// scan quadruples it (confirmed against main: ratio ~4.1x). maxRatio sits
// between those two so the assertion is unambiguous either way, with slack
// for machine noise on the linear side.
func TestHTMLAutoCloseOnCloseBlockedGrowth(t *testing.T) {
	const small = 4000
	const large = 8000

	smallInput := blockedEndTagInput(small)
	largeInput := blockedEndTagInput(large)

	// Take the best of a few samples on each side to reduce scheduler noise.
	const samples = 3
	var smallT, largeT time.Duration
	for i := range samples {
		if d := timeParseSAX(t, smallInput); i == 0 || d < smallT {
			smallT = d
		}
		if d := timeParseSAX(t, largeInput); i == 0 || d < largeT {
			largeT = d
		}
	}

	t.Logf("blocked end-tag scan: n=%d -> %v, n=%d -> %v", small, smallT, large, largeT)

	if smallT <= 0 {
		t.Skip("measured duration too small to compare reliably")
	}
	const maxRatio = 3.0
	ratio := float64(largeT) / float64(smallT)
	if ratio > maxRatio {
		t.Fatalf("htmlAutoCloseOnClose pre-scan looks quadratic again: n=%d..%d ratio=%.2f, want <= %.1f (linear input growth is 2x)",
			small, large, ratio, maxRatio)
	}
}

// BenchmarkParse and BenchmarkParseSAX measure a full pass over the
// libxml2-compat golden corpus (45 files, 118,757 bytes), matching the
// baseline captured in .tmp/htmlstackprobe: SAX 2.156 ms, DOM 5.260 ms per
// pass on main before the stack-index change.
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
	p := html.NewParser().SuppressErrors(true).SuppressWarnings(true)
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		for _, in := range data {
			h := html.SAXCallbacks{}
			_ = p.ParseWithSAX(ctx, in, &h)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	data := loadCorpus(b)
	// Matches the SuppressErrors/SuppressWarnings setting used to capture the
	// DOM baseline (.tmp/htmlstackprobe): without it, error-formatting cost on
	// the golden corpus's error-producing files would dominate the comparison
	// instead of the tree-construction cost this benchmark targets.
	p := html.NewParser().SuppressErrors(true).SuppressWarnings(true)
	ctx := context.Background()
	b.ResetTimer()
	for range b.N {
		for _, in := range data {
			_, _ = p.Parse(ctx, in)
		}
	}
}
