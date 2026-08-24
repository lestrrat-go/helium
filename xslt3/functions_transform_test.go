package xslt3_test

import (
	"fmt"
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

func TestFnTransform(t *testing.T) {
	// TestFnTransformStylesheetParams verifies that stylesheet-params passed
	// through fn:transform() reach the inner stylesheet's xsl:param.
	t.Run("stylesheet params", func(t *testing.T) {
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
	})

	// TestFnTransformStylesheetParamsNS verifies that stylesheet-params with
	// namespaced QName keys are expanded to Clark notation and matched correctly.
	t.Run("stylesheet params NS", func(t *testing.T) {
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
	})

	// TestFnTransformStaticParams verifies that static-params passed through
	// fn:transform() reach the inner stylesheet's static xsl:param.
	t.Run("static params", func(t *testing.T) {
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
	})

	// TestFnTransformInitialMode verifies that initial-mode passed through
	// fn:transform() selects the correct mode in the inner stylesheet.
	t.Run("initial mode", func(t *testing.T) {
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
	})

	// TestFnTransformTemplateParams verifies that template-params passed
	// through fn:transform() reach the initial named template's xsl:param.
	t.Run("template params", func(t *testing.T) {
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
	})

	// TestFnTransformTunnelParams verifies that tunnel-params passed through
	// fn:transform() propagate through tunnel parameters to sub-templates.
	t.Run("tunnel params", func(t *testing.T) {
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
	})

	// TestFnTransformInitialFunction verifies that initial-function and
	// function-params passed through fn:transform() invoke the correct
	// xsl:function and return its result.
	t.Run("initial function", func(t *testing.T) {
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
	})

	// TestFnTransformBaseOutputURI verifies that base-output-uri passed through
	// fn:transform() is visible via current-output-uri() in the inner stylesheet.
	// TestFnTransformInStylesheetBaseOutputURIUsesCallSiteBase verifies that the
	// in-stylesheet fn:transform resolves a relative base-output-uri against the
	// CALL SITE's effective static base URI (honoring an xml:base on the calling
	// template element), not the bare module URI. The module base here is empty, so
	// only the call-site xml:base can produce the expected absolute key.
	t.Run("in stylesheet base output URI uses call site base", func(t *testing.T) {
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
	})

	// TestFnTransformInStylesheetSecondaryKeyAbsoluteWithoutBaseOutputURI verifies
	// that, with base-output-uri OMITTED, an in-stylesheet fn:transform still keys
	// secondary result documents by absolute URIs resolved against the call-site
	// effective static base URI (honoring xml:base). The module base is empty, so
	// only the call-site xml:base yields the expected absolute secondary key.
	t.Run("in stylesheet secondary key absolute without base output URI", func(t *testing.T) {
		ss := compileFnTransformOuter(t, `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner"/>
  <xsl:template match="/" xml:base="http://example.com/callsite/">
    <xsl:variable name="r" select="transform(map{
      'stylesheet-text': $inner,
      'source-node': .
    })"/>
    <result><xsl:value-of select="map:contains($r, 'http://example.com/callsite/s.xml')"/></result>
  </xsl:template>
</xsl:stylesheet>`)

		inner := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:template match="/"><xsl:result-document href="s.xml"><s/></xsl:result-document></xsl:template></xsl:stylesheet>`
		src, _ := helium.NewParser().Parse(t.Context(), []byte(`<dummy/>`))
		out, err := ss.Transform(src).
			SetParameter("inner", xpath3.SingleString(inner)).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, ">true</result>")
	})

	t.Run("base output URI", func(t *testing.T) {
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
	})

	// TestFnTransformCustomURIScheme verifies that fn:transform() resolves
	// relative stylesheet-location using proper URI resolution, and never
	// filepath.Join, so custom URI schemes (e.g. mem://) are preserved.
	t.Run("custom URI scheme", func(t *testing.T) {
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
	})

	// TestFnTransformInheritsRuntimeResolver verifies that fn:unparsed-text
	// called from inside an inner stylesheet invoked via fn:transform()
	// inherits the outer Invocation's URIResolver, instead of being refused
	// by secure-by-default retrieval.
	t.Run("inherits runtime resolver", func(t *testing.T) {
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
	})

	// TestFnTransformInheritsRuntimeResourceCap verifies that fn:doc /
	// fn:unparsed-text inside a stylesheet invoked via fn:transform() honors the
	// outer Invocation's MaxResourceBytes override, falling back to no
	// default cap. The inner stylesheet reads a resolver-backed
	// resource larger than the default cap: it must be refused at the default cap
	// and accepted once the outer Invocation raises (or disables) the bound.
	t.Run("inherits runtime resource cap", func(t *testing.T) {
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
	})

	// TestFnTransformInitialMatchSelectionResultDocument is a regression test for a
	// panic ("assignment to entry in nil map") that occurred when fn:transform was
	// called with a non-empty initial-match-selection and the invoked stylesheet
	// wrote a secondary xsl:result-document. The former forked selection path
	// failed to initialize resultDocItems and resultDocOutputDefs, so
	// execResultDocument panicked when assigning into them. The selection case now
	// routes through the normal executeTransform path, which initializes them.
	t.Run("initial match selection result document", func(t *testing.T) {
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
	})

	// TestFnTransformInitialMatchSelectionInitialModeText verifies that a
	// fn:transform with a non-empty initial-match-selection, an initial-mode, and
	// an inner stylesheet declaring <xsl:output method="text"> selects templates by
	// the requested mode AND serializes through the resolved (text) output
	// definition — exactly as a normal transform would. This guards against the
	// forked selection execution path that skipped output-def resolution and
	// initial-mode resolution.
	t.Run("initial match selection initial mode text", func(t *testing.T) {
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
	})

	// TestFnTransformResultMapKeying exercises the fn:transform result-map assembly
	// rules of F&O 3.1 §14.8.3: the principal-result key, the omission of a
	// principal entry when only secondary result documents are produced, and the
	// resolution of secondary result-document keys against the base output URI.
	t.Run("result map keying", func(t *testing.T) {
		base := "http://www.w3.org/fots/fn/transform/output-doc.xml"
		multiVars := xslXMLVars(multiResultDocXSL, multiResultDocXML)

		testcases := []struct {
			name string
			expr string
		}{
			{
				// fn-transform-13 / 33 / 44: only secondary result documents, so the
				// original map has exactly the three secondary entries — no principal
				// entry under "output" nor under the base output URI.
				name: "no-principal-entry-when-only-secondary-docs",
				expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"base-output-uri":"` + base + `"})
			return map:size($r)=3 and not(map:contains($r,"output")) and not(map:contains($r,"` + base + `"))`,
			},
			{
				// fn-transform-13a / 37: secondary keys are the href resolved against
				// base-output-uri (an absolute URI), not the relative href as written.
				name: "secondary-keys-resolved-absolute",
				expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"base-output-uri":"` + base + `"})
			return contains(string-join(map:keys($r)),"www.w3.org/fots/fn/transform/section2.html")`,
			},
			{
				// fn-transform-33: same, serialized delivery. Assert on the original
				// map so a stray principal entry keyed by base-output-uri would fail.
				name: "serialized-no-principal-entry",
				expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"base-output-uri":"` + base + `","delivery-format":"serialized"})
			return map:size($r)=3 and not(map:contains($r,"output")) and not(map:contains($r,"` + base + `")) and contains(string-join(map:keys($r)),"section2")`,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				require.Equal(t, wantTrue, transformBool(t, tc.expr, multiVars))
			})
		}
	})

	// TestFnTransformPrincipalKey verifies that the principal result is keyed by the
	// base output URI when one is supplied (fn-transform-16 / 17 / 35 / 45 / 88),
	// and by "output" otherwise.
	t.Run("principal key", func(t *testing.T) {
		base := "http://www.w3.org/fots/fn/transform/output-doc.xml"
		vars := xslXMLVars(principalOnlyXSL, `<doc/>`)

		testcases := []struct {
			name string
			expr string
		}{
			{
				// fn-transform-17: principal keyed by base-output-uri, not "output".
				name: "document-principal-keyed-by-base-output-uri",
				expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"delivery-format":"document","base-output-uri":"` + base + `"})
			return not(map:contains($r,"output")) and map:contains($r,"` + base + `") and $r("` + base + `") instance of node()`,
			},
			{
				// fn-transform-35 / 45: serialized principal keyed by base-output-uri.
				name: "serialized-principal-keyed-by-base-output-uri",
				expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"delivery-format":"serialized","base-output-uri":"` + base + `"})
			return map:size($r)=1 and not(map:contains($r,"output")) and $r("` + base + `") instance of xs:string`,
			},
			{
				// No base-output-uri: principal keyed by the literal "output".
				name: "principal-keyed-by-output-without-base-output-uri",
				expr: `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml($xml),"delivery-format":"document"})
			return map:contains($r,"output") and $r("output") instance of node()`,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				require.Equal(t, wantTrue, transformBool(t, tc.expr, vars))
			})
		}
	})

	// TestFnTransformRawPrincipalKeyedByBaseOutputURI mirrors fn-transform-88: raw
	// delivery with a base-output-uri keys the principal result by that URI.
	t.Run("raw principal keyed by base output URI", func(t *testing.T) {
		xsl := `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0' default-mode='x'>
<xsl:template match='/' mode='#unnamed'>WRONG</xsl:template>
<xsl:template match='/' mode='x'>RIGHT</xsl:template>
</xsl:stylesheet>`
		expr := `let $r := transform(map{"stylesheet-text":$xsl,"delivery-format":"raw","base-output-uri":"http://example.com/","source-node":parse-xml('<a><b>89</b></a>')})
	return string($r("http://example.com/")) = 'RIGHT'`
		require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
	})

	// TestFnTransformGlobalContextItem covers fn-transform-82b/82c/82d: the initial
	// match selection is the source-node itself (element vs document), while the
	// global context item defaults to the root of the source-node unless an explicit
	// global-context-item option overrides it.
	t.Run("global context item", func(t *testing.T) {
		vars := xslXMLVars(globalContextXSL, "")

		testcases := []struct {
			name string
			expr string
		}{
			{
				// 82b: source-node is an element, no global-context-item. The global
				// context item defaults to the document root (root-is-doc=true) while
				// the template matches the element (this-is-doc=false, name="").
				name: "82b-element-source-default-gci-is-document",
				expr: `let $in := parse-xml("<dummy/>"),
			$r := transform(map{"source-node":$in/*,"stylesheet-text":$xsl,"xslt-version":3.0})?output
			return $r/out/@root-is-doc="true" and $r/out/@this-is-doc="false" and $r/out=""`,
			},
			{
				// 82c: source-node is a document, global-context-item is the element.
				// The global context item is the element (root-is-doc=false,
				// name="dummy"); the template matches the document (this-is-doc=true).
				name: "82c-document-source-element-gci",
				expr: `let $in := parse-xml("<dummy/>"),
			$r := transform(map{"source-node":$in,"global-context-item":$in/*,"stylesheet-text":$xsl,"xslt-version":3.0})?output
			return $r/out/@root-is-doc="false" and $r/out/@this-is-doc="true" and $r/out="dummy"`,
			},
			{
				// 82d: source-node is an element, global-context-item is the document.
				// The global context item is the document (root-is-doc=true, name="");
				// the template matches the element (this-is-doc=false).
				name: "82d-element-source-document-gci",
				expr: `let $in := parse-xml("<dummy/>"),
			$r := transform(map{"source-node":$in/*,"global-context-item":$in,"stylesheet-text":$xsl,"xslt-version":3.0})?output
			return $r/out/@root-is-doc="true" and $r/out/@this-is-doc="false" and $r/out=""`,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				require.Equal(t, wantTrue, transformBool(t, tc.expr, vars))
			})
		}
	})

	// TestFnTransformNestedResultDocKey verifies that a nested xsl:result-document's
	// key is its href resolved against the ENCLOSING result document's dynamic
	// output URI, not the top-level base output URI.
	t.Run("nested result doc key", func(t *testing.T) {
		xsl := `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'>
<xsl:result-document href='outer/index.html'><p>outer</p>
<xsl:result-document href='inner.html'><p>inner</p></xsl:result-document>
</xsl:result-document>
</xsl:template>
</xsl:stylesheet>`
		// The inner href resolves against the outer document's URI
		// (http://example.com/base/outer/index.html), yielding
		// http://example.com/base/outer/inner.html — NOT http://example.com/base/inner.html.
		expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml('<doc/>'),"base-output-uri":"http://example.com/base/main.xml"})
	return map:contains($r,"http://example.com/base/outer/index.html")
	   and map:contains($r,"http://example.com/base/outer/inner.html")
	   and not(map:contains($r,"http://example.com/base/inner.html"))`
		require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
	})

	// TestFnTransformRelativeBaseOutputURIResolvedAgainstStaticBase verifies that a
	// RELATIVE base-output-uri is resolved against the fn:transform call's static
	// base URI (F&O 3.1 §14.8), so both the principal and secondary result-map keys
	// are absolute URIs.
	t.Run("relative base output URI resolved against static base", func(t *testing.T) {
		xsl := `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'><p>principal</p><xsl:result-document href='s.xml'><s/></xsl:result-document></xsl:template>
