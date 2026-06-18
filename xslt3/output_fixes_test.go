package xslt3_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// A-003: serializeResult must not mutate the caller-supplied OutputDef during
// html/xhtml auto-detection. Reusing one OutputDef{Method:"xml"} across an
// <html> doc and a non-html doc must not turn the second into HTML output.
func TestSerializeResultDoesNotMutateOutputDef(t *testing.T) {
	parse := func(src string) *helium.Document {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		return doc
	}

	outDef := &xslt3.OutputDef{Method: "xml"}

	htmlDoc := parse(`<html><body><br/></body></html>`)
	var htmlBuf strings.Builder
	require.NoError(t, xslt3.SerializeResult(&htmlBuf, htmlDoc, outDef))

	// outDef must still be xml-method after auto-detecting HTML.
	require.Equal(t, "xml", outDef.Method, "outDef.Method must not be mutated")
	require.False(t, outDef.OmitDeclaration, "outDef.OmitDeclaration must not be mutated")

	xmlDoc := parse(`<root><br/></root>`)
	var xmlBuf strings.Builder
	require.NoError(t, xslt3.SerializeResult(&xmlBuf, xmlDoc, outDef))

	// The second (non-html) doc must serialize as XML, not HTML.
	out := xmlBuf.String()
	require.Contains(t, out, "<br/>", "second doc must be XML-serialized with self-closing br")
	require.NotContains(t, out, "<br>", "second doc must not be HTML-serialized")
}

// A-002 / A-004: DefaultOutputDef must return a deep clone, not the internal
// pointer. Mutating the clone — including its pointer fields — must never reach
// through to the stylesheet's internal state.
func TestDefaultOutputDefReturnsClone(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" item-separator="|" build-tree="yes"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	d1 := ss.DefaultOutputDef()
	require.NotNil(t, d1)
	d2 := ss.DefaultOutputDef()
	require.NotNil(t, d2)
	require.NotSame(t, d1, d2, "DefaultOutputDef must return a fresh clone each call")

	// The pointer fields must themselves be fresh allocations, not aliases.
	require.NotNil(t, d1.ItemSeparator)
	require.NotNil(t, d2.ItemSeparator)
	require.NotSame(t, d1.ItemSeparator, d2.ItemSeparator, "ItemSeparator pointer must be deep-cloned")
	require.NotNil(t, d1.BuildTree)
	require.NotNil(t, d2.BuildTree)
	require.NotSame(t, d1.BuildTree, d2.BuildTree, "BuildTree pointer must be deep-cloned")

	// Mutating the returned def — scalar and pointee — must not affect internal state.
	d1.Method = "html"
	*d1.ItemSeparator = "MUTATED"
	*d1.BuildTree = false

	d3 := ss.DefaultOutputDef()
	require.Equal(t, "xml", d3.Method, "mutating a returned def must not corrupt internal state")
	require.NotNil(t, d3.ItemSeparator)
	require.Equal(t, "|", *d3.ItemSeparator, "mutating clone *ItemSeparator must not corrupt internal state")
	require.NotNil(t, d3.BuildTree)
	require.True(t, *d3.BuildTree, "mutating clone *BuildTree must not corrupt internal state")
}

// A-006: Do/WriteTo must not mutate the shared invocationConfig; ResolvedOutputDef
// must return an independent deep-cloned snapshot.
func TestResolvedOutputDefIsSnapshot(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" item-separator="|"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	_, err := inv.Do(t.Context())
	require.NoError(t, err)

	r1 := inv.ResolvedOutputDef()
	require.NotNil(t, r1)
	require.NotNil(t, r1.ItemSeparator)

	// Mutating the snapshot — scalar and pointee — must not affect a later read.
	r1.Method = "html"
	*r1.ItemSeparator = "MUTATED"

	r2 := inv.ResolvedOutputDef()
	require.NotNil(t, r2)
	require.Equal(t, "xml", r2.Method, "ResolvedOutputDef must return an independent snapshot")
	require.NotNil(t, r2.ItemSeparator)
	require.Equal(t, "|", *r2.ItemSeparator, "mutating snapshot *ItemSeparator must not affect a later read")
}

// A-006 race: concurrent Serialize/ResolvedOutputDef on the SAME Invocation
// value must be safe. Run under -race to catch data races on the shared config.
func TestConcurrentSerializeAndResolvedOutputDef(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" item-separator="|"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	ctx := t.Context()

	const n = 16
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for range n {
		go func() {
			defer wg.Done()
			_, err := inv.Serialize(ctx)
			require.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			// Reading the resolved def concurrently with Serialize must not race.
			_ = inv.ResolvedOutputDef()
		}()
	}
	wg.Wait()

	out, err := inv.Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "<out/>")
}
