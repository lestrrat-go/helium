package xslt3_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func innerXSL(name string) string {
	return filepath.Join(heliumtest.CallerDir(0), "testdata", "fn-transform", name)
}

// fnTransformFileResolver is an explicit opt-in compile-time URIResolver that
// reads stylesheet modules from the local filesystem. fn:transform stylesheet
// loading is opt-in (no implicit os.ReadFile), so these tests supply one.
type fnTransformFileResolver struct{}

func (fnTransformFileResolver) Resolve(uri string) (io.ReadCloser, error) {
	return os.Open(uri)
}

func compileFnTransformOuter(t *testing.T, xsltSrc string) *xslt3.Stylesheet {
	t.Helper()
	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().URIResolver(fnTransformFileResolver{}).Compile(ctx, doc)
	require.NoError(t, err)
	return ss
}

// TestFnTransformStylesheetParams verifies that stylesheet-params passed
// through fn:transform() reach the inner stylesheet's xsl:param.
func TestFnTransformStylesheetParams(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'stylesheet-params': map{ QName('','greeting'): 'hello-world' },
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-param.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "hello-world")
}

// TestFnTransformStylesheetParamsNS verifies that stylesheet-params with
// namespaced QName keys are expanded to Clark notation and matched correctly.
func TestFnTransformStylesheetParamsNS(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'stylesheet-params': map{ QName('http://example.com/my','greeting'): 'ns-hello' },
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-ns-param.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "ns-hello")
}

// TestFnTransformStaticParams verifies that static-params passed through
// fn:transform() reach the inner stylesheet's static xsl:param.
func TestFnTransformStaticParams(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'static-params': map{ QName('','version'): '1.2.3' },
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-static-param.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "1.2.3")
}

// TestFnTransformInitialMode verifies that initial-mode passed through
// fn:transform() selects the correct mode in the inner stylesheet.
func TestFnTransformInitialMode(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="src" as="document-node()">
      <xsl:document><root/></xsl:document>
    </xsl:variable>
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'source-node': $src,
      'initial-mode': QName('','special'),
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-modes.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "special-mode")
	require.NotContains(t, out, "default-mode")
}

// TestFnTransformTemplateParams verifies that template-params passed
// through fn:transform() reach the initial named template's xsl:param.
func TestFnTransformTemplateParams(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'template-params': map{ QName('','color'): 'blue' },
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-template-param.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "blue")
}

// TestFnTransformTunnelParams verifies that tunnel-params passed through
// fn:transform() propagate through tunnel parameters to sub-templates.
func TestFnTransformTunnelParams(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'tunnel-params': map{ QName('','secret'): 'tunnel-value' },
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-tunnel.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "tunnel-value")
}

// TestFnTransformInitialFunction verifies that initial-function and
// function-params passed through fn:transform() invoke the correct
// xsl:function and return its result.
func TestFnTransformInitialFunction(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:f="http://example.com/fn">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'initial-function': QName('http://example.com/fn','double'),
      'function-params': [21],
      'delivery-format': 'raw'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-function.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "42")
}

// TestFnTransformBaseOutputURI verifies that base-output-uri passed through
// fn:transform() is visible via current-output-uri() in the inner stylesheet.
func TestFnTransformBaseOutputURI(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'base-output-uri': 'http://example.com/output.xml',
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-output-uri.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	cleaned := out
	if idx := strings.Index(cleaned, "?>"); idx >= 0 {
		cleaned = cleaned[idx+2:]
	}
	require.Contains(t, cleaned, "http://example.com/output.xml")
}

// memResolver serves stylesheet content from an in-memory map keyed by URI.
type memResolver struct {
	files      map[string]string
	calledWith []string
}

func (r *memResolver) Resolve(uri string) (io.ReadCloser, error) {
	r.calledWith = append(r.calledWith, uri)
	content, ok := r.files[uri]
	if !ok {
		return nil, &xpath3.XPathError{Code: "FOXT0003", Message: "not found: " + uri}
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

// TestFnTransformCustomURIScheme verifies that fn:transform() resolves
// relative stylesheet-location using proper URI resolution rather than
// filepath.Join, so custom URI schemes (e.g. mem://) are preserved.
func TestFnTransformCustomURIScheme(t *testing.T) {
	resolver := &memResolver{
		files: map[string]string{
			"mem://pkg/main.xsl": `<?xml version="1.0"?>
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
</xsl:stylesheet>`,
			"mem://pkg/inner.xsl": `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <inner>resolved</inner>
  </xsl:template>
</xsl:stylesheet>`,
		},
	}

	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(resolver.files["mem://pkg/main.xsl"]))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI("mem://pkg/main.xsl").
		URIResolver(resolver).
		Compile(ctx, doc)
	require.NoError(t, err)

	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err, "fn:transform with custom URI scheme should succeed")
	require.Contains(t, out, "resolved")

	// Verify the resolver was called with the correctly resolved URI,
	// not a filepath.Join-corrupted one like "mem:/pkg/inner.xsl".
	require.Contains(t, resolver.calledWith, "mem://pkg/inner.xsl",
		"resolver should receive properly resolved URI, got: %v", resolver.calledWith)
}

// httpResolverFunc adapts a function to the xpath3.URIResolver interface.
type httpResolverFunc func(uri string) (io.ReadCloser, error)

func (f httpResolverFunc) ResolveURI(uri string) (io.ReadCloser, error) { return f(uri) }

// TestFnTransformInheritsRuntimeResolver verifies that fn:unparsed-text
// called from inside an inner stylesheet invoked via fn:transform()
// inherits the outer Invocation's URIResolver, instead of being refused
// by secure-by-default retrieval.
func TestFnTransformInheritsRuntimeResolver(t *testing.T) {
	const dataURI = "http://example.invalid/data/hello.txt"

	var calledWith []string
	runtimeResolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		calledWith = append(calledWith, uri)
		if uri != dataURI {
			return nil, &xpath3.XPathError{Code: "FOUT1170", Message: "not found: " + uri}
		}
		return io.NopCloser(strings.NewReader("hello-from-resolver")), nil
	})

	innerXSLT := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <inner><xsl:value-of select="unparsed-text('` + dataURI + `')"/></inner>
  </xsl:template>
</xsl:stylesheet>`
	outerXSLT := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': 'mem://stylesheets/inner.xsl',
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`

	compileResolver := &memResolver{
		files: map[string]string{
			"mem://stylesheets/inner.xsl": innerXSLT,
		},
	}

	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(outerXSLT))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().URIResolver(compileResolver).Compile(ctx, doc)
	require.NoError(t, err)

	src, _ := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	out, err := ss.Transform(src).URIResolver(runtimeResolver).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "hello-from-resolver",
		"inner fn:unparsed-text should resolve via outer Invocation's URIResolver")
	require.Contains(t, calledWith, dataURI)
}
