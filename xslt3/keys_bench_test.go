package xslt3_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// buildNamespaceHeavySource builds a document whose root declares nsCount
// namespace prefixes and whose elemCount children each declare one more. Every
// element therefore carries nsCount+2 in-scope namespace nodes (the root's
// prefixes, its own, and the implicit xml prefix), which is what makes
// collectInScopeNSNodes expensive during a key table build. The document is
// fixed and deterministic: same bytes for the same arguments, no randomness and
// no external fixture file.
func buildNamespaceHeavySource(nsCount, elemCount int) []byte {
	var buf strings.Builder
	buf.WriteString(`<?xml version="1.0"?>`)
	buf.WriteString(`<root`)
	for i := range nsCount {
		buf.WriteString(` xmlns:r`)
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString(`="urn:bench:root:`)
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString(`"`)
	}
	buf.WriteString(`>`)
	for i := range elemCount {
		id := strconv.Itoa(i)
		buf.WriteString(`<item xmlns:e`)
		buf.WriteString(id)
		buf.WriteString(`="urn:bench:elem:`)
		buf.WriteString(id)
		buf.WriteString(`" id="i`)
		buf.WriteString(id)
		buf.WriteString(`"><v>`)
		buf.WriteString(id)
		buf.WriteString(`</v></item>`)
	}
	buf.WriteString(`</root>`)
	return []byte(buf.String())
}

// keyElementOnlyStylesheet indexes elements only. Its match pattern is a plain
// name test on the child axis, so it can never select a namespace node and the
// key table build skips namespace-node enumeration entirely.
const keyElementOnlyStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:key name="k" match="item" use="name()"/>
  <xsl:template match="/">
    <out><xsl:value-of select="count(key('k','item'))"/></out>
  </xsl:template>
</xsl:stylesheet>`

// keyNamespaceNodeStylesheet indexes the same elements plus namespace nodes.
// The namespace-node() alternative forces the key table build to enumerate
// in-scope namespace nodes on every element, so this arm exercises the path the
// guard deliberately leaves alone.
const keyNamespaceNodeStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:output method="xml" omit-xml-declaration="yes"/>
  <xsl:key name="k" match="item|namespace-node()" use="name()"/>
  <xsl:template match="/">
    <out><xsl:value-of select="count(key('k','item'))"/></out>
  </xsl:template>
</xsl:stylesheet>`

// BenchmarkBuildKeyTableNamespaceHeavy measures a transform whose only real
// work is one key table build over a namespace-heavy document, in both
// directions of the namespace-node guard in buildKeyTable.
//
// The "element-only-key" arm's key pattern cannot select a namespace node, so
// the build skips in-scope namespace-node enumeration. The "namespace-node-key"
// arm's pattern can, so the build still enumerates; it is here to show that the
// non-skipping path costs the same as before the guard was added.
//
// Run it with:
//
//	go test ./xslt3 -run '^$' -bench BenchmarkBuildKeyTableNamespaceHeavy -benchmem -count=6
//
// The baseline for the saving is obtained by running this same file against an
// origin/main checkout (extract one with `git archive origin/main`, copy this
// file in, and run the identical command there). Nothing in the tree stands in
// for the pre-guard implementation.
//
// Measured that way on linux/amd64 (Ryzen 9 7900X3D), the guard cuts the
// element-only arm from 17559135 B/op and 128385 allocs/op to 7700962 B/op and
// 52370 allocs/op, a 56% drop in bytes and a 59% drop in allocation count. The
// namespace-node arm stays at 136.47 MB/op and 972994 allocs/op on both sides.
func BenchmarkBuildKeyTableNamespaceHeavy(b *testing.B) {
	srcBytes := buildNamespaceHeavySource(8, 2000)

	source, err := helium.NewParser().Parse(b.Context(), srcBytes)
	require.NoError(b, err)

	elementOnlySS := compileBenchStylesheet(b, keyElementOnlyStylesheet)
	namespaceNodeSS := compileBenchStylesheet(b, keyNamespaceNodeStylesheet)

	b.Run("element-only-key", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := xslt3.Transform(b.Context(), source, elementOnlySS)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("namespace-node-key", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := xslt3.Transform(b.Context(), source, namespaceNodeSS)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
