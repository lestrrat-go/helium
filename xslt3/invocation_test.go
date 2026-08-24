package xslt3_test

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestInvocation(t *testing.T) {
	t.Run("global parameters", func(t *testing.T) {
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
	})

	t.Run("tunnel parameters", func(t *testing.T) {
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
	})

	t.Run("collection resolver", func(t *testing.T) {
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
	})

	t.Run("base output URI", func(t *testing.T) {
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
	})

	t.Run("on multiple match", func(t *testing.T) {
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
	})

	t.Run("trace writer", func(t *testing.T) {
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
	})

	t.Run("WriteTo", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>hello</out></xsl:template>
</xsl:stylesheet>`)

		var buf bytes.Buffer
		err := ss.Transform(parseTransformSource(t)).WriteTo(t.Context(), &buf)
		require.NoError(t, err)
		require.Contains(t, buf.String(), "<out>hello</out>")
	})

	t.Run("source schemas", func(t *testing.T) {
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
	})

	t.Run("transform selection rejected", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

		// Selection is invalid for Transform.
		_, err := ss.Transform(nil).Selection(xpath3.SingleString("x")).Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "Selection is not valid for Transform")
	})

	t.Run("call template validation", func(t *testing.T) {
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

		// InitialModeParameter is invalid for CallTemplate.
		_, err = ss.CallTemplate("t").SetInitialModeParameter("p", xpath3.SingleString("v")).Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "SetInitialModeParameter is not valid for CallTemplate")
	})

	t.Run("call function validation", func(t *testing.T) {
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

		// InitialModeParameter is invalid for CallFunction.
		_, err = ss.CallFunction("{http://example.com/f}id", xpath3.SingleString("a")).
			SetInitialModeParameter("p", xpath3.SingleString("v")).Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "SetInitialModeParameter is not valid for CallFunction")

		// TunnelParameters is invalid for CallFunction.
		_, err = ss.CallFunction("{http://example.com/f}id", xpath3.SingleString("a")).
			SetTunnelParameter("t", xpath3.SingleString("v")).Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "TunnelParameters is not valid for CallFunction")
	})

	t.Run("initial template param rejected for transform", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

		_, err := ss.Transform(parseTransformSource(t)).
			SetInitialTemplateParameter("p", xpath3.SingleString("v")).
			Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "SetInitialTemplateParameter is not valid for Transform")
	})

	t.Run("initial template param rejected for apply templates", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)

		_, err := ss.ApplyTemplates(parseTransformSource(t)).
			SetInitialTemplateParameter("p", xpath3.SingleString("v")).
			Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "SetInitialTemplateParameter is not valid for ApplyTemplates")
	})

	t.Run("initial template param rejected for call function", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:f="http://example.com/f">
  <xsl:function name="f:id"><xsl:param name="x"/><xsl:sequence select="$x"/></xsl:function>
</xsl:stylesheet>`)

		_, err := ss.CallFunction("{http://example.com/f}id", xpath3.SingleString("a")).
			SetInitialTemplateParameter("p", xpath3.SingleString("v")).
			Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "SetInitialTemplateParameter is not valid for CallFunction")
	})

	t.Run("initial mode param rejected for call template", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="t"><out/></xsl:template>
</xsl:stylesheet>`)

		_, err := ss.CallTemplate("t").
			SetInitialModeParameter("p", xpath3.SingleString("v")).
			Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "SetInitialModeParameter is not valid for CallTemplate")
	})

	t.Run("initial mode param rejected for call function", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:f="http://example.com/f">
  <xsl:function name="f:id"><xsl:param name="x"/><xsl:sequence select="$x"/></xsl:function>
</xsl:stylesheet>`)

		_, err := ss.CallFunction("{http://example.com/f}id", xpath3.SingleString("a")).
			SetInitialModeParameter("p", xpath3.SingleString("v")).
			Do(t.Context())
		require.Error(t, err)
		require.Contains(t, err.Error(), "SetInitialModeParameter is not valid for CallFunction")
	})

	t.Run("serialize", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>text</out></xsl:template>
</xsl:stylesheet>`)

		result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, result, "<out>text</out>")
	})

	t.Run("WriteTo error propagation", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>long text content to trigger write</out></xsl:template>
</xsl:stylesheet>`)

		w := &discardWriter{}
		err := ss.Transform(parseTransformSource(t)).WriteTo(t.Context(), w)
		require.NoError(t, err)
		require.True(t, w.written > 0)
	})

	t.Run("copy on write parameters", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="x" select="'default-x'"/>
  <xsl:param name="y" select="'default-y'"/>
  <xsl:template match="/">
    <out><xsl:value-of select="concat($x, '|', $y)"/></out>
  </xsl:template>
</xsl:stylesheet>`)

		source := parseTransformSource(t)
		base := ss.Transform(source).SetParameter("x", xpath3.SingleString("one"))
		derived := base.SetParameter("y", xpath3.SingleString("two"))

		baseResult, err := base.Serialize(t.Context())
		require.NoError(t, err)
		require.True(t, strings.Contains(baseResult, "<out>one|default-y</out>"), baseResult)

		derivedResult, err := derived.Serialize(t.Context())
		require.NoError(t, err)
		require.True(t, strings.Contains(derivedResult, "<out>one|two</out>"), derivedResult)
	})

	t.Run("OnMultipleMatch mode string", func(t *testing.T) {
		require.Equal(t, "use-last", xslt3.OnMultipleMatchUseLast.String())
		require.Equal(t, "fail", xslt3.OnMultipleMatchFail.String())
		require.Equal(t, "", xslt3.OnMultipleMatchDefault.String())
	})

	t.Run("call-template preserves parameters", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="x" select="'default-x'"/>
  <xsl:template match="/">
    <out>root</out>
  </xsl:template>
  <xsl:template name="t">
    <out><xsl:value-of select="$x"/></out>
  </xsl:template>
</xsl:stylesheet>`)

		source := parseTransformSource(t)
		result, err := ss.CallTemplate("t").
			SourceDocument(source).
			SetParameter("x", xpath3.SingleString("one")).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, result, "<out>one</out>")
	})
}

