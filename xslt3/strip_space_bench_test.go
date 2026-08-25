package xslt3_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

// benchLeaves is the leaf-<item> count held constant across depths so growth
// in the benchmarks below tracks depth alone, not document size.
const benchLeaves = 500

// genStripBenchDoc builds a pretty-printed document nested depth levels deep
// whose innermost element holds benchLeaves <item> children. Every element is
// separated by indentation whitespace, so the number of whitespace-only text
// nodes is ~= depth + 2*benchLeaves, and the deepest ones sit depth levels
// from the root — the shape that makes an O(depth) ancestor walk per
// whitespace node show up as O(depth) total.
func genStripBenchDoc(depth int) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?>` + "\n")
	for i := range depth {
		b.WriteString(" ")
		fmt.Fprintf(&b, "<l%d>\n", i)
	}
	for i := range benchLeaves {
		b.WriteString(" ")
		fmt.Fprintf(&b, "<item id=\"i%d\" class=\"c\" xml:lang=\"en\">v%d</item>\n", i, i)
	}
	for i := depth - 1; i >= 0; i-- {
		b.WriteString(" ")
		fmt.Fprintf(&b, "</l%d>\n", i)
	}
	return []byte(b.String())
}

const benchPlainStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates/></out></xsl:template>
</xsl:stylesheet>`

const benchStripStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:strip-space elements="*"/>
  <xsl:template match="/"><out><xsl:apply-templates/></out></xsl:template>
</xsl:stylesheet>`

func runStripBench(b *testing.B, ssSrc string, depth int) {
	b.Helper()
	ctx := context.Background()
	p := helium.NewParser()
	ssDoc, err := p.Parse(ctx, []byte(ssSrc))
	if err != nil {
		b.Fatal(err)
	}
	ss, err := xslt3.CompileStylesheet(ctx, ssDoc)
	if err != nil {
		b.Fatal(err)
	}
	src := genStripBenchDoc(depth)
	// A transform never mutates its source (the strip pass works on a private
	// copy), so one parse is reused across iterations and the measurement
	// covers the transform alone.
	doc, err := p.Parse(ctx, src)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for range b.N {
		if _, err := xslt3.TransformString(ctx, doc, ss); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStripSpacePlainDepth exercises the default (no xsl:strip-space)
// configuration: the built-in template rules call shouldStripWhitespace for
// every whitespace-only text node, and the schema/rule/DTD verdict is always
// false, so this is the case the reorder (evaluating inheritedXMLSpace last)
// targets directly.
func BenchmarkStripSpacePlainDepth(b *testing.B) {
	for _, d := range []int{2, 8, 32, 128} {
		b.Run(fmt.Sprintf("depth=%d", d), func(b *testing.B) { runStripBench(b, benchPlainStylesheet, d) })
	}
}

// BenchmarkStripSpaceStripDepth is the control: xsl:strip-space elements="*"
// routes the source through copyAndStrip (strip_space_copy.go), a separate,
// already-threaded code path this change does not touch. Its curve should stay
// flat across depth.
func BenchmarkStripSpaceStripDepth(b *testing.B) {
	for _, d := range []int{2, 8, 32, 128} {
		b.Run(fmt.Sprintf("depth=%d", d), func(b *testing.B) { runStripBench(b, benchStripStylesheet, d) })
	}
}

// stripBenchResolver serves one in-memory document for fn:doc().
type stripBenchResolver struct{ data []byte }

func (r stripBenchResolver) ResolveURI(string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(r.data)), nil
}

const benchDocStripStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:strip-space elements="*"/>
  <xsl:template match="/"><out><xsl:value-of select="count(doc('mem:x')//item)"/></out></xsl:template>
</xsl:stylesheet>`

// BenchmarkStripSpaceDocStrip exercises stripWhitespaceFromNodeInto (the strip
// pre-pass fn:doc() runs over a loaded document), the call site the original
// finding missed.
func BenchmarkStripSpaceDocStrip(b *testing.B) {
	ctx := context.Background()
	p := helium.NewParser()
	ssDoc, err := p.Parse(ctx, []byte(benchDocStripStylesheet))
	if err != nil {
		b.Fatal(err)
	}
	ss, err := xslt3.CompileStylesheet(ctx, ssDoc)
	if err != nil {
		b.Fatal(err)
	}
	srcDoc, err := p.Parse(ctx, []byte(`<a/>`))
	if err != nil {
		b.Fatal(err)
	}
	for _, d := range []int{2, 8, 32, 128} {
		b.Run(fmt.Sprintf("depth=%d", d), func(b *testing.B) {
			ext := genStripBenchDoc(d)
			b.SetBytes(int64(len(ext)))
			b.ResetTimer()
			for range b.N {
				if _, err := ss.Transform(srcDoc).URIResolver(stripBenchResolver{data: ext}).Do(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// genStripBenchPreserveDoc is genStripBenchDoc with xml:space="preserve" on
// the root element, so strip-space rules match but every whitespace node is
// retained.
func genStripBenchPreserveDoc(depth int) []byte {
	src := string(genStripBenchDoc(depth))
	return []byte(strings.Replace(src, "<l0>", `<l0 xml:space="preserve">`, 1))
}

// BenchmarkStripSpacePreserveDepth: strip-space rules match every element,
// but xml:space="preserve" retains all whitespace, so applyTemplates
// re-checks each retained whitespace node at execution time. This is the case
// the optional per-transform xml:space memo (execContext.xmlSpacePreserveMemo)
// targets.
func BenchmarkStripSpacePreserveDepth(b *testing.B) {
	ctx := context.Background()
	p := helium.NewParser()
	ssDoc, err := p.Parse(ctx, []byte(benchStripStylesheet))
	if err != nil {
		b.Fatal(err)
	}
	ss, err := xslt3.CompileStylesheet(ctx, ssDoc)
	if err != nil {
		b.Fatal(err)
	}
	for _, d := range []int{2, 8, 32, 128} {
		b.Run(fmt.Sprintf("depth=%d", d), func(b *testing.B) {
			src := genStripBenchPreserveDoc(d)
			doc, err := p.Parse(ctx, src)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(src)))
			b.ResetTimer()
			for range b.N {
				if _, err := xslt3.TransformString(ctx, doc, ss); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
