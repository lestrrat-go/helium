package xslt3_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
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

// TestStylesheetConcurrentSharedSourceNonSchemaAware proves the narrower
// shared-source guarantee: a NON-schema-aware transform treats the caller's
// source document as read-only (whitespace stripping, when it happens, runs on
// a private copy), so a SINGLE source document may be handed to many concurrent
// transforms of the same compiled stylesheet without external synchronization.
// Every goroutine reads the same shared *helium.Document and must produce the
// same deterministic output; under -race, any write to the shared source (or to
// the compiled stylesheet) would be reported.
//
// The stylesheet includes xsl:strip-space so the copy-and-strip path (which
// reads the shared tree) is exercised, and it reads element content/counts so
// the shared source tree is traversed concurrently.
func TestStylesheetConcurrentSharedSourceNonSchemaAware(t *testing.T) {
	const (
		goroutines = 50
		iterations = 25
	)

	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:strip-space elements="*"/>
  <xsl:template match="/root">
    <out n="{count(item)}"><xsl:value-of select="item[1]/@id"/>|<xsl:value-of select="item[last()]/@id"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	// ONE shared source document, read concurrently by every goroutine.
	sharedSrc, err := helium.NewParser().Parse(t.Context(), []byte(
		`<root>  <item id="first"/>  <item id="mid"/>  <item id="last"/>  </root>`))
	require.NoError(t, err)

	const wantFrag = `<out n="3">first|last</out>`

	failures := make([]string, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for iter := range iterations {
				out, err := xslt3.TransformString(t.Context(), sharedSrc, ss)
				if err != nil {
					failures[i] = fmt.Sprintf("goroutine %d iter %d: transform error: %v", i, iter, err)
					return
				}
				if !strings.Contains(out, wantFrag) {
					failures[i] = fmt.Sprintf("goroutine %d iter %d: output missing %q\ngot: %s", i, iter, wantFrag, out)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	for i, f := range failures {
		require.Empty(t, f, "goroutine %d reported a failure", i)
	}
}

// TestStylesheetConcurrentSchemaAwareDistinctSources proves that a schema-aware
// / source-validating transform is safe to run concurrently WHEN EACH GOROUTINE
// USES ITS OWN SOURCE DOCUMENT. Such a transform validates-and-mutates its input
// tree in place (source-schema validation inserts default/fixed attributes), so
// the input must not be shared across concurrent transforms — but with distinct
// per-goroutine sources there is no shared write target, and the compiled
// *Stylesheet and *xsd.Schema remain read-only.
//
// This is the safe pattern the documentation prescribes for schema-aware
// transforms; the unsafe shared-source schema-aware case is deliberately NOT
// exercised here (the documentation warns against it).
func TestStylesheetConcurrentSchemaAwareDistinctSources(t *testing.T) {
	const (
		goroutines = 50
		iterations = 25
	)

	// A schema that declares a default attribute; validation inserts it into the
	// (per-goroutine) source tree, which the stylesheet then reads back.
	schema := compileSchemaString(t, `
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="tag" type="xs:string"/>
            <xs:attribute name="def" type="xs:string" default="DEF"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)

	// The stylesheet reads both the instance tag and the schema-defaulted attr,
	// so the default-attribute insertion (the in-place mutation) is observed.
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/root">
    <out><tag><xsl:value-of select="item/@tag"/></tag><def><xsl:value-of select="item/@def"/></def></out>
  </xsl:template>
</xsl:stylesheet>`)

	failures := make([]string, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			wantTag := fmt.Sprintf("<tag>T%d</tag>", i)
			const wantDef = "<def>DEF</def>"
			for iter := range iterations {
				// Each goroutine parses a FRESH source document per iteration:
				// schema validation mutates the tree in place, so a source may
				// not be reused across transforms even within one goroutine.
				src, err := helium.NewParser().Parse(t.Context(),
					fmt.Appendf(nil, `<root><item tag="T%d"/></root>`, i))
				if err != nil {
					failures[i] = fmt.Sprintf("goroutine %d iter %d: parse error: %v", i, iter, err)
					return
				}
				out, err := ss.Transform(src).SourceSchemas(schema).Serialize(t.Context())
				if err != nil {
					failures[i] = fmt.Sprintf("goroutine %d iter %d: transform error: %v", i, iter, err)
					return
				}
				if !strings.Contains(out, wantTag) {
					failures[i] = fmt.Sprintf("goroutine %d iter %d: output missing %q\ngot: %s", i, iter, wantTag, out)
					return
				}
				if !strings.Contains(out, wantDef) {
					failures[i] = fmt.Sprintf("goroutine %d iter %d: output missing %q (schema default not inserted)\ngot: %s", i, iter, wantDef, out)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	for i, f := range failures {
		require.Empty(t, f, "goroutine %d reported a failure", i)
	}
}

func compileSchemaString(t *testing.T, src string) *xsd.Schema {
	t.Helper()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
	return schema
}