type discardWriter struct {
	written int
}

func (w *discardWriter) Write(p []byte) (int, error) {
	w.written += len(p)
	return len(p), nil
}

func TestResolvedOutputDef(t *testing.T) {
	t.Run("after serialize", func(t *testing.T) {
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
	})

	t.Run("after WriteTo", func(t *testing.T) {
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
	})
}

func TestTransformConvenience(t *testing.T) {
	t.Run("TransformString", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>conv</out></xsl:template>
</xsl:stylesheet>`)

		result, err := xslt3.TransformString(t.Context(), parseTransformSource(t), ss)
		require.NoError(t, err)
		require.Contains(t, result, "<out>conv</out>")
	})

	t.Run("TransformToWriter", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out>conv</out></xsl:template>
</xsl:stylesheet>`)

		var buf bytes.Buffer
		err := xslt3.TransformToWriter(t.Context(), parseTransformSource(t), ss, &buf)
		require.NoError(t, err)
		require.True(t, strings.Contains(buf.String(), "<out>conv</out>"), buf.String())
	})
}

const testSourceXML = `<root/>`

func compileStylesheetString(t *testing.T, src string) *xslt3.Stylesheet {
	t.Helper()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	ss, err := xslt3.CompileStylesheet(t.Context(), doc)
	require.NoError(t, err)
	return ss
}

func parseTransformSource(t *testing.T) *helium.Document {
	t.Helper()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(testSourceXML))
	require.NoError(t, err)
	return doc
}

