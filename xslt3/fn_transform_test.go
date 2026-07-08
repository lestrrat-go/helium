package xslt3_test

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
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
// TestFnTransformInStylesheetBaseOutputURIUsesCallSiteBase verifies that the
// in-stylesheet fn:transform resolves a relative base-output-uri against the
// CALL SITE's effective static base URI (honoring an xml:base on the calling
// template element), not the bare module URI. The module base here is empty, so
// only the call-site xml:base can produce the expected absolute key.
func TestFnTransformInStylesheetBaseOutputURIUsesCallSiteBase(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner"/>
  <xsl:template match="/" xml:base="http://example.com/callsite/">
    <xsl:variable name="r" select="transform(map{
      'stylesheet-text': $inner,
      'source-node': .,
      'base-output-uri': 'out.xml'
    })"/>
    <result><xsl:value-of select="map:contains($r, 'http://example.com/callsite/out.xml')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	inner := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"><p>hi</p></xsl:template></xsl:stylesheet>`
	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner", xpath3.SingleString(inner)).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, ">true</result>")
}

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
    <result><xsl:value-of select="$result('http://example.com/output.xml')"/></result>
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
	// The principal result is keyed by the base output URI (not "output") when
	// base-output-uri is supplied (F&O 3.1 §14.8.3).
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

// TestFnTransformInheritsRuntimeResourceCap verifies that fn:doc /
// fn:unparsed-text inside a stylesheet invoked via fn:transform() honors the
// outer Invocation's MaxResourceBytes override rather than silently falling
// back to the default cap. The inner stylesheet reads a resolver-backed
// resource larger than the default cap: it must be refused at the default cap
// and accepted once the outer Invocation raises (or disables) the bound.
func TestFnTransformInheritsRuntimeResourceCap(t *testing.T) {
	const dataURI = "http://example.invalid/data/big.txt"
	big := strings.Repeat("z", int(xslt3.MaxResourceBytes)+(1<<10))

	runtimeResolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != dataURI {
			return nil, &xpath3.XPathError{Code: "FOUT1170", Message: "not found: " + uri}
		}
		return io.NopCloser(strings.NewReader(big)), nil
	})

	innerXSLT := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <inner><xsl:value-of select="string-length(unparsed-text('` + dataURI + `'))"/></inner>
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

	// Default cap: the inner read exceeds MaxResourceBytes and is refused.
	_, err = ss.Transform(src).URIResolver(runtimeResolver).Serialize(ctx)
	require.Error(t, err, "inner read must be refused at the default cap")

	// Raised cap: the outer Invocation's MaxResourceBytes must thread into the
	// inner transform so the same read now succeeds.
	out, err := ss.Transform(src).
		URIResolver(runtimeResolver).
		MaxResourceBytes(int64(len(big)) + 1).
		Serialize(ctx)
	require.NoError(t, err, "raised cap must thread into the inner fn:transform")
	require.Contains(t, out, strconv.Itoa(len(big)),
		"inner unparsed-text should read the full resource under the raised cap")

	// Disabled cap (negative): also threads through and lifts the bound.
	out, err = ss.Transform(src).
		URIResolver(runtimeResolver).
		MaxResourceBytes(-1).
		Serialize(ctx)
	require.NoError(t, err, "disabled cap must thread into the inner fn:transform")
	require.Contains(t, out, strconv.Itoa(len(big)))
}

// TestFnTransformInitialMatchSelectionResultDocument is a regression test for a
// panic ("assignment to entry in nil map") that occurred when fn:transform was
// called with a non-empty initial-match-selection and the invoked stylesheet
// wrote a secondary xsl:result-document. The former forked selection path
// failed to initialize resultDocItems and resultDocOutputDefs, so
// execResultDocument panicked when assigning into them. The selection case now
// routes through the normal executeTransform path, which initializes them.
func TestFnTransformInitialMatchSelectionResultDocument(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="sel" as="element()*">
      <item>alpha</item>
    </xsl:variable>
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'initial-match-selection': $sel,
      'base-output-uri': 'http://example.com/output.xml',
      'delivery-format': 'serialized'
    })"/>
    <result>
      <xsl:for-each select="map:keys($result)">
        <entry key="{.}"><xsl:value-of select="$result(.)"/></entry>
      </xsl:for-each>
    </result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-resultdoc.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	// Principal output is present (XML-escaped inside the <entry> wrapper),
	// keyed by the base output URI.
	require.Contains(t, out, `key="http://example.com/output.xml"`)
	require.Contains(t, out, "&lt;principal&gt;alpha&lt;/principal&gt;")
	// The secondary xsl:result-document appears in the result map keyed by its
	// href resolved against the base output URI (an absolute URI).
	require.Contains(t, out, `key="http://example.com/secondary.xml"`)
	require.Contains(t, out, "&lt;secondary&gt;alpha&lt;/secondary&gt;")
}

// TestFnTransformInitialMatchSelectionInitialModeText verifies that a
// fn:transform with a non-empty initial-match-selection, an initial-mode, and
// an inner stylesheet declaring <xsl:output method="text"> selects templates by
// the requested mode AND serializes through the resolved (text) output
// definition — exactly as a normal transform would. This guards against the
// forked selection execution path that skipped output-def resolution and
// initial-mode resolution.
func TestFnTransformInitialMatchSelectionInitialModeText(t *testing.T) {
	ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:template match="/">
    <xsl:variable name="sel" as="element()*">
      <item>alpha</item>
    </xsl:variable>
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'initial-match-selection': $sel,
      'initial-mode': QName('','special'),
      'delivery-format': 'serialized'
    })"/>
    <result><xsl:value-of select="$result('output')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

	src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
	out, err := ss.Transform(src).
		SetParameter("inner-loc", xpath3.SingleString(innerXSL("inner-text-modes.xsl"))).
		Serialize(t.Context())
	require.NoError(t, err)
	// The "special" mode template must be selected (not the default-mode one).
	// The whole inner result is wrapped verbatim inside the outer <result>.
	require.Contains(t, out, "<result xmlns:map=\"http://www.w3.org/2005/xpath-functions/map\">special:alpha</result>")
	require.NotContains(t, out, "default:alpha")
	// method="text" serialization: no element wrapper, no XML escaping, and no
	// XML declaration around the inner output. A forked path that left
	// resolvedOutputDef nil would XML-serialize the inner result and emit an
	// escaped declaration / angle brackets (e.g. "&lt;?xml ...").
	require.NotContains(t, out, "&lt;")
}
