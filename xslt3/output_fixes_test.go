package xslt3_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// mutatedMarker is a sentinel written into derived/snapshot OutputDef fields to
// prove that mutating them never reaches compiled or shared stylesheet state.
const mutatedMarker = "MUTATED"

// A-003: serializeResult must not mutate the caller-supplied OutputDef during
// html/xhtml auto-detection. Reusing one OutputDef{Method:"xml"} across an
// <html> doc and a non-html doc must not turn the second into HTML output.
func TestSerializeResultDoesNotMutateOutputDef(t *testing.T) {
	parse := func(src string) *helium.Document {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		return doc
	}

	outDef := &xslt3.OutputDef{Method: outMethodXML}

	htmlDoc := parse(`<html><body><br/></body></html>`)
	var htmlBuf strings.Builder
	require.NoError(t, xslt3.SerializeResult(&htmlBuf, htmlDoc, outDef))

	// outDef must still be xml-method after auto-detecting HTML.
	require.Equal(t, outMethodXML, outDef.Method, "outDef.Method must not be mutated")
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
	*d1.ItemSeparator = mutatedMarker
	*d1.BuildTree = false

	d3 := ss.DefaultOutputDef()
	require.Equal(t, outMethodXML, d3.Method, "mutating a returned def must not corrupt internal state")
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
	*r1.ItemSeparator = mutatedMarker

	r2 := inv.ResolvedOutputDef()
	require.NotNil(t, r2)
	require.Equal(t, outMethodXML, r2.Method, "ResolvedOutputDef must return an independent snapshot")
	require.NotNil(t, r2.ItemSeparator)
	require.Equal(t, "|", *r2.ItemSeparator, "mutating snapshot *ItemSeparator must not affect a later read")
}

// W01: a ResultDocumentHandler must receive an OutputDef whose pointer/slice/map
// fields are independent of the compiled stylesheet. Mutating those fields from
// the handler must not corrupt the compiled named format shared across runs.
type captureResultDocHandler struct {
	outDef *xslt3.OutputDef
}

func (h *captureResultDocHandler) HandleResultDocument(_ string, _ *helium.Document, outDef *xslt3.OutputDef) error {
	h.outDef = outDef
	return nil
}

func TestResultDocumentHandlerOutputDefIsIsolated(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output name="fmt" method="xml" item-separator="|" build-tree="yes"
              suppress-indentation="a b"/>
  <xsl:template match="/">
    <xsl:result-document href="secondary.xml" format="fmt"><secondary/></xsl:result-document>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

	run := func() *xslt3.OutputDef {
		h := &captureResultDocHandler{}
		_, err := ss.Transform(parseTransformSource(t)).ResultDocumentHandler(h).Do(t.Context())
		require.NoError(t, err)
		require.NotNil(t, h.outDef, "handler must receive an OutputDef")
		return h.outDef
	}

	first := run()
	require.NotNil(t, first.ItemSeparator)
	require.Equal(t, "|", *first.ItemSeparator)
	require.NotNil(t, first.BuildTree)
	require.True(t, *first.BuildTree)
	require.Equal(t, []string{"a", "b"}, first.SuppressIndentation)

	// Mutate every pointer/slice/map field on the delivered def.
	*first.ItemSeparator = mutatedMarker
	*first.BuildTree = false
	first.SuppressIndentation[0] = mutatedMarker
	first.SuppressIndentation = append(first.SuppressIndentation, "MUTATED2")
	if first.ResolvedCharMap == nil {
		first.ResolvedCharMap = map[rune]string{}
	}
	first.ResolvedCharMap['x'] = mutatedMarker

	// A second run must observe the original compiled values, proving the first
	// delivered def did not alias compiled/shared state.
	second := run()
	require.NotNil(t, second.ItemSeparator)
	require.Equal(t, "|", *second.ItemSeparator, "compiled item-separator must be unaffected")
	require.NotNil(t, second.BuildTree)
	require.True(t, *second.BuildTree, "compiled build-tree must be unaffected")
	require.Equal(t, []string{"a", "b"}, second.SuppressIndentation, "compiled suppress-indentation must be unaffected")
	require.Empty(t, second.ResolvedCharMap, "compiled char map must be unaffected")
}