func TestParameters(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		p := xslt3.NewParameters()
		p.Set("x", xpath3.SingleString("hello"))
		require.Equal(t, 1, p.Len())

		seq, ok := p.Get("x")
		require.True(t, ok)
		require.Equal(t, 1, seq.Len())
	})

	t.Run("set string", func(t *testing.T) {
		p := xslt3.NewParameters()
		p.SetString("name", "value")

		seq, ok := p.Get("name")
		require.True(t, ok)
		require.Equal(t, 1, seq.Len())
	})

	t.Run("set atomic", func(t *testing.T) {
		p := xslt3.NewParameters()
		p.SetAtomic("num", xpath3.AtomicValue{
			TypeName: xpath3.TypeInteger,
			Value:    42,
		})

		seq, ok := p.Get("num")
		require.True(t, ok)
		require.Equal(t, 1, seq.Len())
	})

	t.Run("get missing", func(t *testing.T) {
		p := xslt3.NewParameters()
		_, ok := p.Get("missing")
		require.False(t, ok)
	})

	t.Run("delete", func(t *testing.T) {
		p := xslt3.NewParameters()
		p.SetString("a", "1")
		p.SetString("b", "2")
		require.Equal(t, 2, p.Len())

		p.Delete("a")
		require.Equal(t, 1, p.Len())

		_, ok := p.Get("a")
		require.False(t, ok)

		_, ok = p.Get("b")
		require.True(t, ok)
	})

	t.Run("clear", func(t *testing.T) {
		p := xslt3.NewParameters()
		p.SetString("a", "1")
		p.SetString("b", "2")
		require.Equal(t, 2, p.Len())

		p.Clear()
		require.Equal(t, 0, p.Len())
	})

	t.Run("clone", func(t *testing.T) {
		p := xslt3.NewParameters()
		p.SetString("x", "original")

		clone := p.Clone()
		require.Equal(t, 1, clone.Len())

		// Mutating the clone does not affect the original.
		clone.SetString("x", "mutated")
		clone.SetString("y", "new")

		require.Equal(t, 1, p.Len())
		seq, ok := p.Get("x")
		require.True(t, ok)
		require.Equal(t, 1, seq.Len())
	})

	t.Run("clone nil", func(t *testing.T) {
		var p *xslt3.Parameters
		clone := p.Clone()
		require.Nil(t, clone)
	})

	t.Run("clone empty", func(t *testing.T) {
		p := xslt3.NewParameters()
		clone := p.Clone()
		require.NotNil(t, clone)
		require.Equal(t, 0, clone.Len())
	})

	t.Run("overwrite", func(t *testing.T) {
		p := xslt3.NewParameters()
		p.SetString("x", "first")
		p.SetString("x", "second")
		require.Equal(t, 1, p.Len())

		seq, ok := p.Get("x")
		require.True(t, ok)
		require.Equal(t, 1, seq.Len())
	})

	t.Run("the constructor", func(t *testing.T) {
		p := xslt3.NewParameters()
		require.NotNil(t, p)
		require.Equal(t, 0, p.Len())
	})
}

func TestNilInput(t *testing.T) {
	t.Run("nil stylesheet transform", func(t *testing.T) {
		ctx := t.Context()
		var ss *xslt3.Stylesheet
		require.NotPanics(t, func() {
			_, err := ss.Transform(nil).Do(ctx)
			require.Error(t, err)
		})
	})

	t.Run("nil stylesheet apply templates", func(t *testing.T) {
		ctx := t.Context()
		var ss *xslt3.Stylesheet
		require.NotPanics(t, func() {
			_, err := ss.ApplyTemplates(nil).Do(ctx)
			require.Error(t, err)
		})
	})

	t.Run("nil stylesheet call template", func(t *testing.T) {
		ctx := t.Context()
		var ss *xslt3.Stylesheet
		require.NotPanics(t, func() {
			_, err := ss.CallTemplate("x").Do(ctx)
			require.Error(t, err)
		})
	})

	t.Run("nil stylesheet call function", func(t *testing.T) {
		ctx := t.Context()
		var ss *xslt3.Stylesheet
		require.NotPanics(t, func() {
			_, err := ss.CallFunction("x").Do(ctx)
			require.Error(t, err)
		})
	})

	t.Run("nil stylesheet serialize", func(t *testing.T) {
		ctx := t.Context()
		var ss *xslt3.Stylesheet
		require.NotPanics(t, func() {
			_, err := ss.Transform(nil).Serialize(ctx)
			require.Error(t, err)
		})
	})

	t.Run("nil stylesheet write to", func(t *testing.T) {
		ctx := t.Context()
		var ss *xslt3.Stylesheet
		require.NotPanics(t, func() {
			var buf []byte
			err := ss.Transform(nil).WriteTo(ctx, nil)
			_ = buf
			require.Error(t, err)
		})
	})

	t.Run("zero compiler compile", func(t *testing.T) {
		ctx := t.Context()
		var c xslt3.Compiler
		require.NotPanics(t, func() {
			_, err := c.Compile(ctx, nil)
			require.Error(t, err)
		})
	})

	t.Run("zero compiler fluent", func(t *testing.T) {
		var c xslt3.Compiler
		require.NotPanics(t, func() {
			c2 := c.BaseURI("http://example.com")
			_ = c2
		})
	})

	t.Run("compiler compile nil doc", func(t *testing.T) {
		ctx := t.Context()
		require.NotPanics(t, func() {
			_, err := xslt3.NewCompiler().Compile(ctx, nil)
			require.Error(t, err)
		})
	})

	t.Run("compile stylesheet nil doc", func(t *testing.T) {
		ctx := t.Context()
		require.NotPanics(t, func() {
			_, err := xslt3.CompileStylesheet(ctx, nil)
			require.Error(t, err)
		})
	})

	t.Run("package level transform nil stylesheet", func(t *testing.T) {
		ctx := t.Context()
		require.NotPanics(t, func() {
			_, err := xslt3.Transform(ctx, nil, nil)
			require.Error(t, err)
		})
	})

	t.Run("zero invocation do", func(t *testing.T) {
		ctx := t.Context()
		var inv xslt3.Invocation
		require.NotPanics(t, func() {
			_, err := inv.Do(ctx)
			require.Error(t, err)
		})
	})

	t.Run("zero invocation serialize", func(t *testing.T) {
		ctx := t.Context()
		var inv xslt3.Invocation
		require.NotPanics(t, func() {
			_, err := inv.Serialize(ctx)
			require.Error(t, err)
		})
	})

	t.Run("transform nil stylesheet", func(t *testing.T) {
		_, err := xslt3.Transform(t.Context(), nil, nil)
		require.EqualError(t, err, "xslt3: nil stylesheet")
	})

	t.Run("transform string nil stylesheet", func(t *testing.T) {
		_, err := xslt3.TransformString(t.Context(), nil, nil)
		require.EqualError(t, err, "xslt3: nil stylesheet")
	})

	t.Run("transform to writer nil stylesheet", func(t *testing.T) {
		var buf bytes.Buffer
		err := xslt3.TransformToWriter(t.Context(), nil, nil, &buf)
		require.EqualError(t, err, "xslt3: nil stylesheet")
	})
}

