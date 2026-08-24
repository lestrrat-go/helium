package xslt3_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

func TestCompiler(t *testing.T) {
	t.Run("base URI", func(t *testing.T) {
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
	})

	t.Run("URI resolver", func(t *testing.T) {
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
	})

	t.Run("package resolver", func(t *testing.T) {
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
	})

	t.Run("static parameters", func(t *testing.T) {
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
	})

	t.Run("set static parameter", func(t *testing.T) {
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
	})

	t.Run("clear static parameters", func(t *testing.T) {
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
	})

	t.Run("clone on write", func(t *testing.T) {
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
	})
}

type stubURIResolver struct {
	calledWith string
}

func (r *stubURIResolver) Resolve(uri string) (io.ReadCloser, error) {
	r.calledWith = uri
	return nil, os.ErrNotExist
}

type stubPackageResolver struct {
	calledName string
}

func (r *stubPackageResolver) ResolvePackage(name string, version string) (io.ReadCloser, string, error) {
	r.calledName = name
	return nil, "", os.ErrNotExist
}

func TestUsePackage(t *testing.T) {
	t.Run("without resolver", func(t *testing.T) {
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
	})

	t.Run("excluded by use when does not require resolver", func(t *testing.T) {
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
	})
}

func TestMustCompile(t *testing.T) {
	t.Run("panics on error", func(t *testing.T) {
		t.Parallel()

		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<not-a-stylesheet/>`))
		require.NoError(t, err)

		require.Panics(t, func() {
			xslt3.NewCompiler().MustCompile(t.Context(), doc)
		})
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		doc, err := helium.NewParser().Parse(t.Context(), []byte(minimalStylesheet))
		require.NoError(t, err)

		require.NotPanics(t, func() {
			ss := xslt3.NewCompiler().MustCompile(t.Context(), doc)
			require.NotNil(t, ss)
		})
	})
}

func TestCompile(t *testing.T) {
	t.Run("the CompileStylesheet convenience function", func(t *testing.T) {
		t.Parallel()

		doc, err := helium.NewParser().Parse(t.Context(), []byte(minimalStylesheet))
		require.NoError(t, err)

		ss, err := xslt3.CompileStylesheet(t.Context(), doc)
		require.NoError(t, err)
		require.NotNil(t, ss)
	})

	t.Run("a DTD-defined external entity in an included stylesheet loads", func(t *testing.T) {
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
		// inject.xsl is loaded as an external parsed general entity; its optional
		// leading TextDecl must carry the mandatory EncodingDecl (a version-only
		// declaration is a not-wf external entity per XML 4.3.1).
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "inject.xsl"), []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xsl:variable xmlns:xsl="http://www.w3.org/1999/XSL/Transform" name="var" select="'from-dtd-entity'"/>`), 0o644))

		mainPath := filepath.Join(tmpDir, "main.xsl")
		p := helium.NewParser().LoadExternalDTD(true).SubstituteEntities(true).BaseURI(mainPath)
		mainData, err := os.ReadFile(mainPath)
		require.NoError(t, err)
		doc, err := p.Parse(t.Context(), mainData)
		require.NoError(t, err)
		// This test exercises the legacy permissive parse of an included stylesheet
		// module that pulls in content via an external DTD-defined general entity.
		// XXE is blocked by default, so opt in explicitly.
		ss, err := xslt3.NewCompiler().BaseURI(mainPath).URIResolver(osOpenResolver{}).AllowExternalEntities(true).Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		result, err := xslt3.TransformString(t.Context(), source, ss)
		require.NoError(t, err)
		require.Contains(t, result, `value="from-dtd-entity"`)
	})

	t.Run("an attribute-set cycle is detected", func(t *testing.T) {
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
	})
}

// recordingCompileResolver records every URI it is asked to resolve and serves
// the bytes registered for that URI. It proves an xsl:include / xsl:import href
// reached the compile-time URIResolver as the intended (uncorrupted) key.
type recordingCompileResolver struct {
	mu       sync.Mutex
	requests []string
	files    map[string][]byte
}

func (r *recordingCompileResolver) Resolve(uri string) (io.ReadCloser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, uri)
	data, ok := r.files[uri]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (r *recordingCompileResolver) seen(uri string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.requests, uri)
}

