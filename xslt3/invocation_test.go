package xslt3_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestInvocationGlobalParameters(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="a" select="'default-a'"/>
  <xsl:param name="b" select="'default-b'"/>
  <xsl:template match="/">
    <out><xsl:value-of select="concat($a, '|', $b)"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	p := xslt3.NewParameters()
	p.SetString("a", "alpha")
	p.SetString("b", "bravo")

	result, err := ss.Transform(parseTransformSource(t)).
		GlobalParameters(p).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "alpha|bravo")

	// Mutating p after GlobalParameters should not affect the invocation.
	p.SetString("a", "mutated")
	result2, err := ss.Transform(parseTransformSource(t)).
		GlobalParameters(p).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result2, "mutated|bravo")
}

func TestInvocationTunnelParameters(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:apply-templates select="root"/>
  </xsl:template>
  <xsl:template match="root">
    <xsl:call-template name="inner"/>
  </xsl:template>
  <xsl:template name="inner">
    <xsl:param name="secret" tunnel="yes" select="'none'"/>
    <out><xsl:value-of select="$secret"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	p := xslt3.NewParameters()
	p.SetString("secret", "tunnel-value")

	result, err := ss.Transform(source).
		TunnelParameters(p).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "tunnel-value")
}

func TestInvocationCollectionResolver(t *testing.T) {
	// CollectionResolver is a fluent setter — verify clone-on-write.
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv1 := ss.Transform(parseTransformSource(t))
	inv2 := inv1.CollectionResolver(nil)

	// Both should execute without error.
	_, err := inv1.Do(t.Context())
	require.NoError(t, err)

	_, err = inv2.Do(t.Context())
	require.NoError(t, err)
}

func TestInvocationBaseOutputURI(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="current-output-uri()"/></out>
  </xsl:template>
</xsl:stylesheet>`)

	result, err := ss.Transform(parseTransformSource(t)).
		BaseOutputURI("file:///output/result.xml").
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "file:///output/result.xml")
}

func TestInvocationOnMultipleMatch(t *testing.T) {
	// Clone-on-write: setting OnMultipleMatch on one invocation
	// doesn't affect the original.
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv1 := ss.Transform(parseTransformSource(t))
	inv2 := inv1.OnMultipleMatch(xslt3.OnMultipleMatchFail)

	_, err := inv1.Do(t.Context())
	require.NoError(t, err)

	_, err = inv2.Do(t.Context())
	require.NoError(t, err)
}

func TestInvocationTraceWriter(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:sequence select="trace('hello', 'label')"/>
  </xsl:template>
</xsl:stylesheet>`)

	var buf bytes.Buffer
	_, err := ss.Transform(parseTransformSource(t)).
		TraceWriter(&buf).
		Do(t.Context())
	require.NoError(t, err)
	require.Contains(t, buf.String(), "label")
}

func TestInvocationWriteTo(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>hello</out></xsl:template>
</xsl:stylesheet>`)

	var buf bytes.Buffer
	err := ss.Transform(parseTransformSource(t)).WriteTo(t.Context(), &buf)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<out>hello</out>")
}

func TestInvocationSourceSchemas(t *testing.T) {
	// SourceSchemas is a fluent setter — verify clone-on-write.
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv1 := ss.Transform(parseTransformSource(t))
	inv2 := inv1.SourceSchemas() // empty slice

	_, err := inv1.Do(t.Context())
	require.NoError(t, err)

	_, err = inv2.Do(t.Context())
	require.NoError(t, err)
}

func TestInvocationTransformSelectionRejected(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	// Selection is invalid for Transform.
	_, err := ss.Transform(nil).Selection(xpath3.SingleString("x")).Do(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "Selection is not valid for Transform")
}

func TestInvocationCallTemplateValidation(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="t"><out/></xsl:template>
</xsl:stylesheet>`)

	// Mode is invalid for CallTemplate.
	_, err := ss.CallTemplate("t").Mode("m").Do(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "Mode is not valid for CallTemplate")

	// Selection is invalid for CallTemplate.
	_, err = ss.CallTemplate("t").Selection(xpath3.SingleString("x")).Do(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "Selection is not valid for CallTemplate")
}

func TestInvocationCallFunctionValidation(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:f="http://example.com/f">
  <xsl:function name="f:id"><xsl:param name="x"/><xsl:sequence select="$x"/></xsl:function>
</xsl:stylesheet>`)

	// Mode is invalid for CallFunction.
	_, err := ss.CallFunction("{http://example.com/f}id", xpath3.SingleString("a")).
		Mode("m").Do(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "Mode is not valid for CallFunction")

	// Selection is invalid for CallFunction.
	_, err = ss.CallFunction("{http://example.com/f}id", xpath3.SingleString("a")).
		Selection(xpath3.SingleString("x")).Do(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "Selection is not valid for CallFunction")
}

func TestOnMultipleMatchModeString(t *testing.T) {
	require.Equal(t, "use-last", xslt3.OnMultipleMatchUseLast.String())
	require.Equal(t, "fail", xslt3.OnMultipleMatchFail.String())
	require.Equal(t, "", xslt3.OnMultipleMatchDefault.String())
}

func TestInvocationSerialize(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>text</out></xsl:template>
</xsl:stylesheet>`)

	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>text</out>")
}

type discardWriter struct {
	written int
}

func (w *discardWriter) Write(p []byte) (int, error) {
	w.written += len(p)
	return len(p), nil
}

func TestInvocationWriteToErrorPropagation(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>long text content to trigger write</out></xsl:template>
</xsl:stylesheet>`)

	w := &discardWriter{}
	err := ss.Transform(parseTransformSource(t)).WriteTo(t.Context(), w)
	require.NoError(t, err)
	require.True(t, w.written > 0)
}

func TestResolvedOutputDefAfterSerialize(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" indent="yes"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	require.Nil(t, inv.ResolvedOutputDef(), "should be nil before execution")

	_, err := inv.Serialize(t.Context())
	require.NoError(t, err)
	require.NotNil(t, inv.ResolvedOutputDef(), "should be populated after Serialize")
}

func TestResolvedOutputDefAfterWriteTo(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" indent="yes"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

	inv := ss.Transform(parseTransformSource(t))
	require.Nil(t, inv.ResolvedOutputDef(), "should be nil before execution")

	var buf bytes.Buffer
	err := inv.WriteTo(t.Context(), &buf)
	require.NoError(t, err)
	require.NotNil(t, inv.ResolvedOutputDef(), "should be populated after WriteTo")
}

func TestTransformStringConvenience(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>conv</out></xsl:template>
</xsl:stylesheet>`)

	result, err := xslt3.TransformString(t.Context(), parseTransformSource(t), ss)
	require.NoError(t, err)
	require.Contains(t, result, "<out>conv</out>")
}

func TestTransformToWriterConvenience(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>conv</out></xsl:template>
</xsl:stylesheet>`)

	var buf bytes.Buffer
	err := xslt3.TransformToWriter(t.Context(), parseTransformSource(t), ss, &buf)
	require.NoError(t, err)
	require.True(t, strings.Contains(buf.String(), "<out>conv</out>"), buf.String())
}