func TestStylesheetReuse(t *testing.T) {
	t.Run("serialization", func(t *testing.T) {
		// The stylesheet has no xsl:output declaration (so ss.outputs[""] starts nil).
		// The template produces a primary xsl:result-document with
		// omit-xml-declaration="yes". During executeTransform, this creates a new
		// OutputDef on ss.outputs[""] with OmitDeclaration=true (line 437+451).
		//
		// If the stylesheet is mutated, the second transform inherits
		// OmitDeclaration=true even without xsl:result-document, changing the
		// serialized output.
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document omit-xml-declaration="yes">
      <out>hello</out>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`)

		source := parseTransformSource(t)

		// First transform — xsl:result-document sets omit-xml-declaration=yes
		result1, err := ss.Transform(source).Serialize(t.Context())
		require.NoError(t, err)

		// Second transform with the SAME compiled stylesheet
		result2, err := ss.Transform(source).Serialize(t.Context())
		require.NoError(t, err)

		// Both results must be identical. If the stylesheet was mutated, the
		// second run may differ (e.g., xml declaration present vs absent).
		require.Equal(t, result1, result2,
			"second transform produced different output — stylesheet was mutated during first execution")
	})

	t.Run("character map", func(t *testing.T) {
		// A stylesheet with a character-map. The ResolvedCharMap is written
		// to ss.outputs[""] during execution. Verify repeated execution
		// produces identical results.
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" use-character-maps="copy"/>
  <xsl:character-map name="copy">
    <xsl:output-character character="&#169;" string="(c)"/>
  </xsl:character-map>
  <xsl:template match="/">
    <out>&#169;</out>
  </xsl:template>
</xsl:stylesheet>`)

		source := parseTransformSource(t)

		result1, err := ss.Transform(source).Serialize(t.Context())
		require.NoError(t, err)
		require.True(t, strings.Contains(result1, "(c)"), "character map not applied: %s", result1)

		result2, err := ss.Transform(source).Serialize(t.Context())
		require.NoError(t, err)

		require.Equal(t, result1, result2,
			"second transform produced different output — stylesheet was mutated during first execution")
	})

	t.Run("concurrent", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" use-character-maps="copy"/>
  <xsl:character-map name="copy">
    <xsl:output-character character="&#169;" string="(c)"/>
  </xsl:character-map>
  <xsl:template match="/">
    <out>&#169;</out>
  </xsl:template>
</xsl:stylesheet>`)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		// Run multiple transforms concurrently on the same stylesheet.
		// Under -race this will detect data races on shared stylesheet state.
		var wg sync.WaitGroup
		errs := make([]error, 10)
		results := make([]string, 10)
		for i := range 10 {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				r, err := ss.Transform(source).Serialize(t.Context())
				errs[idx] = err
				results[idx] = r
			}(i)
		}
		wg.Wait()

		for i := range 10 {
			require.NoError(t, errs[i], "goroutine %d failed", i)
		}

		// All results must be identical
		for i := 1; i < 10; i++ {
			require.Equal(t, results[0], results[i], "goroutine %d produced different output", i)
		}
	})
}

// fn:transform error codes asserted by the option-validation table.
const (
	wantFOXT0002 = "FOXT0002"
	wantXPTY0004 = "XPTY0004"
)

// XSLT-namespace requested-property local names asserted by the capability table.
const (
	propSupportsStreaming = "supports-streaming"
	propIsSchemaAware     = "is-schema-aware"
)

// nsInitialTemplateStylesheet declares a named template in a non-empty
// namespace (app:main). It exercises the initial-template QName resolution: a
// QName-valued initial-template option must keep its namespace so the
// {http://www.example.com}main template is found (QT3 fn-transform-10/11).
const nsInitialTemplateStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:app="http://www.example.com">
  <xsl:template match="/"><x><xsl:value-of select="."/></x></xsl:template>
  <xsl:template name="app:main"><out>ns-named</out></xsl:template>
</xsl:stylesheet>`

func TestTransformOptions(t *testing.T) {
	// TestTransformInitialTemplateQName proves sub-fix 1: an initial-template
	// supplied as a namespaced xs:QName resolves to the namespaced named template
	// dropping no namespace and never failing with XTDE0820.
	t.Run("an initial-template QName", func(t *testing.T) {
		sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc>this</doc>`))
		require.NoError(t, err)

		t.Run("NamespacedQNameResolves", func(t *testing.T) {
			out, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'initial-template': QName('http://www.example.com','main'), 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(nsInitialTemplateStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "ns-named")
			require.Contains(t, out, "<out")
		})

		// A namespaced QName whose namespace does not match any template still fails,
		// confirming the namespace participates in the lookup (not silently dropped).
		t.Run("WrongNamespaceFails", func(t *testing.T) {
			_, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'initial-template': QName('http://wrong.example.org','main'), 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(nsInitialTemplateStylesheet)},
				transformFns(),
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), "XTDE0820")
		})
	})

	// TestTransformOptionValidation proves sub-fix 2: fn:transform rejects invalid,
	// mutually-exclusive, and mistyped option combinations (QT3 fn-transform-err-2,
	// -3, -4, -5, -6, -13, -17, -18).
	t.Run("option validation", func(t *testing.T) {
		sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc>this</doc>`))
		require.NoError(t, err)

		testCases := []struct {
			name string
			expr string
			code string
		}{
			{
				// err-2: stylesheet-text and stylesheet-location are mutually exclusive.
				name: "TwoStylesheetSources_TextAndLocation",
				expr: `transform(map{'stylesheet-text': $ss, 'stylesheet-location': 'x.xsl', 'source-node': .})?output`,
				code: wantFOXT0002,
			},
			{
				// err-3: stylesheet-text and stylesheet-node are mutually exclusive.
				name: "TwoStylesheetSources_TextAndNode",
				expr: `transform(map{'stylesheet-text': $ss, 'stylesheet-node': ., 'source-node': .})?output`,
				code: wantFOXT0002,
			},
			{
				// err-6: initial-mode and initial-template are mutually exclusive.
				name: "TwoEntryPoints_ModeAndTemplate",
				expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'initial-mode': QName('','m'), 'initial-template': QName('','t')})?output`,
				code: wantFOXT0002,
			},
			{
				// err-17: source-node and initial-match-selection are mutually exclusive.
				name: "SourceNodeAndInitialMatchSelection",
				expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'initial-match-selection': .})?output`,
				code: wantFOXT0002,
			},
			{
				// err-13: delivery-format value not one of document/serialized/raw.
				name: "BadDeliveryFormat",
				expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'doc'})?output`,
				code: wantFOXT0002,
			},
			{
				// err-18: stylesheet-params keys must be QNames, not strings.
				name: "StylesheetParamsStringKey",
				expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'stylesheet-params': map{'debug': true()}})?output`,
				code: wantFOXT0002,
			},
			{
				// err-4: xslt-version value is mistyped (string, not numeric).
				name: "XSLTVersionWrongType",
				expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'xslt-version': '2.0'})?output`,
				code: wantXPTY0004,
			},
			{
				// err-5: stylesheet-params value is mistyped (string, not a map).
				name: "StylesheetParamsWrongType",
				expr: `transform(map{'stylesheet-text': $ss, 'source-node': ., 'stylesheet-params': 'v'})?output`,
				code: wantXPTY0004,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := evalTransform(t, tc.expr, sourceDoc,
					map[string]xpath3.Sequence{"ss": xpath3.SingleString(simpleInvokableStylesheet)},
					transformFns(),
				)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.code)
			})
		}
	})

	// TestTransformRequestedProperties proves sub-fix 3: an unsatisfiable
	// requested-properties entry raises FOXT0006. helium advertises
	// supports-streaming = false and is otherwise schema-aware / backwards-compatible
	// / namespace-axis-capable (QT3 fn-transform-71, -73, -75, -77).
	t.Run("requested properties", func(t *testing.T) {
		sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc>this</doc>`))
		require.NoError(t, err)

		const xsltNS = "http://www.w3.org/1999/XSL/Transform"

		unsatisfiable := []struct {
			name     string
			property string
			value    string
		}{
			// fn-transform-71: request a non-schema-aware processor.
			{"NonSchemaAware", propIsSchemaAware, "false()"},
			// fn-transform-73: request no backwards-compatibility support.
			{"NoBackwardsCompatibility", "supports-backwards-compatibility", "false()"},
			// fn-transform-75: request no namespace-axis support.
			{"NoNamespaceAxis", "supports-namespace-axis", "false()"},
			// fn-transform-77: request streaming support (helium does not stream).
			{"Streaming", propSupportsStreaming, "true()"},
			// A boolean requested as the xs:string lexical form "yes" is honored the
			// same as true() — an unsatisfiable streaming request still raises.
			{"StreamingYesString", propSupportsStreaming, `'yes'`},
			// The "1" lexical form is likewise a true value.
			{"StreamingOneString", propSupportsStreaming, `'1'`},
			// A false-valued property requested as "no" against an advertised-true
			// capability is also unsatisfiable.
			{"NoSchemaAwareNoString", propIsSchemaAware, `'no'`},
		}

		for _, tc := range unsatisfiable {
			t.Run("Unsatisfiable_"+tc.name, func(t *testing.T) {
				expr := `transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'requested-properties': map{QName('` + xsltNS + `','` + tc.property + `'): ` + tc.value + `}})?output`
				_, err := evalTransform(t, expr, sourceDoc,
					map[string]xpath3.Sequence{"ss": xpath3.SingleString(simpleInvokableStylesheet)},
					transformFns(),
				)
				require.Error(t, err)
				require.Contains(t, err.Error(), "FOXT0006")
			})
		}

		// Satisfiable requests must NOT raise: they match helium's advertised
		// capabilities, so the transform proceeds normally.
		satisfiable := []struct {
			name     string
			property string
			value    string
		}{
			{"SchemaAware", propIsSchemaAware, "true()"},
			{"BackwardsCompatibility", "supports-backwards-compatibility", "true()"},
			{"NamespaceAxis", "supports-namespace-axis", "true()"},
			{"NoStreaming", propSupportsStreaming, "false()"},
			// The xs:string lexical forms of a satisfiable request must also succeed.
			{"NoStreamingNoString", propSupportsStreaming, `'no'`},
			{"NoStreamingZeroString", propSupportsStreaming, `'0'`},
			{"SchemaAwareYesString", propIsSchemaAware, `'yes'`},
		}

		for _, tc := range satisfiable {
			t.Run("Satisfiable_"+tc.name, func(t *testing.T) {
				expr := `transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'requested-properties': map{QName('` + xsltNS + `','` + tc.property + `'): ` + tc.value + `}})?output`
				out, err := evalTransform(t, expr, sourceDoc,
					map[string]xpath3.Sequence{"ss": xpath3.SingleString(simpleInvokableStylesheet)},
					transformFns(),
				)
				require.NoError(t, err)
				require.Contains(t, out, "<out>this</out>")
			})
		}
	})

	// TestTransformSerializationParams proves sub-fix 4: the serialization-params
	// option is applied to the serialized delivery output (QT3 fn-transform-30/31).
	t.Run("serialization parameters", func(t *testing.T) {
		sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		// fn-transform-30: indent yes vs no must produce different serializations.
		t.Run("IndentChangesOutput", func(t *testing.T) {
			indented, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'indent': true()}})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(nestedOutputStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)

			compact, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'indent': false()}})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(nestedOutputStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)

			require.NotEqual(t, indented, compact, "indent=yes and indent=no must serialize differently")
		})

		// The method serialization parameter switches the output method: with
		// method=text only text nodes are emitted, so the element markup disappears.
		t.Run("MethodText", func(t *testing.T) {
			out, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'method': 'text'}})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(nestedOutputStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Equal(t, "x", strings.TrimSpace(out))
		})

		// A serialization-params entry present with an empty-sequence value resets
		// that parameter to its serialization default, overriding the inherited
		// xsl:output value. Here method="text" from the stylesheet is reset to the
		// "xml" default, so element markup reappears in the output.
		t.Run("EmptySequenceResetsToDefault", func(t *testing.T) {
			// Baseline: inherited method="text" emits only the text node.
			inherited, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(methodTextOutputStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Equal(t, "x", strings.TrimSpace(inherited))
			require.NotContains(t, inherited, "<out")

			// method: () resets the method to its "xml" default instead of leaving
			// the inherited text method in place, so the element markup is emitted.
			reset, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'method': ()}})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(methodTextOutputStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Contains(t, reset, "<out")
			require.Contains(t, reset, "<b>x</b>")
		})

		// QT3 fn-transform-31: serialization-params map{'method':'html'} over a
		// stylesheet with no xsl:output must switch to html serialization (void
		// <br> element, no self-closing slash) without raising SESU0007. The base
		// output def carries the inherited xml version "1.0"; because the method is
		// switched to html via serialization-params without an explicit version,
		// the version must not carry over and trigger the SESU0007 html-version
		// check.
		t.Run("MethodHTMLResetsInheritedVersion", func(t *testing.T) {
			out, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'method': 'html'}})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(htmlOutputStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<br>")
			require.NotContains(t, out, "<br/>")
			require.NotContains(t, out, "<br />")
		})

		// An EXPLICIT version="1.0" alongside method="html" must still raise
		// SESU0007: html only supports versions 4.0/4.01/5.0. The version reset
		// applies only when the serialization-params map omits version.
		t.Run("MethodHTMLExplicitBadVersionErrors", func(t *testing.T) {
			_, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'method': 'html', 'version': '1.0'}})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(htmlOutputStylesheet)},
				transformFns(),
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), "SESU0007")
		})

		// An EXPLICIT base stylesheet version (<xsl:output method="html"
		// version="1.0"/>) must survive serialization-params that omit version: the
		// html-method version reset applies only to an inherited (non-explicit)
		// version, so SESU0007 must still fire here.
		t.Run("ExplicitBaseVersionSurvivesUnrelatedParams", func(t *testing.T) {
			_, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized', 'serialization-params': map{'indent': true()}})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(htmlExplicitBadVersionStylesheet)},
				transformFns(),
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), "SESU0007")
		})
	})
}

