package xslt3_test

import (
	"io"
	"os"
	"path/filepath"
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
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(minimalStylesheet))
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
	t.Parallel()

	r := &stubURIResolver{}
	c1 := xslt3.NewCompiler()
	c2 := c1.URIResolver(r)

	// Compile a stylesheet that imports a non-existent module to trigger the resolver.
	importSheet := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:import href="missing.xsl"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(importSheet))
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
	t.Parallel()

	r := &stubPackageResolver{}
	c := xslt3.NewCompiler().PackageResolver(r)

	pkgSheet := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/missing"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(pkgSheet))
	require.NoError(t, err)

	_, err = c.Compile(t.Context(), doc)
	require.Error(t, err)
	require.Equal(t, "http://example.com/missing", r.calledName)
}

func TestUsePackageWithoutResolver(t *testing.T) {
	t.Parallel()

	pkgSheet := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:use-package name="http://example.com/some-package"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(pkgSheet))
	require.NoError(t, err)

	// Compile without a PackageResolver — must fail, not silently succeed.
	_, err = xslt3.NewCompiler().Compile(t.Context(), doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "PackageResolver")
}

func TestUsePackageExcludedByUseWhenDoesNotRequireResolver(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:param name="use-package" as="xs:boolean" static="yes" select="false()"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"/>
  <xsl:use-package name="http://example.com/some-package" use-when="$use-package"/>
  <xsl:template match="/"><out>ok</out></xsl:template>
</xsl:stylesheet>`))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	result, err := ss.Transform(parseTransformSource(t)).Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, result, "<out>ok</out>")
}

func TestCompilerStaticParameters(t *testing.T) {
	t.Parallel()

	// Static parameters affect compile-time use-when evaluation.
	// When debug='yes', the use-when branch includes the debug template.
	p := xslt3.NewParameters()
	p.SetString("debug", "yes")

	c1 := xslt3.NewCompiler().StaticParameters(p)

	// Mutating the original Parameters does not affect c1.
	p.SetString("debug", "no")

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
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
	t.Parallel()

	c1 := xslt3.NewCompiler().SetStaticParameter("mode", xpath3.SingleString("a"))
	c2 := c1.SetStaticParameter("mode", xpath3.SingleString("b"))

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
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
	t.Parallel()

	c1 := xslt3.NewCompiler().SetStaticParameter("mode", xpath3.SingleString("a"))
	c2 := c1.ClearStaticParameters()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
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
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<not-a-stylesheet/>`))
	require.NoError(t, err)

	require.Panics(t, func() {
		xslt3.NewCompiler().MustCompile(t.Context(), doc)
	})
}

func TestMustCompileSuccess(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(minimalStylesheet))
	require.NoError(t, err)

	require.NotPanics(t, func() {
		ss := xslt3.NewCompiler().MustCompile(t.Context(), doc)
		require.NotNil(t, ss)
	})
}

func TestCompilerCloneOnWrite(t *testing.T) {
	t.Parallel()

	c1 := xslt3.NewCompiler().BaseURI("file:///a.xsl")
	c2 := c1.BaseURI("file:///b.xsl")

	doc, err := helium.NewParser().Parse(t.Context(), []byte(minimalStylesheet))
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
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(minimalStylesheet))
	require.NoError(t, err)

	ss, err := xslt3.CompileStylesheet(t.Context(), doc)
	require.NoError(t, err)
	require.NotNil(t, ss)
}

func TestAttributeSetCycleDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xsl  string
	}{
		{
			name: "direct self-cycle",
			xsl: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:attribute-set name="a" use-attribute-sets="a"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`,
		},
		{
			name: "indirect two-node cycle",
			xsl: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:attribute-set name="a" use-attribute-sets="b"/>
  <xsl:attribute-set name="b" use-attribute-sets="a"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			doc, err := helium.NewParser().Parse(ctx, []byte(tc.xsl))
			require.NoError(t, err)

			_, err = xslt3.CompileStylesheet(ctx, doc)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), "XTSE0720"),
				"expected XTSE0720 in error, got: %v", err)
		})
	}
}

func TestCompileFileLoadsDTDDefinedExternalEntityInIncludedStylesheet(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.xsl"), []byte(`<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:include href="child.xsl"/>
  <xsl:template match="/">
    <out value="{$var}"/>
  </xsl:template>
</xsl:stylesheet>`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "child.xsl"), []byte(`<?xml version="1.0"?>
<!DOCTYPE xsl:stylesheet SYSTEM "child.dtd">
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  &inject;
</xsl:stylesheet>`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "child.dtd"), []byte(`<!ENTITY inject SYSTEM "inject.xsl">`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "inject.xsl"), []byte(`<?xml version="1.0"?>
<xsl:variable xmlns:xsl="http://www.w3.org/1999/XSL/Transform" name="var" select="'from-dtd-entity'"/>`), 0o644))

	mainPath := filepath.Join(tmpDir, "main.xsl")
	p := helium.NewParser().LoadExternalDTD(true).SubstituteEntities(true).BaseURI(mainPath)
	mainData, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	doc, err := p.Parse(t.Context(), mainData)
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(mainPath).Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, result, `value="from-dtd-entity"`)
}
