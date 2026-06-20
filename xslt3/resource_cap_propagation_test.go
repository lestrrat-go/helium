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

// These regressions pin the invariant that a non-default per-resource cap
// configured on the Compiler (or effective Invocation) is threaded into every
// nested compile / stylesheet construction site, so resource reads never
// silently fall back to the default MaxResourceBytes cap.

// resolverFunc adapts a function to the xslt3.URIResolver interface used at
// compile time for xsl:import / xsl:include / simplified-stylesheet loads.
type resolverFunc func(uri string) (io.ReadCloser, error)

func (f resolverFunc) Resolve(uri string) (io.ReadCloser, error) { return f(uri) }

// capPackageResolver serves a single fixed package source for any name, plus
// satisfies any nested xsl:include the package issues via a companion
// URIResolver supplied on the Compiler.
type capPackageResolver struct {
	source  []byte
	baseURI string
}

func (r capPackageResolver) ResolvePackage(_ string, _ string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader(string(r.source))), r.baseURI, nil
}

// A simplified (literal-result-element) stylesheet must honor the Compiler's
// MaxResourceBytes cap on its own runtime resource reads. Before the fix the
// cap was dropped during compileSimplified, so an over-cap fn:doc read used the
// default 10 MiB bound instead of the configured one.
func TestSimplifiedStylesheetHonorsCompilerCap(t *testing.T) {
	t.Parallel()

	const u = "http://example.invalid/big.xml"
	body := "<root>" + strings.Repeat("a", 4096) + "</root>"

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != u {
			return nil, &xpath3.XPathError{Code: "FOUT1170", Message: "not found: " + uri}
		}
		return io.NopCloser(strings.NewReader(body)), nil
	})

	// Simplified stylesheet: root is a literal result element with xsl:version.
	simplified := `<out xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xsl:version="3.0">` +
		`<xsl:copy-of select="doc('` + u + `')"/></out>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(simplified))
	require.NoError(t, err)

	// Compile with a cap far below the resource size.
	ss, err := xslt3.NewCompiler().MaxResourceBytes(64).Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	_, err = ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"simplified stylesheet must honor the Compiler MaxResourceBytes cap")
}

// xsl:use-package bounds the package FILE read, but the cap must also flow into
// the COMPILE of that package so its own xsl:include loads are bounded. Before
// the fix the nested include fell back to the default cap.
func TestUsePackageNestedIncludeHonorsCap(t *testing.T) {
	t.Parallel()

	const includeURI = "http://example.invalid/included.xsl"
	// An included module larger than the configured cap.
	includedBody := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<!-- ` + strings.Repeat("x", 4096) + ` -->` +
		`<xsl:template name="helper"><h/></xsl:template>` +
		`</xsl:stylesheet>`

	pkgBody := `<?xml version="1.0"?>` +
		`<xsl:package name="http://example.com/pkg" package-version="1.0" version="3.0"` +
		` xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:include href="` + includeURI + `"/>` +
		`<xsl:template name="main" visibility="public"><out/></xsl:template>` +
		`</xsl:package>`

	includeResolver := resolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != includeURI {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(includedBody)), nil
	})

	mainSheet := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:use-package name="http://example.com/pkg"/>` +
		`<xsl:template match="/"><out/></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(mainSheet))
	require.NoError(t, err)

	// Cap below the included module size but above the package file size.
	_, err = xslt3.NewCompiler().
		MaxResourceBytes(512).
		PackageResolver(capPackageResolver{source: []byte(pkgBody)}).
		URIResolver(includeResolver).
		Compile(t.Context(), doc)
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"xsl:use-package nested include must honor the Compiler MaxResourceBytes cap")
}

// Runtime fn:transform bounds the immediate stylesheet read, but the cap must
// also flow into the nested COMPILE so resources loaded while compiling the
// nested stylesheet (e.g. its xsl:include) honor the Invocation cap, and the
// resulting ErrResourceTooLarge stays observable via errors.Is.
func TestFnTransformNestedCompileHonorsInvocationCap(t *testing.T) {
	t.Parallel()

	const nestedLoc = "http://example.invalid/nested.xsl"
	const includeURI = "http://example.invalid/nested-include.xsl"

	// The included module is larger than the configured cap.
	includedBody := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<!-- ` + strings.Repeat("y", 8192) + ` -->` +
		`<xsl:template name="helper"><h/></xsl:template>` +
		`</xsl:stylesheet>`

	// The nested stylesheet itself is small, so its file read passes the cap;
	// only its xsl:include should trip the bound during the nested compile.
	nestedBody := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:include href="` + includeURI + `"/>` +
		`<xsl:template match="/"><nested/></xsl:template>` +
		`</xsl:stylesheet>`

	// fn:transform resolves the stylesheet-location (and the nested module's
	// xsl:include) through the COMPILE-TIME URIResolver carried on the
	// stylesheet, so it is configured on the Compiler. The cap, however, is set
	// only on the Invocation to prove the effective Invocation cap threads into
	// the nested compile.
	resolver := resolverFunc(func(uri string) (io.ReadCloser, error) {
		switch uri {
		case nestedLoc:
			return io.NopCloser(strings.NewReader(nestedBody)), nil
		case includeURI:
			return io.NopCloser(strings.NewReader(includedBody)), nil
		default:
			return nil, os.ErrNotExist
		}
	})

	outer := `<?xml version="1.0"?>` +
		`<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
		`<xsl:template match="/"><xsl:sequence select="transform(map{` +
		`'stylesheet-location':'` + nestedLoc + `',` +
		`'source-node':.})?output"/></xsl:template>` +
		`</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(outer))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().URIResolver(resolver).Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	// Cap above the nested file but below its include; only propagation into the
	// nested compile makes the include read trip ErrResourceTooLarge.
	_, err = ss.Transform(source).
		MaxResourceBytes(2048).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"fn:transform nested-compile resources must honor the Invocation MaxResourceBytes cap")
}