</xsl:stylesheet>`
		// The static base URI is supplied to the standalone fn:transform via
		// WithTransformBaseURI; the relative base-output-uri "out/doc.xml" resolves
		// against it to http://example.com/base/out/doc.xml.
		fns := transformFnsWith(xslt3.WithTransformBaseURI("http://example.com/base/main.xsl"))
		expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml('<doc/>'),"base-output-uri":"out/doc.xml"})
	return map:contains($r,"http://example.com/base/out/doc.xml")
	   and map:contains($r,"http://example.com/base/out/s.xml")
	   and $r("http://example.com/base/out/doc.xml")//p = "principal"
	   and $r("http://example.com/base/out/s.xml")//s`
		got, err := evalTransform(t, expr, nil, xslXMLVars(xsl, ""), fns)
		require.NoError(t, err)
		require.Equal(t, wantTrue, got)
	})

	// TestFnTransformSecondaryKeyAbsoluteWithoutBaseOutputURI verifies that when
	// base-output-uri is OMITTED, secondary result-document keys are still absolute
	// URIs (resolved against the call's static base, here WithTransformBaseURI),
	// while the principal key stays the literal "output" (F&O 3.1 §14.8).
	t.Run("secondary key absolute without base output URI", func(t *testing.T) {
		xsl := `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'><p>principal</p><xsl:result-document href='s.xml'><s/></xsl:result-document></xsl:template>
</xsl:stylesheet>`
		fns := transformFnsWith(xslt3.WithTransformBaseURI("http://example.com/call/main.xsl"))
		expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml('<doc/>')})
	return map:contains($r,"output")
	   and map:contains($r,"http://example.com/call/s.xml")
	   and not(map:contains($r,"s.xml"))`
		got, err := evalTransform(t, expr, nil, xslXMLVars(xsl, ""), fns)
		require.NoError(t, err)
		require.Equal(t, wantTrue, got)
	})

	// TestFnTransformNestedCollidingHrefKeys verifies that two nested
	// xsl:result-documents writing the SAME relative href under DIFFERENT enclosing
	// output URIs resolve to distinct absolute URIs and both survive in the result
	// map (no storage overwrite), each with its own content.
	t.Run("nested colliding href keys", func(t *testing.T) {
		xsl := `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'>
