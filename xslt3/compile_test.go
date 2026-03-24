package xslt3_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

const minimalStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

func TestCompilerBaseURI(t *testing.T) {
	doc, err := helium.Parse(t.Context(), []byte(minimalStylesheet))
	require.NoError(t, err)

	c1 := xslt3.NewCompiler()
	c2 := c1.BaseURI("file:///a.xsl")

	// c1 is not mutated by c2
	ss1, err := c1.Compile(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss1)

	ss2, err := c2.Compile(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss2)
}

type stubURIResolver struct {
	calledWith string
}

func (r *stubURIResolver) Resolve(uri string) (io.ReadCloser, error) {
	r.calledWith = uri
	return nil, os.ErrNotExist
}

func TestCompilerURIResolver(t *testing.T) {
	r := &stubURIResolver{}
	c1 := xslt3.NewCompiler()
	c2 := c1.URIResolver(r)

	// Compile a stylesheet that imports a non-existent module to trigger the resolver.
	importSheet := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:import href="missing.xsl"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.Parse(t.Context(), []byte(importSheet))
	require.NoError(t, err)

	// c1 (no resolver) should fail differently than c2 (with resolver).
	_, err = c2.Compile(t.Context(), doc)
	// The resolver was called and returned os.ErrNotExist.
	require.Error(t, err)
	require.NotEmpty(t, r.calledWith)

	// c1 must not have been affected.
	_, err = c1.Compile(t.Context(), doc)
	require.Error(t, err)
}

type stubPackageResolver struct {
	calledName string
}

func (r *stubPackageResolver) ResolvePackage(name string, version string) (io.ReadCloser, string, error) {
	r.calledName = name
	return nil, "", os.ErrNotExist
}

func TestCompilerPackageResolver(t *testing.T) {
	r := &stubPackageResolver{}
	c := xslt3.NewCompiler().PackageResolver(r)

	pkgSheet := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/missing"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.Parse(t.Context(), []byte(pkgSheet))
	require.NoError(t, err)

	_, err = c.Compile(t.Context(), doc)
	require.Error(t, err)
	require.Equal(t, "http://example.com/missing", r.calledName)
}

func TestCompilerStaticParameters(t *testing.T) {
	// Static parameters affect compile-time use-when evaluation.
	// When debug='yes', the use-when branch includes the debug template.
	p := xslt3.NewParameters()
	p.SetString("debug", "yes")

	c1 := xslt3.NewCompiler().StaticParameters(p)

	// Mutating the original Parameters does not affect c1.
	p.SetString("debug", "no")

	doc, err := helium.Parse(t.Context(), []byte(`<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="debug" static="yes" select="'no'"/>
  <xsl:template match="/" use-when="$debug = 'yes'">
    <out>debug-on</out>
  </xsl:template>
  <xsl:template match="/" use-when="$debug != 'yes'">
    <out>debug-off</out>
  </xsl:template>
</xsl:stylesheet>`))
	require.NoError(t, err)

	ss, err := c1.Compile(t.Context(), doc)
	require.NoError(t, err)

	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "debug-on")
}

func TestCompilerSetStaticParameter(t *testing.T) {
	c1 := xslt3.NewCompiler().SetStaticParameter("mode", xpath3.SingleString("a"))
	c2 := c1.SetStaticParameter("mode", xpath3.SingleString("b"))

	doc, err := helium.Parse(t.Context(), []byte(`<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="mode" static="yes" select="'default'"/>
  <xsl:template match="/" use-when="$mode = 'a'">
    <out>mode-a</out>
  </xsl:template>
  <xsl:template match="/" use-when="$mode = 'b'">
    <out>mode-b</out>
  </xsl:template>
  <xsl:template match="/" use-when="$mode = 'default'">
    <out>mode-default</out>
  </xsl:template>
</xsl:stylesheet>`))
	require.NoError(t, err)

	source := parseTransformSource(t)

	// c1 has mode=a
	ss1, err := c1.Compile(t.Context(), doc)
	require.NoError(t, err)
	r1, err := ss1.Transform(source).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, r1, "mode-a")

	// c2 has mode=b (overrides a)
	ss2, err := c2.Compile(t.Context(), doc)
	require.NoError(t, err)
	r2, err := ss2.Transform(source).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, r2, "mode-b")
}

func TestCompilerClearStaticParameters(t *testing.T) {
	c1 := xslt3.NewCompiler().SetStaticParameter("mode", xpath3.SingleString("a"))
	c2 := c1.ClearStaticParameters()

	doc, err := helium.Parse(t.Context(), []byte(`<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="mode" static="yes" select="'default'"/>
  <xsl:template match="/" use-when="$mode = 'a'">
    <out>mode-a</out>
  </xsl:template>
  <xsl:template match="/" use-when="$mode = 'default'">
    <out>mode-default</out>
  </xsl:template>
</xsl:stylesheet>`))
	require.NoError(t, err)

	// c2 has no static params, so mode should be 'default'
	ss, err := c2.Compile(t.Context(), doc)
	require.NoError(t, err)
	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "mode-default")
}

func TestMustCompilePanicsOnError(t *testing.T) {
	doc, err := helium.Parse(t.Context(), []byte(`<not-a-stylesheet/>`))
	require.NoError(t, err)

	require.Panics(t, func() {
		xslt3.NewCompiler().MustCompile(t.Context(), doc)
	})
}

func TestMustCompileSuccess(t *testing.T) {
	doc, err := helium.Parse(t.Context(), []byte(minimalStylesheet))
	require.NoError(t, err)

	require.NotPanics(t, func() {
		ss := xslt3.NewCompiler().MustCompile(t.Context(), doc)
		require.NotNil(t, ss)
	})
}

func TestCompilerCloneOnWrite(t *testing.T) {
	c1 := xslt3.NewCompiler().BaseURI("file:///a.xsl")
	c2 := c1.BaseURI("file:///b.xsl")

	doc, err := helium.Parse(t.Context(), []byte(minimalStylesheet))
	require.NoError(t, err)

	// Both compilers should compile without interference.
	ss1, err := c1.Compile(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss1)

	ss2, err := c2.Compile(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss2)
}

func TestCompileStylesheetConvenience(t *testing.T) {
	doc, err := helium.Parse(t.Context(), []byte(minimalStylesheet))
	require.NoError(t, err)

	ss, err := xslt3.CompileStylesheet(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss)
}

func TestCompileFileConvenience(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.xsl"
	require.NoError(t, os.WriteFile(path, []byte(minimalStylesheet), 0644))

	ss, err := xslt3.CompileFile(t.Context(), path)
	require.NoError(t, err)
	require.NotNil(t, ss)

	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.True(t, strings.Contains(result, "<out/>") || strings.Contains(result, "<out></out>"), result)
}