// simpleInvokableStylesheet has both a match="/" rule and a named template so it
// is invokable through any entry point the validation tests supply.
const simpleInvokableStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="."/></out></xsl:template>
  <xsl:template name="t"><out>named</out></xsl:template>
</xsl:stylesheet>`

// nestedOutputStylesheet emits a nested element structure so that the indent
// serialization parameter produces observably different output.
const nestedOutputStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><a><b>x</b></a></out></xsl:template>
</xsl:stylesheet>`

// methodTextOutputStylesheet declares xsl:output method="text" so the inherited
// serialization method differs from the "xml" default. It lets a
// serialization-params entry present with an empty-sequence value be observed
// resetting the method back to its default.
const methodTextOutputStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/"><out><a><b>x</b></a></out></xsl:template>
</xsl:stylesheet>`

// htmlOutputStylesheet has an <html> root that emits an empty <br/> element and
// declares NO xsl:output. It lets serialization-params map{'method':'html'}
// switch the output method to html (QT3 fn-transform-31): the html serializer
// emits the void <br> element without a self-closing slash.
const htmlOutputStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><html><body><br/></body></html></xsl:template>
</xsl:stylesheet>`

// htmlExplicitBadVersionStylesheet declares xsl:output method="html"
// version="1.0" explicitly (an unsupported html version). serialization-params
// that omit version must NOT clear this explicit base version, so SESU0007 must
// still fire.
const htmlExplicitBadVersionStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="html" version="1.0"/>
  <xsl:template match="/"><html><body><br/></body></html></xsl:template>