// W01: a primary xsl:result-document delivers its effective output def via
// ResolvedOutputDef. Mutating that snapshot's pointer/slice/map fields must not
// corrupt the compiled stylesheet across runs.
func TestPrimaryResultDocumentResolvedOutputDefIsIsolated(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" item-separator="|" build-tree="yes"/>
  <xsl:template match="/">
    <xsl:result-document item-separator=";"><out/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

	run := func() *xslt3.OutputDef {
		inv := ss.Transform(parseTransformSource(t))
		_, err := inv.Do(t.Context())
		require.NoError(t, err)
		r := inv.ResolvedOutputDef()
		require.NotNil(t, r)
		return r
	}

	first := run()
	require.NotNil(t, first.ItemSeparator)
	require.Equal(t, ";", *first.ItemSeparator, "result-document override must apply")

	*first.ItemSeparator = mutatedMarker
	if first.BuildTree != nil {
		*first.BuildTree = false
	}

	second := run()
	require.NotNil(t, second.ItemSeparator)
	require.Equal(t, ";", *second.ItemSeparator, "compiled/override state must be unaffected across runs")
}

// A primary xsl:result-document with an explicit false boolean serialization
// AVT must override an inherited true from xsl:output. Before the fix the merge
// OR-ed the override with the inherited value, so an explicit false could never
// turn an inherited true back off.
func TestPrimaryResultDocBooleanFalseOverridesInheritedTrue(t *testing.T) {
	resolve := func(t *testing.T, output, resultDocAttrs string) *xslt3.OutputDef {
		t.Helper()
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  `+output+`
  <xsl:template match="/">
    <xsl:result-document `+resultDocAttrs+`><out/></xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)
		inv := ss.Transform(parseTransformSource(t))
		_, err := inv.Do(t.Context())
		require.NoError(t, err)
		r := inv.ResolvedOutputDef()
		require.NotNil(t, r)
		return r
	}

	t.Run("indent false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" indent="yes"/>`, `indent="{false()}"`)
		require.False(t, r.Indent, "explicit indent=false must override inherited indent=yes")
	})

	t.Run("indent inherited yes stays on when not overridden", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" indent="yes"/>`, `method="xml"`)
		require.True(t, r.Indent, "inherited indent=yes must remain on when not overridden")
	})

	t.Run("omit-xml-declaration false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" omit-xml-declaration="yes"/>`, `omit-xml-declaration="{false()}"`)
		require.False(t, r.OmitDeclaration, "explicit omit-xml-declaration=false must override inherited yes")
	})

	t.Run("byte-order-mark false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" byte-order-mark="yes"/>`, `byte-order-mark="{false()}"`)
		require.False(t, r.ByteOrderMark, "explicit byte-order-mark=false must override inherited yes")
	})

	t.Run("escape-uri-attributes false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="html" escape-uri-attributes="yes"/>`, `escape-uri-attributes="{false()}"`)
		require.NotNil(t, r.EscapeURIAttributes)
		require.False(t, *r.EscapeURIAttributes, "explicit escape-uri-attributes=false must override inherited yes")
	})

	t.Run("include-content-type false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="html" include-content-type="yes"/>`, `include-content-type="{false()}"`)
		require.NotNil(t, r.IncludeContentType)
		require.False(t, *r.IncludeContentType, "explicit include-content-type=false must override inherited yes")
	})

	t.Run("undeclare-prefixes false overrides inherited yes", func(t *testing.T) {
		r := resolve(t, `<xsl:output method="xml" version="1.1" undeclare-prefixes="yes"/>`, `undeclare-prefixes="{false()}"`)
		require.False(t, r.UndeclarePrefixes, "explicit undeclare-prefixes=false must override inherited yes")
	})
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
