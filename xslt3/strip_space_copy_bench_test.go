package xslt3_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// buildLargeStripSpaceSource generates an XML document with many elements and
// abundant whitespace-only text nodes between them (indentation/newlines), plus
// a couple of namespace declarations so the namespace-handling path is exercised.
// The shape is a few thousand elements so the per-node cost of the source copy is
// visible in the benchmark.
func buildLargeStripSpaceSource(sections, itemsPerSection int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?>` + "\n")
	b.WriteString(`<catalog xmlns="urn:bench:cat" xmlns:m="urn:bench:meta">` + "\n")
	for s := range sections {
		b.WriteString("  <section id=\"")
		b.WriteString(strconv.Itoa(s))
		b.WriteString("\">\n")
		for i := range itemsPerSection {
			b.WriteString("    <item m:rank=\"")
			b.WriteString(strconv.Itoa(i))
			b.WriteString("\">\n")
			b.WriteString("      <name>Item ")
			b.WriteString(strconv.Itoa(i))
			b.WriteString("</name>\n")
			b.WriteString("      <m:note>note</m:note>\n")
			b.WriteString("    </item>\n")
		}
		b.WriteString("  </section>\n")
	}
	b.WriteString("</catalog>\n")
	return []byte(b.String())
}

const stripBenchStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:c="urn:bench:cat" version="3.0">
  <xsl:strip-space elements="*"/>
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

const noStripBenchStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:c="urn:bench:cat" version="3.0">
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
</xsl:stylesheet>`

func compileBenchStylesheet(b *testing.B, src string) *xslt3.Stylesheet {
	b.Helper()
	doc, err := helium.NewParser().Parse(b.Context(), []byte(src))
	require.NoError(b, err)
	ss, err := xslt3.NewCompiler().Compile(b.Context(), doc)
	require.NoError(b, err)
	return ss
}

// BenchmarkStripSpaceTransform measures the strip-space transform path (which
// triggers the single-pass source copy) against an identical transform with no
// strip-space rules (no copy at all), so the copy overhead is directly visible.
func BenchmarkStripSpaceTransform(b *testing.B) {
	srcBytes := buildLargeStripSpaceSource(40, 30) // ~5k+ elements with whitespace

	source, err := helium.NewParser().Parse(b.Context(), srcBytes)
	require.NoError(b, err)

	stripSS := compileBenchStylesheet(b, stripBenchStylesheet)
	noStripSS := compileBenchStylesheet(b, noStripBenchStylesheet)

	b.Run("strip-space", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := xslt3.Transform(b.Context(), source, stripSS)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("no-strip-space", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := xslt3.Transform(b.Context(), source, noStripSS)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