const (
	declInclude   = "include"
	declImport    = "import"
	stylesMainXSL = "/styles/main.xsl"
)

const childModule = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:template match="/data">
    <out value="{@v}"/>
  </xsl:template>
</xsl:stylesheet>`

func TestInclude(t *testing.T) {
	// TestIncludeAbsoluteURIHrefPassedThrough verifies that an xsl:include /
	// xsl:import href that is an absolute URI reference (with a scheme but no "://"
	// authority, e.g. "urn:shared" or "file:/modules/child.xsl") is handed to the
	// URIResolver unchanged — not filepath.Join'ed onto the stylesheet base.
	t.Run("absolute URI href passed through", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name string
			decl string // include or import
			base string // compiler base URI
			href string
		}{
			{"include urn", declInclude, stylesMainXSL, "urn:shared"},
			{"import urn", declImport, stylesMainXSL, "urn:shared"},
			{"include file scheme single slash", declInclude, stylesMainXSL, "file:/modules/child.xsl"},
			{"import data scheme", declImport, stylesMainXSL, "data:application/xslt+xml,child"},
			{"include http scheme", declInclude, stylesMainXSL, "http://example.com/modules/child.xsl"},
			// Windows drive-letter paths are filesystem paths, not URIs. With a URI
			// base they used to fall through to RFC 3986 resolution and be
			// lowercased / dot-segment-mangled; they must reach the resolver
			// verbatim. A URI base is what triggers the corruption, so use one here.
			{"include windows drive forward slash", declInclude, "mem:/styles/main.xsl", "C:/modules/child.xsl"},
			{"import windows drive back slash", declImport, "mem:/styles/main.xsl", `C:\modules\child.xsl`},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:` + tc.decl + ` href="` + tc.href + `"/>
</xsl:stylesheet>`

				resolver := &recordingCompileResolver{files: map[string][]byte{
					tc.href: []byte(childModule),
				}}

				doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
				require.NoError(t, err)

				// A non-empty base is what triggered the corruption: the
				// absolute href used to be joined/resolved against it.
				ss, err := xslt3.NewCompiler().
					BaseURI(tc.base).
					URIResolver(resolver).
					Compile(t.Context(), doc)
				require.NoError(t, err)
				require.NotNil(t, ss)

				require.True(t, resolver.seen(tc.href),
					"resolver should have been asked for %q uncorrupted; got %v", tc.href, resolver.requests)

				// Confirm the module actually loaded and its template runs.
				source, err := helium.NewParser().Parse(t.Context(), []byte(`<data v="hello"/>`))
				require.NoError(t, err)
				out, err := xslt3.TransformString(t.Context(), source, ss)
				require.NoError(t, err)
				require.Contains(t, out, `value="hello"`)
			})
		}
	})

	// TestIncludeRelativeHrefResolvedAgainstBase verifies that a relative href is
	// still resolved against the (URI) stylesheet base, not passed through bare.
	t.Run("relative href resolved against base", func(t *testing.T) {
		t.Parallel()

		main := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:include href="child.xsl"/>
</xsl:stylesheet>`

		// Base is a URI, so the relative href must resolve via RFC 3986 to a
		// sibling under the same base directory.
		resolver := &recordingCompileResolver{files: map[string][]byte{
			"mem:/styles/child.xsl": []byte(childModule),
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
		require.NoError(t, err)

		_, err = xslt3.NewCompiler().
			BaseURI("mem:/styles/main.xsl").
			URIResolver(resolver).
			Compile(t.Context(), doc)
		require.NoError(t, err)

		require.True(t, resolver.seen("mem:/styles/child.xsl"),
			"relative href should resolve against URI base; got %v", resolver.requests)
	})
}

func TestModuleBase(t *testing.T) {
	// TestModuleRootUseWhenResolvesAgainstEffectiveBase is a regression for a root
	// use-when on an included/imported stylesheet module whose ROOT element carries
	// xml:base. The root use-when must be evaluated against the module's EFFECTIVE
	// static base (its root xml:base folded into the module URI), exactly like the
	// module's own globals and templates — not the including module's base nor the
	// bare module URI.
	//
	// The module's root use-when is doc-available('flag.xml') with root
	// xml:base="sub/". Only mem://pkg/sub/flag.xml exists. With the correct base the
	// reference resolves to mem://pkg/sub/flag.xml (module included); with the wrong
	// base it resolves to mem://pkg/flag.xml (or against the including module's
	// base), which does not exist, so the whole module would be wrongly excluded.
	t.Run("a module-root use-when resolves against the effective base", func(t *testing.T) {
		const (
			mainBase  = "mem://pkg/main.xsl"
			moduleURI = "mem://pkg/inc.xsl"
			flagURI   = "mem://pkg/sub/flag.xml" // effective base (module URI + xml:base="sub/")
			module    = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xml:base="sub/"
  use-when="doc-available('flag.xml')">
  <xsl:template match="item"><out>MODULE</out></xsl:template>
</xsl:stylesheet>`
		)

		for _, link := range []struct {
			name    string
			linkElt string
		}{
			{"import", `<xsl:import href="inc.xsl"/>`},
			{"include", `<xsl:include href="inc.xsl"/>`},
		} {
			for _, flag := range []struct {
				name        string
				flagPresent bool
			}{
				{"flag_present_module_included", true},
				{"flag_absent_module_excluded", false},
			} {
				t.Run(link.name+" "+flag.name, func(t *testing.T) {
					ctx := t.Context()
					main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + link.linkElt + `
  <xsl:template match="/"><result><xsl:apply-templates select="//item"/></result></xsl:template>
</xsl:stylesheet>`

					files := map[string]string{moduleURI: module}
					if flag.flagPresent {
						files[flagURI] = `<flag/>`
					}
					resolver := &exactURIResolver{files: files}

					doc, err := helium.NewParser().Parse(ctx, []byte(main))
					require.NoError(t, err)

					ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver).Compile(ctx, doc)
					require.NoError(t, err)

					source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>BUILTIN</item></doc>`))
					require.NoError(t, err)

					result, err := ss.Transform(source).Serialize(ctx)
					require.NoError(t, err)

					if flag.flagPresent {
						require.Contains(t, result, "<out>MODULE</out>",
							"flag present: module's root use-when must resolve against effective base %q and include the module", flagURI)
						require.True(t, resolver.askedFor(flagURI),
							"root use-when doc-available must probe the effective-base URI %q; asked=%v", flagURI, resolver.asked)
					} else {
						require.NotContains(t, result, "<out>MODULE</out>",
							"flag absent: module's root use-when must evaluate false and exclude the module")
						require.Contains(t, result, "BUILTIN",
							"flag absent: with the module excluded the built-in template emits the item text")
					}
				})
			}
		}
	})

	// TestEmbeddedModuleRootUseWhenResolvesAgainstEmbeddedBase is a regression for a
	// root use-when on an EMBEDDED stylesheet module (one selected by a fragment
	// identifier, whose root element's parent is another element, not the Document).
	//
	// moduleEffectiveBaseURI deliberately returns the bare module URI for an
	// embedded root (its xml:base is normally re-applied by the later
	// stylesheetBaseURI ancestor walk during child compilation). But the ROOT
	// use-when has no such later walk, so its doc-available()/doc() must apply the
	// embedded root's own xml:base here. The embedded stylesheet carries
	// xml:base="sub/" and use-when="doc-available('flag.xml')"; only
	// mem://pkg/sub/flag.xml exists. With the correct embedded base the reference
	// resolves there and the module is included; with the bare module URI it
	// resolves to mem://pkg/flag.xml, which is absent, wrongly excluding the module.
	t.Run("an embedded module-root use-when resolves against the embedded base", func(t *testing.T) {
		const (
			mainBase  = "mem://pkg/main.xsl"
			moduleURI = "mem://pkg/emb.xsl"
			flagURI   = "mem://pkg/sub/flag.xml" // embedded root xml:base="sub/" folded onto module URI
			module    = `<?xml version="1.0"?>
<wrapper xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:stylesheet id="style1" version="3.0"
    xml:base="sub/"
    use-when="doc-available('flag.xml')">
    <xsl:template match="item"><out>EMBEDDED</out></xsl:template>
  </xsl:stylesheet>
</wrapper>`
		)

		for _, link := range []struct {
			name    string
			linkElt string
		}{
			{"import", `<xsl:import href="emb.xsl#style1"/>`},
			{"include", `<xsl:include href="emb.xsl#style1"/>`},
		} {
			t.Run(link.name, func(t *testing.T) {
				ctx := t.Context()
				main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + link.linkElt + `
  <xsl:template match="/"><result><xsl:apply-templates select="//item"/></result></xsl:template>
</xsl:stylesheet>`

				resolver := &exactURIResolver{files: map[string]string{
					moduleURI: module,
					flagURI:   `<flag/>`,
				}}

				doc, err := helium.NewParser().Parse(ctx, []byte(main))
				require.NoError(t, err)

				ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver).Compile(ctx, doc)
				require.NoError(t, err)

				source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>BUILTIN</item></doc>`))
				require.NoError(t, err)

				result, err := ss.Transform(source).Serialize(ctx)
				require.NoError(t, err)

				require.Contains(t, result, "<out>EMBEDDED</out>",
					"embedded module's root use-when must resolve against the embedded root's xml:base (effective %q) and include the module", flagURI)
				require.True(t, resolver.askedFor(flagURI),
					"root use-when doc-available must probe the embedded-base URI %q; asked=%v", flagURI, resolver.asked)
			})
		}
	})

	// TestUseWhenDocAvailableHonorsMaxResourceBytes is a regression for compile-time
	// use-when reads ignoring Compiler.MaxResourceBytes. The compile-time use-when
	// evaluator routes doc()/doc-available() through the compiler URIResolver; it
	// must also enforce the compiler's per-resource read cap, matching the runtime
	// evaluator. With a 1-byte cap, doc-available() on an over-cap resource must be
	// false; with the default cap the small resource loads.
	t.Run("use-when doc-available honors MaxResourceBytes", func(t *testing.T) {
		const (
			mainBase = "mem://pkg/main.xsl"
			flagURI  = "mem://pkg/flag.xml"
		)
		main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/" use-when="doc-available('flag.xml')"><result>WITH-FLAG</result></xsl:template>
  <xsl:template match="/" use-when="not(doc-available('flag.xml'))"><result>NO-FLAG</result></xsl:template>
</xsl:stylesheet>`

		for _, tc := range []struct {
			name   string
			cap    int64
			capSet bool
			want   string
		}{
			{"default_cap_flag_loads", 0, false, "WITH-FLAG"},
			{"tiny_cap_flag_over_limit", 1, true, "NO-FLAG"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				ctx := t.Context()
				// flag.xml is comfortably larger than the 1-byte cap.
				resolver := &exactURIResolver{files: map[string]string{flagURI: `<flag>data</flag>`}}

				doc, err := helium.NewParser().Parse(ctx, []byte(main))
				require.NoError(t, err)

				compiler := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver)
				if tc.capSet {
					compiler = compiler.MaxResourceBytes(tc.cap)
				}
				ss, err := compiler.Compile(ctx, doc)
				require.NoError(t, err)

				source, err := helium.NewParser().Parse(ctx, []byte(`<doc/>`))
				require.NoError(t, err)

				result, err := ss.Transform(source).Serialize(ctx)
				require.NoError(t, err)
				require.Contains(t, result, tc.want)
			})
		}
	})

	// TestModuleRootXMLBaseResolvesGlobals is an end-to-end regression for an
	// included/imported stylesheet module whose ROOT element carries xml:base.
	//
	// The main module gets root xml:base folded into its effective static base in
	// compile(), but an external module is loaded with c.baseURI set to the bare
	// module URI and compiled directly (loadExternalStylesheet → compileTopLevel
	// for xsl:import; the two-phase collect/compile path for xsl:include). Before
	// the fix, the module root's xml:base was silently dropped, so a global
	// variable in that module resolved doc()/document() against the bare module URI
	// ("mem://pkg/data.xml") instead of the declaration-site effective base under
	// xml:base="sub/" ("mem://pkg/sub/data.xml").
	t.Run("a module-root xml:base resolves globals", func(t *testing.T) {
		const (
			mainBase   = "mem://pkg/main.xsl"
			moduleURI  = "mem://pkg/inc.xsl"
			wantURI    = "mem://pkg/sub/data.xml" // effective base (module URI + xml:base="sub/")
			preFixURI  = "mem://pkg/data.xml"     // bare module URI (the dropped-xml:base bug)
			moduleTmpl = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xml:base="sub/">
  <xsl:variable name="g" select="doc('data.xml')/data/@v"/>
</xsl:stylesheet>`
		)

		for _, tc := range []struct {
			name    string
			linkElt string
		}{
			{"import", `<xsl:import href="inc.xsl"/>`},
			{"include", `<xsl:include href="inc.xsl"/>`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				ctx := t.Context()
				main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + tc.linkElt + `
  <xsl:template match="/"><out><xsl:value-of select="$g"/></out></xsl:template>
</xsl:stylesheet>`

				compileResolver := fileMapResolver{files: map[string]string{moduleURI: moduleTmpl}}
				doc, err := helium.NewParser().Parse(ctx, []byte(main))
				require.NoError(t, err)

				ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(compileResolver).Compile(ctx, doc)
				require.NoError(t, err)

				runtimeResolver := &recordingURIResolver{files: map[string][]byte{
					wantURI: []byte(`<data v="DEEP"/>`),
				}}
				source := parseTransformSource(t)
				result, err := ss.Transform(source).URIResolver(runtimeResolver).Serialize(ctx)
				require.NoError(t, err)

				require.True(t, runtimeResolver.seen(wantURI),
					"global doc() must resolve against the module's effective base %q; got %v", wantURI, runtimeResolver.requests)
				require.False(t, runtimeResolver.seen(preFixURI),
					"global doc() must NOT resolve against the bare module URI %q (dropped xml:base); got %v", preFixURI, runtimeResolver.requests)
				require.Contains(t, result, "<out>DEEP</out>")
			})
		}
	})

	// TestModuleDocSelfResolvesAgainstEffectiveBase is a regression for doc(”) /
	// document(”) called from within an imported/included stylesheet module whose
	// ROOT element carries xml:base. Such a module compiles its templates under the
	// FOLDED effective module base (module URI + root xml:base), so doc(”) from a
	// module template must resolve to the MODULE's OWN document — not the principal
	// stylesheet.
	//
	// The module is at mem://pkg/inc.xsl with root xml:base="sub/", so its effective
	// base is mem://pkg/sub/inc.xsl. Its module document is cached only under the
	// bare URI before this fix, so the doc(”) lookup (keyed on the template's folded
	// base URI) missed and wrongly fell back to the principal stylesheet, returning
	// MAIN-DATA instead of MODULE-DATA.
	t.Run("doc self-resolution uses the effective base", func(t *testing.T) {
		const (
			mainBase  = "mem://pkg/main.xsl"
			moduleURI = "mem://pkg/inc.xsl"
		)

		for _, fn := range []struct {
			name string
			expr string
		}{
			{"doc", `doc('')`},
			{"document", `document('')`},
		} {
			for _, link := range []struct {
				name    string
				linkElt string
			}{
				{declImport, `<xsl:import href="inc.xsl"/>`},
				{declInclude, `<xsl:include href="inc.xsl"/>`},
			} {
				t.Run(fn.name+" "+link.name, func(t *testing.T) {
					ctx := t.Context()

					module := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:d="urn:test"
  xml:base="sub/">
  <d:data>MODULE-DATA</d:data>
  <xsl:template match="item"><out><xsl:value-of select="` + fn.expr + `//*[local-name()='data']"/></out></xsl:template>
</xsl:stylesheet>`

					main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:d="urn:test">
  ` + link.linkElt + `
  <d:data>MAIN-DATA</d:data>
  <xsl:template match="/"><result><xsl:apply-templates select="//item"/></result></xsl:template>
</xsl:stylesheet>`

					resolver := &exactURIResolver{files: map[string]string{moduleURI: module}}

					doc, err := helium.NewParser().Parse(ctx, []byte(main))
					require.NoError(t, err)

					ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver).Compile(ctx, doc)
					require.NoError(t, err)

					source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>x</item></doc>`))
					require.NoError(t, err)

					result, err := ss.Transform(source).Serialize(ctx)
					require.NoError(t, err)

					require.Contains(t, result, "<out>MODULE-DATA</out>",
						"%s from a module with root xml:base must resolve to the module's own document, not the principal stylesheet", fn.expr)
					require.NotContains(t, result, "MAIN-DATA",
						"%s must not fall back to the principal stylesheet document", fn.expr)
				})
			}
		}
	})

	// TestEmbeddedModuleWrapperXMLBase covers an EMBEDDED stylesheet module (selected
	// by a fragment identifier, whose root element's parent is another element) whose
	// effective static base must fold the FULL xml:base ancestor chain — the wrapper
	// element's xml:base AND the embedded xsl:stylesheet's own xml:base — onto the
	// module document URI.
	//
	// Module at mem://pkg/emb.xsl:
	//
	//	<wrapper xml:base="outer/">
	//	  <xsl:stylesheet id="style1" xml:base="inner/" ...>
	//
	// so the embedded module's effective base is mem://pkg/outer/inner/. The root
	// use-when and every template/global in the embedded module must resolve relative
	// references against that base, not mem://pkg/inner/ (wrapper xml:base dropped)
	// nor the bare mem://pkg/emb.xsl.
	t.Run("an embedded module wrapper xml:base", func(t *testing.T) {
		const (
			mainBase    = "mem://pkg/main.xsl"
			moduleURI   = "mem://pkg/emb.xsl"
			flagURI     = "mem://pkg/outer/inner/flag.xml" // wrapper "outer/" + stylesheet "inner/" folded onto module URI
			effectiveBU = "mem://pkg/outer/inner/"
			module      = `<?xml version="1.0"?>
<wrapper xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xml:base="outer/">
  <xsl:stylesheet id="style1" version="3.0"
    xml:base="inner/"
    use-when="doc-available('flag.xml')">
    <xsl:template match="item"><base><xsl:value-of select="static-base-uri()"/></base></xsl:template>
  </xsl:stylesheet>
</wrapper>`
		)

		for _, link := range []struct {
			name    string
			linkElt string
		}{
			{declImport, `<xsl:import href="emb.xsl#style1"/>`},
			{declInclude, `<xsl:include href="emb.xsl#style1"/>`},
		} {
			t.Run(link.name, func(t *testing.T) {
				ctx := t.Context()
				main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + link.linkElt + `
  <xsl:template match="/"><result><xsl:apply-templates select="//item"/></result></xsl:template>
</xsl:stylesheet>`

				resolver := &exactURIResolver{files: map[string]string{
					moduleURI: module,
					flagURI:   `<flag/>`,
				}}

				doc, err := helium.NewParser().Parse(ctx, []byte(main))
				require.NoError(t, err)

				ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver).Compile(ctx, doc)
				require.NoError(t, err)

				source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>x</item></doc>`))
				require.NoError(t, err)

				result, err := ss.Transform(source).Serialize(ctx)
				require.NoError(t, err)

				// (a-root-use-when) The root use-when doc-available('flag.xml') must
				// resolve against the embedded module's effective base (wrapper +
				// stylesheet xml:base), so the module is INCLUDED and its template runs.
				require.True(t, resolver.askedFor(flagURI),
					"root use-when must probe the full-chain effective base %q; asked=%v", flagURI, resolver.asked)
				// (a-template-base) The template's static base URI must equal the
				// embedded module's effective base — the wrapper xml:base must not be
				// dropped.
				require.Contains(t, result, "<base>"+effectiveBU+"</base>",
					"template static base must fold the wrapper + stylesheet xml:base to %q", effectiveBU)
			})
		}
	})

	// TestEmbeddedModuleWrapperDocSelfResolves covers doc(”) / document(”) called
	// from inside an EMBEDDED stylesheet module whose effective base folds a wrapper
	// xml:base and the embedded stylesheet's own xml:base. The module's templates
	// compile under that folded effective base, so the module document must be cached
	// under that same key for doc(”)/document(”) to resolve to the module's OWN
	// document, falling back to no principal stylesheet.
	t.Run("an embedded module wrapper doc self-resolves", func(t *testing.T) {
		const (
			mainBase  = "mem://pkg/main.xsl"
			moduleURI = "mem://pkg/emb.xsl"
		)

		for _, fn := range []struct {
			name string
			expr string
		}{
			{"doc", `doc('')`},
			{"document", `document('')`},
		} {
			for _, link := range []struct {
				name    string
				linkElt string
			}{
				{declImport, `<xsl:import href="emb.xsl#style1"/>`},
				{declInclude, `<xsl:include href="emb.xsl#style1"/>`},
			} {
				t.Run(fn.name+" "+link.name, func(t *testing.T) {
					ctx := t.Context()

					module := `<?xml version="1.0"?>
<wrapper xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:d="urn:test" xml:base="outer/">
  <xsl:stylesheet id="style1" version="3.0" xml:base="inner/">
    <d:data>MODULE-DATA</d:data>
    <xsl:template match="item"><out><xsl:value-of select="` + fn.expr + `//*[local-name()='data']"/></out></xsl:template>
  </xsl:stylesheet>
</wrapper>`

					main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:d="urn:test">
  ` + link.linkElt + `
  <d:data>MAIN-DATA</d:data>
  <xsl:template match="/"><result><xsl:apply-templates select="//item"/></result></xsl:template>
</xsl:stylesheet>`

					resolver := &exactURIResolver{files: map[string]string{moduleURI: module}}

					doc, err := helium.NewParser().Parse(ctx, []byte(main))
					require.NoError(t, err)

					ss, err := xslt3.NewCompiler().BaseURI(mainBase).URIResolver(resolver).Compile(ctx, doc)
					require.NoError(t, err)

					source, err := helium.NewParser().Parse(ctx, []byte(`<doc><item>x</item></doc>`))
					require.NoError(t, err)

					result, err := ss.Transform(source).Serialize(ctx)
					require.NoError(t, err)

					require.Contains(t, result, "<out>MODULE-DATA</out>",
						"%s from an embedded module with wrapper xml:base must resolve to the module's own document", fn.expr)
					require.NotContains(t, result, "MAIN-DATA",
						"%s must not fall back to the principal stylesheet document", fn.expr)
				})
			}
		}
	})

	// TestDocumentResolutionPreservesURIBase is an end-to-end regression for the
	// base-URI corruption fixed in this change. When the stylesheet is compiled
	// with Compiler.BaseURI("mem://pkg/main.xsl"), doc('doc.xml') and
	// document('doc.xml') must reach the runtime URIResolver with the URI
	// "mem://pkg/doc.xml" — i.e. the sibling reference resolved against the FULL
	// URI base preserving scheme+authority.
	//
	// Before the fix, ec.baseDir() ran filepath.Dir over the URI base first,
	// collapsing "mem://pkg/main.xsl" to "mem:/pkg", so the resolver was instead
	// asked for "mem:/pkg/doc.xml" (host dropped). This test fails on that path.
	t.Run("document resolution preserves the URI base", func(t *testing.T) {
		const wantURI = "mem://pkg/doc.xml"

		resolver := &recordingURIResolver{files: map[string][]byte{
			wantURI: []byte(`<data v="hello"/>`),
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(docURIBaseStylesheet))
		require.NoError(t, err)

		ss, err := xslt3.NewCompiler().BaseURI("mem://pkg/main.xsl").Compile(t.Context(), doc)
		require.NoError(t, err)

		source := parseTransformSource(t)
		result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
		require.NoError(t, err)

		require.True(t, resolver.seen(wantURI),
			"runtime resolver must receive %q; got %v", wantURI, resolver.requests)
		// Confirm the host was not dropped (the pre-fix bug produced mem:/pkg/...).
		require.False(t, resolver.seen("mem:/pkg/doc.xml"),
			"resolver must not receive the host-collapsed URI; got %v", resolver.requests)

		require.Contains(t, result, "<a>hello</a>")
		require.Contains(t, result, "<b>hello</b>")
	})

	// TestDocumentResolutionLocalBaseUnchanged guards against a regression in
	// local-filesystem document resolution: a relative doc()/document() href under
	// a plain local base must still resolve against the containing directory via
	// filepath, NOT be treated as a URI.
	t.Run("document resolution leaves a local base unchanged", func(t *testing.T) {
		const wantURI = "/styles/doc.xml"

		resolver := &recordingURIResolver{files: map[string][]byte{
			wantURI: []byte(`<data v="local"/>`),
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(docURIBaseStylesheet))
		require.NoError(t, err)

		ss, err := xslt3.NewCompiler().BaseURI("/styles/main.xsl").Compile(t.Context(), doc)
		require.NoError(t, err)

		source := parseTransformSource(t)
		result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
		require.NoError(t, err)

		require.True(t, resolver.seen(wantURI),
			"runtime resolver must receive %q; got %v", wantURI, resolver.requests)
		require.Contains(t, result, "<a>local</a>")
	})
}

// docURIBaseStylesheet calls doc()/document() with a relative href so the
// runtime must resolve it against the stylesheet's base URI.
const docURIBaseStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <a><xsl:value-of select="doc('doc.xml')/data/@v"/></a>
      <b><xsl:value-of select="document('doc.xml')/data/@v"/></b>
    </out>
  </xsl:template>
</xsl:stylesheet>`

func TestUnionPattern(t *testing.T) {
	// ENG-007: an empty-body template with a union match pattern is split into
	// separate rules sharing no Body[0] identity. When a single node matches more
	// than one branch of the union (overlapping alternatives), the split branches
	// must not be flagged as conflicting with each other under
	// on-multiple-match="fail".
	t.Run("empty body no false conflict", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="root/*"/></out></xsl:template>
  <xsl:template match="node() | *"/>
</xsl:stylesheet>`)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/></root>`))
		require.NoError(t, err)

		result, err := ss.Transform(source).
			OnMultipleMatch(xslt3.OnMultipleMatchFail).
			Serialize(t.Context())
		require.NoError(t, err, "split union branches must not self-conflict")
		require.Contains(t, result, "<out/>")
	})

	// A genuine conflict between two DIFFERENT templates of equal precedence and
	// priority must still raise XTDE0540 under on-multiple-match="fail".
	t.Run("genuine conflict still fails", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:apply-templates select="root/*"/></out></xsl:template>
  <xsl:template match="a" priority="1"><x/></xsl:template>
  <xsl:template match="a" priority="1"><y/></xsl:template>
</xsl:stylesheet>`)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<root><a/></root>`))
		require.NoError(t, err)

		_, err = ss.Transform(source).
			OnMultipleMatch(xslt3.OnMultipleMatchFail).
			Serialize(t.Context())
		require.Error(t, err, "genuine conflict must raise XTDE0540")
		require.Contains(t, err.Error(), "XTDE0540")
	})
}