</xsl:stylesheet>`

var errHandlerAborted = errors.New("handler aborted transform")

type messageAbortHandler struct {
	called bool
}

func (r *messageAbortHandler) HandleMessage(msg string, terminate bool) error {
	r.called = true
	return errHandlerAborted
}

type resultDocumentAbortHandler struct {
	called bool
}

func (r *resultDocumentAbortHandler) HandleResultDocument(href string, doc *helium.Document, outDef *xslt3.OutputDef) error {
	r.called = true
	return errHandlerAborted
}

type rawResultAbortHandler struct {
	called bool
}

func (r *rawResultAbortHandler) HandleRawResult(seq xpath3.Sequence) error {
	r.called = true
	return errHandlerAborted
}

type primaryItemsAbortHandler struct {
	called bool
}

func (r *primaryItemsAbortHandler) HandlePrimaryItems(seq xpath3.Sequence) error {
	r.called = true
	return errHandlerAborted
}

type annotationAbortHandler struct {
	called bool
}

func (r *annotationAbortHandler) HandleAnnotations(annotations map[helium.Node]string, declarations xpath3.SchemaDeclarations) error {
	r.called = true
	return errHandlerAborted
}

func TestHandlerError(t *testing.T) {
	t.Run("a message handler error aborts the transform", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:message terminate="no">boom</xsl:message>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

		handler := &messageAbortHandler{}
		_, err := ss.Transform(parseTransformSource(t)).MessageHandler(handler).Do(t.Context())

		require.True(t, handler.called)
		require.ErrorIs(t, err, errHandlerAborted)
	})

	t.Run("a result-document handler error aborts the transform", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:result-document href="secondary.xml"><secondary/></xsl:result-document>
    <out/>
  </xsl:template>
</xsl:stylesheet>`)

		handler := &resultDocumentAbortHandler{}
		_, err := ss.Transform(parseTransformSource(t)).ResultDocumentHandler(handler).Do(t.Context())

		require.True(t, handler.called)
		require.ErrorIs(t, err, errHandlerAborted)
	})

	t.Run("a raw-result handler error aborts the transform", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/" as="xs:string">
    <xsl:sequence select="'hello'"/>
  </xsl:template>
</xsl:stylesheet>`)

		handler := &rawResultAbortHandler{}
		_, err := ss.Transform(parseTransformSource(t)).RawResultHandler(handler).Do(t.Context())

		require.True(t, handler.called)
		require.ErrorIs(t, err, errHandlerAborted)
	})

	t.Run("a primary-items handler error aborts the transform", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="adaptive"/>
  <xsl:template match="/">
    <xsl:sequence select="1"/>
  </xsl:template>
</xsl:stylesheet>`)

		handler := &primaryItemsAbortHandler{}
		_, err := ss.Transform(parseTransformSource(t)).PrimaryItemsHandler(handler).Do(t.Context())

		require.True(t, handler.called)
		require.ErrorIs(t, err, errHandlerAborted)
	})

	t.Run("an annotation handler error aborts the transform", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema namespace="">
    <xs:schema>
      <xs:element name="out" type="xs:integer"/>
    </xs:schema>
  </xsl:import-schema>
  <xsl:template match="/">
    <xsl:element name="out" type="xs:integer">42</xsl:element>
  </xsl:template>
</xsl:stylesheet>`)

		handler := &annotationAbortHandler{}
		_, err := ss.Transform(parseTransformSource(t)).AnnotationHandler(handler).Do(t.Context())

		require.True(t, handler.called)
		require.ErrorIs(t, err, errHandlerAborted)
	})
}
