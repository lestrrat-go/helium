package xslt3

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testStylesMainXSL = "/styles/main.xsl"
	testDocsDir       = "/docs"
	testDocsMainXML   = "/docs/main.xml"
)

// TestResolveAgainstBaseURIAbsolute verifies that resolveAgainstBaseURI
// (used by document() / xsl:source-document resolution) treats an absolute
// URI reference that has a scheme but no "://" authority (e.g. "urn:shared",
// "file:/docs/d.xml") as absolute and returns it UNCHANGED, instead of
// filepath.Join'ing it onto the base directory.
func TestResolveAgainstBaseURIAbsolute(t *testing.T) {
	for _, tc := range []struct {
		name string
		uri  string
		base string
		want string
	}{
		{"urn opaque", "urn:shared", testDocsMainXML, "urn:shared"},
		{"file single slash", "file:/docs/d.xml", testDocsMainXML, "file:/docs/d.xml"},
		{"http authority", "http://example.com/d.xml", testDocsMainXML, "http://example.com/d.xml"},
		{"relative against local base", "child.xml", testDocsMainXML, "/docs/child.xml"},
		// Root-relative ref against a URI base keeps scheme+authority.
		{"root-relative against uri base", "/other/d.xml", "mem:/docs/main.xml", "mem:/other/d.xml"},
		{"relative against uri base", "child.xml", "mem:/docs/main.xml", "mem:/docs/child.xml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveAgainstBaseURI(tc.uri, tc.base)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestLoadParameterDocumentURIAbsolute verifies that the serialization
// parameter-document loader hands an absolute-URI href (scheme, no "://")
// to its loader unchanged rather than filepath.Join'ing it onto the base.
func TestLoadParameterDocumentURIAbsolute(t *testing.T) {
	for _, tc := range []struct {
		name string
		base string
		href string
		want string
	}{
		{"urn opaque", testStylesMainXSL, "urn:params", "urn:params"},
		{"file single slash", testStylesMainXSL, "file:/params/p.xml", "file:/params/p.xml"},
		{"http authority", testStylesMainXSL, "http://example.com/p.xml", "http://example.com/p.xml"},
		{"relative against local base", testStylesMainXSL, "p.xml", "/styles/p.xml"},
		{"root-relative against uri base", "mem:/styles/main.xsl", "/p/p.xml", "mem:/p/p.xml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var seen string
			loadBytes := func(_ context.Context, uri string) ([]byte, error) {
				seen = uri
				// Return a non-nil error to short-circuit parsing; we only
				// care which URI the loader was asked for.
				return nil, errStopAfterResolve
			}
			_ = loadParameterDocumentFromFile(context.Background(), &OutputDef{}, tc.base, tc.href, loadBytes, false)
			require.Equal(t, tc.want, seen)
		})
	}
}

// TestResolveTransformStylesheetLocURIBase verifies that fn:transform
// stylesheet-location resolution (functions_transform.go) is URI-aware for the
// base: when the stylesheet base URI has a scheme, a filepath-absolute /
// root-relative stylesheet-location such as "/inner.xsl" is resolved against
// the base scheme/authority (mem://pkg/main.xsl + /inner.xsl ->
// mem://pkg/inner.xsl) instead of being passed through verbatim, while a
// purely-local absolute path against a local base is left unchanged.
func TestResolveTransformStylesheetLocURIBase(t *testing.T) {
	for _, tc := range []struct {
		name string
		base string
		loc  string
		want string
	}{
		// Regression: root-relative loc under a URI base must keep scheme+authority.
		{"root-relative under uri base", "mem://pkg/main.xsl", "/inner.xsl", "mem://pkg/inner.xsl"},
		{"root-relative under uri base no authority", "mem:/pkg/main.xsl", "/inner.xsl", "mem:/inner.xsl"},
		{"relative under uri base", "mem://pkg/main.xsl", "inner.xsl", "mem://pkg/inner.xsl"},
		{"absolute uri loc", "mem://pkg/main.xsl", "urn:other", "urn:other"},
		// Purely-local absolute path under a local base is left unchanged.
		{"local absolute under local base", testStylesMainXSL, "/inner.xsl", "/inner.xsl"},
		{"local relative under local base", testStylesMainXSL, "inner.xsl", "/styles/inner.xsl"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, resolveStylesheetLocation(tc.base, tc.loc))
		})
	}
}

// TestResolveDocumentURIAbsolute verifies that runtime document()/doc()
// resolution (functions.go resolveDocumentURI) is URI-aware: an absolute-URI
// ref carrying a scheme with no "//" authority (e.g. document('urn:doc'),
// doc('file:/x.xml')) is returned UNCHANGED rather than filepath.Join'ed onto
// the base dir, and a relative ref against a URI base keeps the base
// scheme/authority instead of losing it before retrieveDocumentBytes.
func TestResolveDocumentURIAbsolute(t *testing.T) {
	ec := &execContext{}
	for _, tc := range []struct {
		name    string
		uri     string
		baseDir string
		want    string
	}{
		// Regression: absolute-URI refs (scheme, no "//") pass through unchanged.
		{"urn opaque", "urn:doc", testDocsDir, "urn:doc"},
		{"file single slash", "file:/x.xml", testDocsDir, "file:/x.xml"},
		{"http authority", "http://example.com/d.xml", testDocsDir, "http://example.com/d.xml"},
		// Relative ref against a URI base keeps scheme/authority.
		{"relative under uri base", "child.xml", "mem://pkg", "mem://pkg/child.xml"},
		{"root-relative under uri base", "/other.xml", "mem://pkg/sub", "mem://pkg/other.xml"},
		// Both local: historical filepath behavior preserved.
		{"local relative", "child.xml", testDocsDir, "/docs/child.xml"},
		{"local absolute", "/abs.xml", testDocsDir, "/abs.xml"},
		{"file triple slash", "file:///abs/x.xml", testDocsDir, "/abs/x.xml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ec.resolveDocumentURI(tc.uri, tc.baseDir)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestRootXMLBaseURIAware verifies that root xml:base handling (compile.go) is
// URI-aware: an absolute-URI xml:base with no "//" authority (xml:base="urn:base")
// is used as-is, and a root-relative xml:base ("/pkg/") under a URI base keeps
// the base scheme/authority (resolved per RFC 3986) instead of being corrupted
// by a strings.Contains("://") classification + filepath.Join.
func TestRootXMLBaseURIAware(t *testing.T) {
	for _, tc := range []struct {
		name    string
		baseURI string
		xmlBase string
		want    string
	}{
		// Regression: absolute-URI xml:base without "//" is used as-is.
		{"absolute uri without slashes", "mem://pkg/main.xsl", "urn:base", "urn:base"},
		// Regression: root-relative xml:base under a URI base keeps scheme+authority.
		{"root-relative under uri base", "mem://pkg/a/main.xsl", "/pkg2/", "mem://pkg/pkg2/main.xsl"},
		{"relative under uri base", "mem://pkg/a/main.xsl", "inner/", "mem://pkg/a/inner/main.xsl"},
		{"absolute uri with authority", "file:///a/main.xsl", "http://h/x.xsl", "http://h/x.xsl"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveRootXMLBase(tc.baseURI, tc.xmlBase)
			require.Equal(t, tc.want, got)
		})
	}
}

var errStopAfterResolve = stopAfterResolveError{}

type stopAfterResolveError struct{}

func (stopAfterResolveError) Error() string { return "stop after resolve" }
