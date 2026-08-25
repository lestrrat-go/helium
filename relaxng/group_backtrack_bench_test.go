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
// it runs unmodified on the base revision. Resolve the base at run time and
// archive the same object you name as the base: this branch is rebased as main
// moves, so no hardcoded hash stays correct.
//
//	BASE=$(git merge-base HEAD origin/main)
//	mkdir -p <scratch>
//	git archive "$BASE" | tar -x -C <scratch>
//	cp relaxng/group_backtrack_bench_test.go <scratch>/relaxng/
//	go test -C <scratch> ./relaxng -run none -bench BenchmarkGroupBacktrack -benchmem -benchtime=1x
//
// Use a small -benchtime on the base side: one iteration of the first workload
// takes seconds and allocates over 20 GB.
//
// B/op and allocs/op are the evidence this change rests on. They are what the
// change actually removes, and unlike wall clock they do reproduce: across runs
// the base figures hold to within about 0.1% and the figures on this side to
// within a few bytes. Measured against the merge base of this branch with
// origin/main, -benchtime=1x on the base side and the default -benchtime here:
//
//	flexible group backtracking, 1600 children
//	  base: 23166291560 B/op  1298591 allocs/op   (~23 GB, ~1.3M allocations)
//	  here:      975732 B/op    11242 allocs/op   (~976 KB, ~11K allocations)
//
//	plain zeroOrMore, 16000 children
//	  base:  2108558320 B/op    16052 allocs/op   (~2.1 GB)
//	  here:     1190274 B/op       21 allocs/op   (~1.2 MB)
//
// Wall clock is indicative only. On linux/amd64, go1.26.6, AMD Ryzen 9 7900X3D:
//
//	flexible group backtracking, 1600 children:  base ~4.3 s/op, here ~1.1 ms/op
//	plain zeroOrMore, 16000 children:            base ~590 ms/op, here ~1.5 ms/op
//
// ns/op is machine-dependent, and a spread of roughly +/-20% across machines and
// across runs on one machine is expected, so read these as orders of magnitude
// and not as figures to reproduce digit for digit. The recorded run took its
// base side at ec7cf3f4, which was the merge base at the time.
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