func TestNamespaceFixup(t *testing.T) {
	// TestNamespace2614ExcludedLiteralPrefixCanBeRebound covers W3C
	// namespace-2614. An excluded literal result element prefix is available to
	// xsl:namespace, and namespace fixup gives the element's original namespace a
	// generated prefix.
	t.Run("an excluded literal prefix can be rebound", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <p:item xmlns:p="http://p.uri/" xsl:exclude-result-prefixes="p">
      <xsl:namespace name="p">http://q.uri/</xsl:namespace>
    </p:item>
  </xsl:template>
</xsl:stylesheet>`)

		out, err := xslt3.TransformString(t.Context(), parseTransformSource(t), ss)
		require.NoError(t, err)

		doc, err := helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err)
		elem := doc.DocumentElement()
		require.Equal(t, "item", elem.LocalName())
		require.Equal(t, "http://p.uri/", elem.URI())
		require.Equal(t, "p_0", elem.Prefix())
		require.Equal(t, "http://q.uri/", namespaceURI(elem, "p"))
		require.Equal(t, "http://p.uri/", namespaceURI(elem, elem.Prefix()))
	})

	// TestNamespaceRebindOnNonExcludedLiteralResultElementRaisesXTDE0430 keeps
	// the normal conflict rule for a literal result element whose prefix was not
	// excluded from the result tree.
	t.Run("a rebind on a non-excluded literal result element raises XTDE0430", func(t *testing.T) {
		ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <p:item xmlns:p="http://p.uri/">
      <xsl:namespace name="p">http://q.uri/</xsl:namespace>
    </p:item>
  </xsl:template>
</xsl:stylesheet>`)

		_, err := xslt3.TransformString(t.Context(), parseTransformSource(t), ss)
		require.ErrorContains(t, err, "XTDE0430")
	})
}

func namespaceURI(elem *helium.Element, prefix string) string {
	for _, ns := range elem.Namespaces() {
		if ns.Prefix() == prefix {
			return ns.URI()
		}
	}
	return ""
}