<xsl:result-document href='dir1/index.html'><xsl:result-document href='x.xml'><v>one</v></xsl:result-document></xsl:result-document>
<xsl:result-document href='dir2/index.html'><xsl:result-document href='x.xml'><v>two</v></xsl:result-document></xsl:result-document>
</xsl:template>
</xsl:stylesheet>`
		expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml('<doc/>'),"base-output-uri":"http://example.com/base/main.xml"})
	return map:contains($r,"http://example.com/base/dir1/x.xml")
	   and map:contains($r,"http://example.com/base/dir2/x.xml")
	   and $r("http://example.com/base/dir1/x.xml")//v = "one"
	   and $r("http://example.com/base/dir2/x.xml")//v = "two"`
		require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
	})

	// TestFnTransformSameAbsoluteURITwiceRaisesXTDE1490 confirms that two
	// result-documents whose hrefs resolve to the SAME absolute output URI still
	// collide (XTDE1490), even through the resolved-URI storage keying.
	t.Run("same absolute URI twice raises XTDE1490", func(t *testing.T) {
		xsl := `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'>
<xsl:result-document href='sub/x.xml'><a/></xsl:result-document>
<xsl:result-document href='sub/nested/../x.xml'><b/></xsl:result-document>
</xsl:template>
</xsl:stylesheet>`
		expr := `fn:transform(map{"stylesheet-text":$xsl,"source-node":parse-xml('<doc/>'),"base-output-uri":"http://example.com/base/main.xml"})`
		err := transformErr(t, expr, xslXMLVars(xsl, ""))
		require.Error(t, err)
		require.Contains(t, err.Error(), "XTDE1490")
	})

	// TestFnTransformSerializedTextPreservesTrailingNewline guards that method="text"
	// serialized delivery keeps a legitimate trailing newline.
	t.Run("serialized text preserves trailing newline", func(t *testing.T) {
		xsl := "<xsl:stylesheet version=\"3.0\" xmlns:xsl=\"http://www.w3.org/1999/XSL/Transform\">" +
			"<xsl:output method=\"text\"/>" +
			"<xsl:template name='main'>a\nb\n</xsl:template></xsl:stylesheet>"
		expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"delivery-format":"serialized"})?output
	return $r = concat("a", codepoints-to-string(10), "b", codepoints-to-string(10))`
		require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
	})

	t.Run("serialized XML preserves trailing content newline", func(t *testing.T) {
		xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">` +
			`<xsl:output method="xml" encoding="iso-8859-1" omit-xml-declaration="yes"/>` +
			`<xsl:template name="main"><out/><xsl:text>&#10;</xsl:text></xsl:template>` +
			`</xsl:stylesheet>`
		expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"delivery-format":"serialized"})?output
	return $r = concat("<out/>", codepoints-to-string(10))`
		require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
	})

	// TestFnTransformGlobalContextItemTypeCheck verifies that an explicit
	// fn:transform global-context-item (an item(), here an atomic value) is what
	// gets type-checked against xsl:global-context-item/@as — not the source
	// document. An integer matches as="xs:integer"; a string is an XTTE0590 error.
	t.Run("global context item type check", func(t *testing.T) {
		xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:global-context-item as="xs:integer"/>
<xsl:template name="main"><out>ok</out></xsl:template>
</xsl:stylesheet>`
		vars := xslXMLVars(xsl, "")

		// An integer global-context-item matches as="xs:integer".
		okExpr := `let $r := transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"global-context-item":42})?output
	return $r/out = "ok"`
		require.Equal(t, wantTrue, transformBool(t, okExpr, vars))

		// A string global-context-item does not match as="xs:integer" → XTTE0590.
		badExpr := `transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"global-context-item":"hello"})?output`
		_, err := evalTransform(t, badExpr, nil, vars, transformFns())
		require.Error(t, err)
		require.Contains(t, err.Error(), "XTTE0590")
	})

	// TestFnTransformGlobalContextItemCardinality covers F&O 3.1 §14.8: the
	// global-context-item option has required type item() — a present-but-empty or
	// multi-item value is an XPTY0004 type error, never silently absent/truncated.
	t.Run("global context item cardinality", func(t *testing.T) {
		xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:template name="main"><out>ok</out></xsl:template>
</xsl:stylesheet>`
		vars := xslXMLVars(xsl, "")

		for _, tc := range []struct{ name, gci string }{
			{"empty", "()"},
			{"multi", "(1,2)"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				expr := `transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"global-context-item":` + tc.gci + `})?output`
				err := transformErr(t, expr, vars)
				require.Error(t, err)
				require.Contains(t, err.Error(), "XPTY0004")
			})
		}
	})

	// TestFnTransformGlobalContextItemUse covers the xsl:global-context-item @use
	// modes crossed with the fn:transform source-node / global-context-item options
	// (XSLT 3.0 §5.4.3.1 + F&O 3.1 §14.8).
	t.Run("global context item use", func(t *testing.T) {
		// A global variable that captures "." and a named entry template.
		tmpl := `<xsl:variable name="v" select="."/>
