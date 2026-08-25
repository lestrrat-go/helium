package relaxng_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// benchElementA is the single `<element name="a"><empty/></element>` pattern
// both benchmark workloads repeat.
const benchElementA = `<element name="a"><empty/></element>`

// benchGrammar compiles a RELAX NG grammar from a string, failing the
// benchmark on any compile error. It deliberately duplicates compileGrammar
// instead of calling it: compileGrammar takes a *testing.T, and this file must
// stay self-contained so it can be dropped verbatim into a base-revision
// checkout (see the BenchmarkGroupBacktrack doc comment).
func benchGrammar(b *testing.B, schema string) *relaxng.Grammar {
	b.Helper()

	doc, err := helium.NewParser().Parse(b.Context(), []byte(schema))
	require.NoError(b, err)

	grammar, err := relaxng.NewCompiler().Compile(b.Context(), doc)
	require.NoError(b, err)

	return grammar
}

// benchDoc parses `<root>` with n `<a/>` children.
func benchDoc(b *testing.B, n int) *helium.Document {
	b.Helper()

	var src strings.Builder
	src.WriteString(`<root>`)
	for range n {
		src.WriteString(`<a/>`)
	}
	src.WriteString(`</root>`)

	doc, err := helium.NewParser().Parse(b.Context(), []byte(src.String()))
	require.NoError(b, err)

	return doc
}

// BenchmarkGroupBacktrack measures the two validation workloads this change
// targets, so the before/after figures are recorded rather than merely
// asserted against the coarse ceiling in TestGroupBacktrackAllocationBound.
//
// "flexible group backtracking" drives backtrackGroupFlexible: a group of a
// greedy zeroOrMore followed by a mandatory member, over 1600 children. The
// backtracker replays the same child range many times, so a per-repetition
// deep copy of the remaining sibling slice compounds quadratically.
// "plain zeroOrMore, no backtracking" drives the ordinary repetition path over
// 16000 children, where every repetition step used to copy the whole remaining
// tail once.
//
// Grammar compilation and document parsing sit outside the timed region: only
// Validate is measured.
//
// Run with:
//
//	go test ./relaxng -run none -bench BenchmarkGroupBacktrack -benchmem
//
// Baseline procedure. This file uses no symbol introduced by this change, so
// it runs unmodified on the base revision:
//
//	git archive origin/main | tar -x -C <scratch>
//	cp relaxng/group_backtrack_bench_test.go <scratch>/relaxng/
//	go test -C <scratch> ./relaxng -run none -bench BenchmarkGroupBacktrack -benchmem -benchtime=1x
//
// Use a small -benchtime on the base side: one iteration of the first workload
// takes seconds and allocates over 20 GB.
//
// Measured on linux/amd64, go1.26.6, AMD Ryzen 9 7900X3D, against base
// ec7cf3f4 (-benchtime=1x on the base side, default -benchtime on this side):
//
//	flexible group backtracking, 1600 children
//	  base: 4341622568 ns/op  23166291560 B/op  1298591 allocs/op
//	  here:    1090871 ns/op       975732 B/op    11242 allocs/op
//
//	plain zeroOrMore, 16000 children
//	  base:  586837773 ns/op   2108558320 B/op    16052 allocs/op
//	  here:    1450759 ns/op      1190274 B/op       21 allocs/op
func BenchmarkGroupBacktrack(b *testing.B) {
	b.Run("flexible group backtracking", func(b *testing.B) {
		const m = 1600
		grammar := benchGrammar(b, `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="root"><group>`+
			`<zeroOrMore>`+benchElementA+`</zeroOrMore>`+benchElementA+`</group></element></start></grammar>`)
		doc := benchDoc(b, m)
		validator := relaxng.NewValidator(grammar)
		ctx := b.Context()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if err := validator.Validate(ctx, doc); err != nil {
				b.Fatalf("failed to validate group(zeroOrMore(a), a) over %d elements: %s", m, err)
			}
		}
	})

	b.Run("plain zeroOrMore, no backtracking", func(b *testing.B) {
		const m = 16000
		grammar := benchGrammar(b, `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="root">`+
			`<zeroOrMore>`+benchElementA+`</zeroOrMore></element></start></grammar>`)
		doc := benchDoc(b, m)
		validator := relaxng.NewValidator(grammar)
		ctx := b.Context()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if err := validator.Validate(ctx, doc); err != nil {
				b.Fatalf("failed to validate zeroOrMore(a) over %d elements: %s", m, err)
			}
		}
	})
}
