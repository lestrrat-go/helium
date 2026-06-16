package xslt3_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// osOpenResolver is an explicit opt-in compile-time URIResolver that reads
// modules straight off the local filesystem. Tests that load real files supply
// it explicitly because implicit filesystem access is no longer permitted.
type osOpenResolver struct{}

func (osOpenResolver) Resolve(uri string) (io.ReadCloser, error) {
	return os.Open(uri)
}

// fileMapResolver is an xslt3.URIResolver (method Resolve) that serves
// content from an in-memory map keyed by URI.
type fileMapResolver struct {
	files map[string]string
}

func (r fileMapResolver) Resolve(uri string) (io.ReadCloser, error) {
	content, ok := r.files[uri]
	if !ok {
		return nil, &resolverNotFoundError{uri: uri}
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

type resolverNotFoundError struct {
	uri string
}

func (e *resolverNotFoundError) Error() string { return "not found: " + e.uri }

const ddIncludedXSL = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="helper">
    <helper>included</helper>
  </xsl:template>
</xsl:stylesheet>`

func ddMainXSL(directive string) string {
	return `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  ` + directive + `
  <xsl:template match="/">
    <out><xsl:call-template name="helper"/></out>
  </xsl:template>
</xsl:stylesheet>`
}

// TestImportIncludeDefaultDeny verifies that xsl:import and xsl:include of a
// local module fail to compile when no Compiler.URIResolver is configured
// (filesystem access is opt-in), and succeed when a resolver is supplied.
func TestImportIncludeDefaultDeny(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"
	// xsl:import/include resolve href against baseURI via filepath.Join, which
	// collapses "mem://" to "mem:/"; the resolver receives that resolved form.
	const moduleURI = "mem:/stylesheets/included.xsl"

	for _, tc := range []struct {
		name      string
		directive string
	}{
		{name: "import", directive: `<xsl:import href="included.xsl"/>`},
		{name: "include", directive: `<xsl:include href="included.xsl"/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			mainSrc := ddMainXSL(tc.directive)

			// Without a resolver: default-deny.
			docDeny, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
			require.NoError(t, err)
			_, err = xslt3.NewCompiler().BaseURI(baseURI).Compile(ctx, docDeny)
			require.Error(t, err, "compile must fail without a URIResolver")
			require.Contains(t, err.Error(), "no URIResolver configured",
				"error should explain that filesystem access is opt-in")

			// With a resolver: success.
			resolver := fileMapResolver{files: map[string]string{
				moduleURI: ddIncludedXSL,
			}}
			docAllow, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
			require.NoError(t, err)
			ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, docAllow)
			require.NoError(t, err, "compile must succeed with a URIResolver")

			src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
			require.NoError(t, err)
			out, err := ss.Transform(src).Serialize(ctx)
			require.NoError(t, err)
			require.Contains(t, out, "included")
		})
	}
}

// TestFnTransformStylesheetLocationDefaultDeny verifies that fn:transform with
// a stylesheet-location denies loading when no compile-time URIResolver is
// configured, and succeeds when one is.
func TestFnTransformStylesheetLocationDefaultDeny(t *testing.T) {
	const outerURI = "mem://stylesheets/outer.xsl"
	const innerURI = "mem://stylesheets/inner.xsl"

	outerSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': 'inner.xsl',
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`

	innerSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <inner>transformed</inner>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	// Without a compile-time resolver: stylesheet-location loading is denied.
	docDeny, err := helium.NewParser().Parse(ctx, []byte(outerSrc))
	require.NoError(t, err)
	ssDeny, err := xslt3.NewCompiler().BaseURI(outerURI).Compile(ctx, docDeny)
	require.NoError(t, err, "outer stylesheet has no static module dependency; it compiles")
	srcDeny, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	_, err = ssDeny.Transform(srcDeny).Serialize(ctx)
	require.Error(t, err, "fn:transform must deny stylesheet-location without a resolver")
	require.Contains(t, err.Error(), "no URIResolver configured")

	// With a compile-time resolver: success.
	resolver := fileMapResolver{files: map[string]string{
		innerURI: innerSrc,
	}}
	docAllow, err := helium.NewParser().Parse(ctx, []byte(outerSrc))
	require.NoError(t, err)
	ssAllow, err := xslt3.NewCompiler().BaseURI(outerURI).URIResolver(resolver).Compile(ctx, docAllow)
	require.NoError(t, err)
	srcAllow, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ssAllow.Transform(srcAllow).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "transformed")
}
