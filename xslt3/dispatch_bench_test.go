package xslt3_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

// buildDispatchStylesheet builds a stylesheet with nRules name-specific
// template rules (only the last one of which ever matches) plus one rule for
// the elements that actually occur in the source document.
//
// The matching rule is declared FIRST so the equal-priority "later
// declaration wins" tiebreak (XSLT 3.0 6.4) puts it at the END of the sorted
// mode list: every unused rule is tested before it, which is the worst case a
// linear scan hits, and the ordinary case for a real stylesheet whose general
// rules are declared early.
func buildDispatchStylesheet(nRules int) string {
	var b strings.Builder
	b.WriteString(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">`)
	b.WriteString(`<xsl:template match="/"><out><xsl:apply-templates select="//item"/></out></xsl:template>`)
	b.WriteString(`<xsl:template match="item"><i><xsl:value-of select="@n"/></i></xsl:template>`)
	for i := range nRules {
		fmt.Fprintf(&b, `<xsl:template match="unused%d"><x/></xsl:template>`, i)
	}
	b.WriteString(`</xsl:stylesheet>`)
	return b.String()
}

// buildDispatchPathStylesheet is the same shape but every unused rule is a
// two-step path pattern, so each rejected probe also walks an ancestor.
func buildDispatchPathStylesheet(nRules int) string {
	var b strings.Builder
	b.WriteString(`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">`)
	b.WriteString(`<xsl:template match="/"><out><xsl:apply-templates select="//item"/></out></xsl:template>`)
	for i := range nRules {
		fmt.Fprintf(&b, `<xsl:template match="a%d//b%d/c%d"><x/></xsl:template>`, i, i, i)
	}
	b.WriteString(`<xsl:template match="item"><i><xsl:value-of select="@n"/></i></xsl:template>`)
	b.WriteString(`</xsl:stylesheet>`)
	return b.String()
}

func buildDispatchSource(nItems int) []byte {
	var b strings.Builder
	b.WriteString(`<doc>`)
	for i := range nItems {
		fmt.Fprintf(&b, `<item n="%d">v%d</item>`, i, i)
	}
	b.WriteString(`</doc>`)
	return []byte(b.String())
}

func runDispatchBench(b *testing.B, ssSrc string, nItems int) {
	b.Helper()
	p := helium.NewParser()
	sdoc, err := p.Parse(b.Context(), []byte(ssSrc))
	if err != nil {
		b.Fatal(err)
	}
	ss, err := xslt3.NewCompiler().Compile(b.Context(), sdoc)
	if err != nil {
		b.Fatal(err)
	}
	src, err := p.Parse(b.Context(), buildDispatchSource(nItems))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		if _, err := xslt3.Transform(b.Context(), src, ss); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDispatchRules sweeps the number of never-matching name-test rules
// ahead of the matching one in the sorted mode list.
//
// Measured base-vs-PR on 2026-08-25, linux/amd64, AMD Ryzen 9 7900X3D, with:
//
//	go test ./xslt3 -run '^$' -bench 'BenchmarkDispatch' -benchtime=1s -count=6
//
// The BASELINE is that identical command run inside a `git archive origin/main`
// extraction with THIS FILE copied into it unmodified. The benchmark touches
// only helium.NewParser/Parse, xslt3.NewCompiler().Compile and xslt3.Transform,
// all of which already exist on the base, so it compiles there without edits.
// That is why no pre-change, unindexed dispatch path is kept in the tree to
// benchmark against: the base revision itself is the control.
//
// Medians of 6 runs, compared with benchstat:
//
//	rules   base sec/op   indexed sec/op   delta
//	    1        2.403m           2.458m        ~
//	   10        2.806m           2.561m        ~
//	   50        3.259m           2.334m   -28.4%
//	  100        3.844m           2.430m   -36.8%
//	  200        5.343m           2.098m   -60.7%
//	  400        9.406m           1.950m   -79.3%
//	  800       17.346m           1.819m   -89.5%
//
// The claim is the SCALING, not any single figure. On the base the cost grows
// linearly in the rule count once the rule term dominates the fixed per-node
// transform cost: 200 to 800 rules is 4x the rules and 5.343m to 17.346m is
// 3.2x the time. With the index the same sweep is FLAT, 2.458m at 1 rule and
// 1.819m at 800.
//
// BenchmarkDispatchPathRules behaves the same way over 1..400 rules: the base
// runs 2.270m to 9.916m while the indexed build runs 2.150m to 2.927m.
// BenchmarkDispatchNodes pins the rule count at 200 and confirms both sides
// stay linear in node count, the indexed one at roughly a third of the slope
// (at 2000 nodes, 26.549m base against 7.551m indexed).
//
// Allocations are unchanged on every point (within 0.1% of the base for both
// B/op and allocs/op), so the saving is in probes performed, not in garbage.
//
// These are wall-clock figures from one machine, recorded as evidence for the
// change. Nothing in the test suite asserts them, and CI does not run
// benchmarks as pass/fail.
func BenchmarkDispatchRules(b *testing.B) {
	for _, n := range []int{1, 10, 50, 100, 200, 400, 800} {
		b.Run(fmt.Sprintf("name/rules=%d", n), func(b *testing.B) { runDispatchBench(b, buildDispatchStylesheet(n), 500) })
	}
}

// BenchmarkDispatchPathRules is BenchmarkDispatchRules with two-step path
// patterns instead of bare name tests.
func BenchmarkDispatchPathRules(b *testing.B) {
	for _, n := range []int{1, 10, 50, 100, 200, 400} {
		b.Run(fmt.Sprintf("path/rules=%d", n), func(b *testing.B) { runDispatchBench(b, buildDispatchPathStylesheet(n), 500) })
	}
}

// BenchmarkDispatchNodes holds the rule count fixed and sweeps the number of
// dispatched nodes, confirming the node-count factor is separately linear.
func BenchmarkDispatchNodes(b *testing.B) {
	for _, n := range []int{125, 250, 500, 1000, 2000} {
		b.Run(fmt.Sprintf("nodes=%d", n), func(b *testing.B) { runDispatchBench(b, buildDispatchStylesheet(200), n) })
	}
}
