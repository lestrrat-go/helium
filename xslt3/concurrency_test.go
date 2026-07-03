package xslt3_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestStylesheetConcurrentReuse verifies the documented concurrency contract:
// a single *Stylesheet returned by CompileStylesheet is immutable after compile
// and safe to Transform concurrently from many goroutines. Each goroutine
// transforms its OWN source document and asserts it gets back exactly the
// output derived from that document — never another goroutine's data. Run under
// -race, any shared-mutable hazard on the compiled stylesheet (a lazily
// populated map, an in-place cache write, cross-transform state bleed) would be
// reported as a data race or surface as a mismatched result.
//
// The stylesheets deliberately exercise SHARED READ STATE that a naive
// implementation might mutate per transform: an xsl:key (whose key table is
// built per transform from the shared compiled key definition), a global
// xsl:variable (evaluated per transform), and xsl:for-each-group grouping.
func TestStylesheetConcurrentReuse(t *testing.T) {
	const (
		goroutines = 50
		iterations = 25
	)

	cases := []struct {
		name       string
		stylesheet string
		// source builds the per-goroutine source document; i is the goroutine
		// index, embedded throughout so any cross-goroutine bleed is detectable.
		source func(i int) string
		// expect returns substrings that MUST appear in goroutine i's output.
		expect func(i int) []string
	}{
		{
			// Exercises a global xsl:variable ($tag) and an xsl:key lookup
			// (key('byId', @want)). The key table is built per transform from
			// the shared compiled key definition; if that build leaked into the
			// stylesheet, concurrent transforms over distinct sources would
			// corrupt each other's lookups.
			name: "keys_and_global_variable",
			stylesheet: `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:key name="byId" match="item" use="@id"/>
  <xsl:variable name="tag" select="/root/@tag"/>
  <xsl:template match="/root">
    <out><tag><xsl:value-of select="$tag"/></tag><val><xsl:value-of select="key('byId', @want)/@val"/></val><n><xsl:value-of select="count(item)"/></n></out>
  </xsl:template>
</xsl:stylesheet>`,
			source: func(i int) string {
				var b strings.Builder
				fmt.Fprintf(&b, `<root tag="T%d" want="k%d-1">`, i, i)
				for j := range 3 {
					fmt.Fprintf(&b, `<item id="k%d-%d" val="v%d-%d"/>`, i, j, i, j)
				}
				b.WriteString(`</root>`)
				return b.String()
			},
			expect: func(i int) []string {
				return []string{
					fmt.Sprintf("<tag>T%d</tag>", i),
					fmt.Sprintf("<val>v%d-1</val>", i),
					"<n>3</n>",
				}
			},
		},
		{
			// Exercises xsl:for-each-group group-by with a sort, plus a global
			// variable ($who). Per-goroutine category cardinalities are a
			// function of i, so a bled current-group()/current-grouping-key()
			// state would produce the wrong counts.
			name: "for_each_group",
			stylesheet: `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:variable name="who" select="/root/@who"/>
  <xsl:template match="/root">
    <out who="{$who}"><xsl:for-each-group select="item" group-by="@cat"><xsl:sort select="current-grouping-key()"/><g cat="{current-grouping-key()}" n="{count(current-group())}"/></xsl:for-each-group></out>
  </xsl:template>
</xsl:stylesheet>`,
			source: func(i int) string {
				aCount := 1 + i%3
				bCount := 1 + (i+1)%3
				var b strings.Builder
				fmt.Fprintf(&b, `<root who="W%d">`, i)
				// Interleave A/B items so grouping actually reorders them.
				for j := 0; j < aCount || j < bCount; j++ {
					if j < aCount {
						fmt.Fprintf(&b, `<item cat="A" v="%d"/>`, j)
					}
					if j < bCount {
						fmt.Fprintf(&b, `<item cat="B" v="%d"/>`, j)
					}
				}
				b.WriteString(`</root>`)
				return b.String()
			},
			expect: func(i int) []string {
				aCount := 1 + i%3
				bCount := 1 + (i+1)%3
				return []string{
					fmt.Sprintf(`who="W%d"`, i),
					fmt.Sprintf(`<g cat="A" n="%d"/>`, aCount),
					fmt.Sprintf(`<g cat="B" n="%d"/>`, bCount),
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Compile ONE stylesheet, shared read-only across every goroutine.
			ss := compileStylesheetString(t, tc.stylesheet)

			// Pre-parse each goroutine's source once. Each goroutine owns a
			// distinct *helium.Document, so any failure is attributable to the
			// shared *Stylesheet rather than a shared source tree.
			sources := make([]*helium.Document, goroutines)
			for i := range sources {
				doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.source(i)))
				require.NoError(t, err)
				sources[i] = doc
			}

			// Each goroutine records its own failure into its own slot: no
			// shared write target, so the test harness itself introduces no race.
			failures := make([]string, goroutines)
			var wg sync.WaitGroup
			for i := range goroutines {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					want := tc.expect(i)
					for iter := range iterations {
						out, err := xslt3.TransformString(t.Context(), sources[i], ss)
						if err != nil {
							failures[i] = fmt.Sprintf("goroutine %d iter %d: transform error: %v", i, iter, err)
							return
						}
						for _, w := range want {
							if !strings.Contains(out, w) {
								failures[i] = fmt.Sprintf("goroutine %d iter %d: output missing %q\ngot: %s", i, iter, w, out)
								return
							}
						}
					}
				}(i)
			}
			wg.Wait()

			for i, f := range failures {
				require.Empty(t, f, "goroutine %d reported a failure", i)
			}
		})
	}
}