<xsl:template name="main"><out><xsl:value-of select="$v"/></out></xsl:template>`
		absentXSL := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:global-context-item use="absent"/>` + tmpl + `</xsl:stylesheet>`
		requiredXSL := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:global-context-item use="required"/><xsl:template name="main"><out>ok</out></xsl:template></xsl:stylesheet>`

		t.Run("absent-supplied-gci-ignored-global-dot-is-absent", func(t *testing.T) {
			// use="absent": a supplied global-context-item is ignored, so the global
			// "." reference has no context item → XPDY0002.
			expr := `transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"global-context-item":42})?output`
			err := transformErr(t, expr, xslXMLVars(absentXSL, ""))
			require.Error(t, err)
			require.Contains(t, err.Error(), "XPDY0002")
		})

		t.Run("absent-with-source-node-global-dot-is-absent", func(t *testing.T) {
			expr := `transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"source-node":parse-xml("<a/>")})?output`
			err := transformErr(t, expr, xslXMLVars(absentXSL, ""))
			require.Error(t, err)
			require.Contains(t, err.Error(), "XPDY0002")
		})

		t.Run("absent-with-source-node-match-template-applies", func(t *testing.T) {
			// use="absent" makes only the GLOBAL CONTEXT ITEM absent; the source-node
			// is still the initial match selection, so a match template applies
			// normally (no XTDE0040). No global "." reference here.
			matchXSL := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:global-context-item use="absent"/><xsl:template match="a"><out>matched</out></xsl:template></xsl:stylesheet>`
			expr := `transform(map{"stylesheet-text":$xsl,"source-node":parse-xml("<a/>")})?output/out = "matched"`
			require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(matchXSL, "")))
		})

		t.Run("absent-with-source-node-and-match-template-global-dot-XPDY0002", func(t *testing.T) {
			// Even with a source-node present and a match template, use="absent" keeps
			// the global context item absent, so a global "." reference still fails.
			matchXSL := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:global-context-item use="absent"/><xsl:variable name="v" select="."/><xsl:template match="a"><out><xsl:value-of select="$v"/></out></xsl:template></xsl:stylesheet>`
			expr := `transform(map{"stylesheet-text":$xsl,"source-node":parse-xml("<a/>")})?output`
			err := transformErr(t, expr, xslXMLVars(matchXSL, ""))
			require.Error(t, err)
			require.Contains(t, err.Error(), "XPDY0002")
		})

		t.Run("required-no-source-no-gci-raises-XTDE3086", func(t *testing.T) {
			expr := `transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main")})?output`
			err := transformErr(t, expr, xslXMLVars(requiredXSL, ""))
			require.Error(t, err)
			require.Contains(t, err.Error(), "XTDE3086")
		})

		t.Run("required-with-source-node-ok", func(t *testing.T) {
			expr := `transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"source-node":parse-xml("<a/>")})?output/out = "ok"`
			require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(requiredXSL, "")))
		})

		t.Run("required-with-explicit-gci-ok", func(t *testing.T) {
			expr := `transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"global-context-item":42})?output/out = "ok"`
			require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(requiredXSL, "")))
		})
	})

	// TestFnTransformGlobalContextItemNoLeak confirms an explicit non-node
	// global-context-item is visible to global variables ("." = the item) but does
	// NOT leak into template execution, where "." remains the matched source node.
	t.Run("global context item no leak", func(t *testing.T) {
		xsl := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:variable name="v" select="."/>
<xsl:template match="a"><out gv="{$v}" ctx="{name(.)}"/></xsl:template>
</xsl:stylesheet>`
		// gci=42 (atomic) → $v = 42; the template matches <a>, so name(.) = "a".
		expr := `let $r := transform(map{"stylesheet-text":$xsl,"source-node":parse-xml("<a/>")/*,"global-context-item":42})?output
	return $r/out/@gv = "42" and $r/out/@ctx = "a"`
		require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
	})

	// TestFnTransformSerializedNoTrailingNewline mirrors fn-transform-err-8: a
	// serialized principal result must not carry a spurious trailing newline, so an
	// ends-with test against the closing tag succeeds.
	t.Run("serialized no trailing newline", func(t *testing.T) {
		xsl := `<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<xsl:template name='main'><x>done</x></xsl:template>
</xsl:stylesheet>`
		expr := `let $r := fn:transform(map{"stylesheet-text":$xsl,"initial-template":QName("","main"),"base-output-uri":"fn/transform/output.xml","delivery-format":"serialized"})?*
	return ends-with($r, "</x>")`
		require.Equal(t, wantTrue, transformBool(t, expr, xslXMLVars(xsl, "")))
	})

	// TestTransformPostProcess drives the fn:transform post-process callback for all
	// three delivery formats, mirroring QT3 cases fn-transform-79 (document),
	// fn-transform-80 (serialized), and fn-transform-81 (raw). The callback receives
	// the result value and its return replaces the delivered output.
	t.Run("the post-process option", func(t *testing.T) {
		sourceDoc := helium.NewDefaultDocument()

		// fn-transform-79: delivery-format document. post-process navigates the
		// result document node and returns <b>89</b>, deep-equal to parse-xml('<b>89</b>')/*.
		t.Run("Document", func(t *testing.T) {
			expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='1.0'><xsl:template match='/'><out><xsl:copy-of select='.'/></out></xsl:template></xsl:stylesheet>" return
			let $expected := parse-xml('<b>89</b>')/* return
			let $trans-result := transform(map{"stylesheet-text":$xsl,
				"delivery-format":"document",
				"source-node": parse-xml('<a><b>89</b></a>'),
				"post-process": function($uri, $doc) { $doc/out/a/b }
				}) return
			deep-equal($trans-result("output"), $expected)`
			out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
			require.NoError(t, err)
			require.Equal(t, "true", out)
		})

		// fn-transform-80: delivery-format serialized. post-process truncates the
		// serialized string.
		t.Run("Serialized", func(t *testing.T) {
			expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='1.0'><xsl:template match='/'><out><xsl:copy-of select='.'/></out></xsl:template></xsl:stylesheet>" return
			let $trans-result := transform(map{"stylesheet-text":$xsl,
				"delivery-format":"serialized",
				"serialization-params": map { "method":"xml", "omit-xml-declaration":true(), "indent":false() },
				"source-node": parse-xml('<a><b>89</b></a>'),
				"post-process": function($uri, $out) { concat(substring($out, 1, 12), '...') }
				}) return
			deep-equal($trans-result("output"), "<out><a><b>8...")`
			out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
			require.NoError(t, err)
			require.Equal(t, "true", out)
		})

		// fn-transform-81: delivery-format raw. post-process arithmetic on the raw
		// atomic value (42 + 3 = 45).
		t.Run("Raw", func(t *testing.T) {
			expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='1.0'><xsl:template match='/'><xsl:sequence select='42'/></xsl:template></xsl:stylesheet>" return
			let $trans-result := transform(map{"stylesheet-text":$xsl,
				"delivery-format":"raw",
				"source-node": parse-xml('<a><b>89</b></a>'),
				"post-process": function($uri, $out) { $out + 3 }
				}) return
			deep-equal($trans-result("output"), 45)`
			out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
			require.NoError(t, err)
			require.Equal(t, "true", out)
		})
	})

	// TestTransformStylesheetNodeElement drives fn:transform with a stylesheet-node
	// that is a bare (simplified / literal-result) element, and no document
	// node, mirroring QT3 cases fn-transform-7d and fn-transform-7e. The element is
	// used as the stylesheet root even when it is not its owner document's document
	// element.
	t.Run("a stylesheet-node element argument", func(t *testing.T) {
		sourceDoc := helium.NewDefaultDocument()

		// fn-transform-7d: the simplified stylesheet <out> is the SECOND child of a
		// document fragment (the first is <noise/>); the source is a separate document.
		t.Run("FragmentElement", func(t *testing.T) {
			expr := `let $xsl := "<noise/>
			<out xmlns:xsl='http://www.w3.org/1999/XSL/Transform' xsl:version='2.0'>
			 <xsl:value-of select='.' />
			</out>" return
			transform(map{"stylesheet-node":parse-xml-fragment($xsl)/out, "source-node":parse-xml("<doc>this</doc>") })?output//out = 'this'`
			out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
			require.NoError(t, err)
			require.Equal(t, "true", out)
		})

		// fn-transform-7e: stylesheet and source are nodes in the SAME fragment
		// document; the stylesheet-node is the <out> element sibling of <doc>.
		t.Run("SameDocumentElement", func(t *testing.T) {
			expr := `let $src := parse-xml-fragment("<doc>this</doc>
			<out xmlns:xsl='http://www.w3.org/1999/XSL/Transform' xsl:version='2.0'>
			 <xsl:value-of select='/doc' />
			</out>") return
			transform(map{"stylesheet-node":$src/out, "source-node":$src })?output//out = 'this'`
			out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
			require.NoError(t, err)
			require.Equal(t, "true", out)
		})
	})

	// TestTransformSerializationParamsQName drives fn:transform with a serialized
	// delivery format and a serialization-params map carrying QName-valued
	// (cdata-section-elements, suppress-indentation) and map-valued
	// (use-character-maps) parameters, mirroring QT3 cases fn-transform-65,
	// fn-transform-66, and fn-transform-67. Each parameter must be applied to the
	// serialized principal result, unioned (cdata-section-elements /
	// suppress-indentation) or merged with override precedence (use-character-maps)
	// over the stylesheet's own xsl:output values.
	t.Run("QName serialization parameters", func(t *testing.T) {
		sourceDoc := helium.NewDefaultDocument()

		// fn-transform-65: serialization-params cdata-section-elements. The stylesheet
		// declares cdata-section-elements='b my:b'; the param adds (my:c, c). The
		// serialized output wraps the content of b, my:b, my:c, and c in CDATA, but
		// leaves d as ordinary text.
		t.Run("CDATASectionElements", func(t *testing.T) {
			expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
            xmlns:xs='http://www.w3.org/2001/XMLSchema'
            xmlns:my='http://www.w3.org/fots/fn/transform/myfunctions' version='2.0'>
            <xsl:output cdata-section-elements='b my:b'/>
            <xsl:template name='main'>
              <my:a>
                <my:b>green</my:b>
                <my:c>blue</my:c>
                <b>red</b>
                <c>pink</c>
                <d>black</d>
              </my:a>
            </xsl:template>
        </xsl:stylesheet>" return
            let $result := transform(map{"stylesheet-text":$xsl,
            "initial-template": fn:QName('','main'),
            "delivery-format" : "serialized",
            "serialization-params": map{'cdata-section-elements': (QName('http://www.w3.org/fots/fn/transform/myfunctions','c'), QName('', 'c'))}
            }) return
            ($result("output") instance of xs:string
             and contains($result("output"), "[CDATA[green]]")
             and contains($result("output"), "[CDATA[blue]]")
             and contains($result("output"), "[CDATA[red]]")
             and contains($result("output"), "[CDATA[pink]]")
             and contains($result("output"), "<d>black</d>"))`
			out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
			require.NoError(t, err)
			require.Equal(t, "true", out)
		})

		// fn-transform-66: serialization-params suppress-indentation. The stylesheet
		// declares indent='yes' suppress-indentation='b my:b'; the param adds
		// (my:c, c). The serialized output keeps b, my:b, my:c, and c un-indented but
		// indents d.
		t.Run("SuppressIndentation", func(t *testing.T) {
			expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
            xmlns:xs='http://www.w3.org/2001/XMLSchema'
            xmlns:my='http://www.w3.org/fots/fn/transform/myfunctions' version='3.0'>
            <xsl:output indent='yes' suppress-indentation='b my:b'/>
            <xsl:template name='main'>
              <my:a>
                <my:b><t>green</t></my:b>
                <my:c><t>blue</t></my:c>
                <b><t>red</t></b>
                <c><t>pink</t></c>
                <d><t>black</t></d>
              </my:a>
            </xsl:template>
        </xsl:stylesheet>" return
            let $result := transform(map{"stylesheet-text":$xsl,
            "initial-template": fn:QName('','main'),
            "delivery-format" : "serialized",
            "serialization-params": map{'suppress-indentation': (QName('http://www.w3.org/fots/fn/transform/myfunctions','c'), QName('', 'c'))}
            }) return
            ($result("output") instance of xs:string
             and contains($result("output"), "><t>green</t><")
             and contains($result("output"), "><t>blue</t><")
             and contains($result("output"), "><t>red</t><")
             and contains($result("output"), "><t>pink</t><")
             and matches($result("output"), ">\s+<t>black</t>\s+<"))`
			out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
			require.NoError(t, err)
			require.Equal(t, "true", out)
		})

		// fn-transform-67: serialization-params use-character-maps. The stylesheet's
		// map-one maps '-'→'(hyphen)' and '*'→'(asterisk)'; the param map remaps
		// '*'→'(star)'. The serialized output merges the two, with the param winning
		// for '*'.
		t.Run("UseCharacterMaps", func(t *testing.T) {
			expr := `let $xsl := "<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
            xmlns:xs='http://www.w3.org/2001/XMLSchema'
            xmlns:my='http://www.w3.org/fots/fn/transform/myfunctions' version='2.0'>
            <xsl:output use-character-maps='map-one'/>
            <xsl:character-map name='map-one'>
              <xsl:output-character character='-' string='(hyphen)'/>
              <xsl:output-character character='*' string='(asterisk)'/>
            </xsl:character-map>
            <xsl:template name='main'>
              <out>a-b*c</out>
            </xsl:template>
        </xsl:stylesheet>" return
            let $result := transform(map{"stylesheet-text":$xsl,
            "initial-template": fn:QName('','main'),
            "delivery-format" : "serialized",
            "serialization-params": map{'use-character-maps': map{'*':'(star)'}}
            }) return
            ($result("output") instance of xs:string
             and contains($result("output"), ">a(hyphen)b(star)c</out>"))`
			out, err := evalTransform(t, expr, sourceDoc, nil, transformFns())
			require.NoError(t, err)
			require.Equal(t, "true", out)
		})
	})

	// TestTransformFunctionStylesheetBaseURI verifies that fn:transform honors the
	// stylesheet-base-uri option (and a stylesheet-node's own document base URI)
	// when resolving a relative xsl:include inside a stylesheet-text/-node.
	t.Run("the stylesheet base URI", func(t *testing.T) {
		sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<data>hi</data>`))
		require.NoError(t, err)

		// The include href "sub/inc.xsl" resolves against the base
		// "http://example.com/base/main.xsl" to this URI.
		resolver := mapURIResolver{files: map[string]string{
			"http://example.com/base/sub/inc.xsl": includedTemplateStylesheet,
		}}
		fnsWith := func(opts ...xslt3.TransformOption) map[xpath3.QualifiedName]xpath3.Function {
			return map[xpath3.QualifiedName]xpath3.Function{
				{URI: xpath3.NSFn, Name: "transform"}: xslt3.TransformFunction(
					append([]xslt3.TransformOption{xslt3.WithTransformURIResolver(resolver)}, opts...)...),
			}
		}

		// With no stylesheet-base-uri and no call static base URI, the relative
		// include href gets no base applied, so the compiler attempts to load the
		// RAW relative "sub/inc.xsl". A recording resolver proves that unbased URI is
		// what is attempted (not a resolver-file-absence for some based URI) and
		// that the transform then fails — the genuine no-usable-base path that
		// fn-transform-err-9 exercises. (helium reports XTSE0165 when NO resolver at
		// all is configured; here a resolver is present but the unbased relative URI
		// is unresolvable regardless.)
		t.Run("StylesheetTextNoBaseAttemptsRawRelative", func(t *testing.T) {
			rec := &recordingRejectResolver{}
			fns := transformFnsWith(xslt3.WithTransformURIResolver(rec))
			_, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(includingStylesheet)},
				fns,
			)
			require.Error(t, err)
			// The unbased relative href is what the compiler tried to load — never an
			// absolute/based URI. This distinguishes the no-base path from mere
			// resolver-file-absence.
			require.Equal(t, []string{"sub/inc.xsl"}, rec.requested)
		})

		// With no resolver at all, the relative include cannot be loaded and the
		// nested compile fails with the specific XTSE0165 (opt-in resolver) error.
		t.Run("StylesheetTextNoResolverXTSE0165", func(t *testing.T) {
			_, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(includingStylesheet)},
				transformFns(),
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), "XTSE0165")
		})

		// stylesheet-base-uri (absolute) supplies the base for the inline text so the
		// relative include resolves.
		t.Run("StylesheetTextAbsoluteBaseURI", func(t *testing.T) {
			out, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'stylesheet-base-uri': $base, 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{
					"ss":   xpath3.SingleString(includingStylesheet),
					"base": xpath3.SingleString("http://example.com/base/main.xsl"),
				},
				fnsWith(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<out>data</out>")
		})

		// A relative stylesheet-base-uri is resolved against the call's static base
		// URI (WithTransformBaseURI) before it is used as the include base.
		t.Run("StylesheetTextRelativeBaseURI", func(t *testing.T) {
			out, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'stylesheet-base-uri': $base, 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{
					"ss":   xpath3.SingleString(includingStylesheet),
					"base": xpath3.SingleString("base/main.xsl"),
				},
				fnsWith(xslt3.WithTransformBaseURI("http://example.com/root.xsl")),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<out>data</out>")
		})

		// stylesheet-node defaults its base URI from the node's own document base
		// URI (fn-transform-24 semantics): a relative include resolves without any
		// stylesheet-base-uri option.
		t.Run("StylesheetNodeDocBaseDefault", func(t *testing.T) {
			ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(includingStylesheet))
			require.NoError(t, err)
			ssDoc.SetURL("http://example.com/base/main.xsl")
			out, err := evalTransform(t,
				`transform(map{'stylesheet-node': $ssnode, 'source-node': ., 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ssnode": xpath3.ItemSlice{xpath3.NodeItem{Node: ssDoc}}},
				fnsWith(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<out>data</out>")
		})

		// stylesheet-base-uri overrides the stylesheet-node's own document base URI
		// (fn-transform-23 semantics).
		t.Run("StylesheetNodeBaseURIOption", func(t *testing.T) {
			ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(includingStylesheet))
			require.NoError(t, err)
			ssDoc.SetURL("http://elsewhere.example.org/wrong/main.xsl")
			out, err := evalTransform(t,
				`transform(map{'stylesheet-node': $ssnode, 'source-node': ., 'stylesheet-base-uri': $base, 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{
					"ssnode": xpath3.ItemSlice{xpath3.NodeItem{Node: ssDoc}},
					"base":   xpath3.SingleString("http://example.com/base/main.xsl"),
				},
				fnsWith(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<out>data</out>")
		})
	})

	// TestTransformFunctionStandalone drives xslt3.TransformFunction as a
	// registered xpath3 function through a real xpath3.Evaluator — the standalone
	// path that the QT3 harness exercises (no outer running stylesheet).
	t.Run("a standalone invocation", func(t *testing.T) {
		sourceDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<data>hi</data>`))
		require.NoError(t, err)

		// Baseline: the bare xpath3 stub (no injected fn:transform) is
		// unimplemented, so the standalone path needs the xslt3 injection.
		t.Run("StubUnimplemented", func(t *testing.T) {
			_, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(innerTransformStylesheet)},
				nil,
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not implemented")
		})

		t.Run("StylesheetText", func(t *testing.T) {
			out, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(innerTransformStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<out>data</out>")
		})

		t.Run("StylesheetNode", func(t *testing.T) {
			ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(innerTransformStylesheet))
			require.NoError(t, err)
			out, err := evalTransform(t,
				`transform(map{'stylesheet-node': $ssnode, 'source-node': ., 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ssnode": xpath3.ItemSlice{xpath3.NodeItem{Node: ssDoc}}},
				transformFns(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<out>data</out>")
		})

		t.Run("InitialTemplate", func(t *testing.T) {
			out, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'initial-template': QName('','go'), 'delivery-format': 'serialized'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(innerTransformStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "<out>named-template</out>")
		})

		t.Run("SecondarySerializationError", func(t *testing.T) {
			const stylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template name="go">
    <xsl:result-document href="secondary">
      <xsl:value-of select="codepoints-to-string(1)"/>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>`
			_, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'initial-template': QName('','go'), 'delivery-format': 'serialized'})?*`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(stylesheet)},
				transformFns(),
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), "FOXT0003")
		})

		t.Run("DocumentDelivery", func(t *testing.T) {
			out, err := evalTransform(t,
				`transform(map{'stylesheet-text': $ss, 'source-node': ., 'delivery-format': 'document'})?output`,
				sourceDoc,
				map[string]xpath3.Sequence{"ss": xpath3.SingleString(innerTransformStylesheet)},
				transformFns(),
			)
			require.NoError(t, err)
			require.Contains(t, out, "data")
		})
	})
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

// httpResolverFunc adapts a function to the xpath3.URIResolver interface.
type httpResolverFunc func(uri string) (io.ReadCloser, error)

func (f httpResolverFunc) ResolveURI(uri string) (io.ReadCloser, error) { return f(uri) }

// wantTrue is the string value of an XPath expression that evaluates to true().
const wantTrue = "true"

// transformSourceVar is the map key for the $xml source-document variable.
const transformSourceVar = "xml"

// xslXMLVars binds the $xsl (stylesheet) and $xml (source) let-variables read by
// the fn:transform result-map test expressions.
func xslXMLVars(xsl, xml string) map[string]xpath3.Sequence {
	return map[string]xpath3.Sequence{
		"xsl":              xpath3.SingleString(xsl),
		transformSourceVar: xpath3.SingleString(xml),
	}
}

// transformBool compiles and evaluates a boolean-valued XPath expression that
// drives the standalone fn:transform, returning its string value ("true" /
// "false"). vars supplies the let-bindings the expression reads.
func transformBool(t *testing.T, expr string, vars map[string]xpath3.Sequence) string {
	t.Helper()
	got, err := evalTransform(t, expr, nil, vars, transformFns())
	require.NoError(t, err)
	return got
}

// multiResultDocXSL emits one xsl:result-document per section and no principal
// output — the shape that must yield no principal ("output") entry (bug 30209).
const multiResultDocXSL = `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'><xsl:for-each select='//section'><xsl:result-document href='section{position()}.html'><out><xsl:value-of select='position()'/></out></xsl:result-document></xsl:for-each></xsl:template>
</xsl:stylesheet>`

const multiResultDocXML = `<doc><section>s1</section><section>s2</section><section>s3</section></doc>`

// principalOnlyXSL produces a single principal result element and no secondary
// result documents.
const principalOnlyXSL = `<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='3.0'>
<xsl:template match='/'><out><xsl:value-of select='name(*)'/></out></xsl:template>
</xsl:stylesheet>`

// globalContextXSL is the fn-transform-82 variable-with-context stylesheet: a
// global variable captures the global context item; the match='.' template
// reports whether the global context item and the matched node are document
// nodes, plus the name of the global context item.
const globalContextXSL = `<xsl:stylesheet version='3.0' xmlns:xsl='http://www.w3.org/1999/XSL/Transform'>
<xsl:variable name='v' select="."/>
<xsl:template match='.'><out root-is-doc="{$v instance of document-node()}" this-is-doc="{. instance of document-node()}"><xsl:value-of select='name($v)'/></out></xsl:template>
</xsl:stylesheet>`

// transformErr drives the standalone fn:transform and returns the error (nil on
// success). vars supplies the let-bindings the expression reads.
func transformErr(t *testing.T, expr string, vars map[string]xpath3.Sequence) error {
	t.Helper()
	_, err := evalTransform(t, expr, nil, vars, transformFns())
	return err
}

// innerTransformStylesheet is a small stylesheet exercised through the
// standalone fn:transform (xslt3.TransformFunction) registered on a bare
// xpath3.Evaluator. The match="/" template echoes the source root element name;
// the named template emits a fixed marker.
const innerTransformStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="name(*)"/></out></xsl:template>
  <xsl:template name="go"><out>named-template</out></xsl:template>
</xsl:stylesheet>`

// evalTransform compiles and evaluates expr against sourceDoc as the context
// node, with the supplied variable bindings and fn:transform behavior (found or
// stub) determined by fns.
func evalTransform(t *testing.T, expr string, sourceDoc *helium.Document, vars map[string]xpath3.Sequence, fns map[xpath3.QualifiedName]xpath3.Function) (string, error) {
	t.Helper()
	e := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	if vars != nil {
		e = e.Variables(vars)
	}
	if fns != nil {
		e = e.Functions(nil, fns)
	}
	x, err := xpath3.NewCompiler().Compile(expr)
	require.NoError(t, err)
	res, err := e.Evaluate(t.Context(), x, sourceDoc)
	if err != nil {
		return "", err
	}
	return res.StringValue(), nil
}

func transformFns() map[xpath3.QualifiedName]xpath3.Function {
	return transformFnsWith()
}

// transformFnsWith registers the standalone fn:transform (built with the given
// options) under the fn:transform QName for a bare xpath3.Evaluator.
func transformFnsWith(opts ...xslt3.TransformOption) map[xpath3.QualifiedName]xpath3.Function {
	return map[xpath3.QualifiedName]xpath3.Function{
		{URI: xpath3.NSFn, Name: "transform"}: xslt3.TransformFunction(opts...),
	}
}

// mapURIResolver serves a fixed set of URIs from an in-memory map, used to
// exercise relative xsl:include resolution in the fn:transform
// stylesheet-base-uri tests.
type mapURIResolver struct {
	files map[string]string
}

func (r mapURIResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	content, ok := r.files[uri]
	if !ok {
		return nil, fmt.Errorf("mapURIResolver: no such URI %q", uri)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

// recordingRejectResolver records every URI it is asked for and refuses to
// serve any of them. It lets a test prove which URI the compiler attempted to
// load (e.g. a raw relative href when no base URI was applied), and not only
// that resolution failed.
type recordingRejectResolver struct {
	requested []string
}

func (r *recordingRejectResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	r.requested = append(r.requested, uri)
	return nil, fmt.Errorf("recordingRejectResolver: refusing %q", uri)
}

// includedTemplateStylesheet is served as an xsl:include target; its match="/"
// template echoes the source root element name.
const includedTemplateStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out><xsl:value-of select="name(*)"/></out></xsl:template>
</xsl:stylesheet>`

// includingStylesheet pulls in the template above via a relative href, so it
// only compiles when a usable base URI resolves the include.
const includingStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="sub/inc.xsl"/>
</xsl:stylesheet>`
