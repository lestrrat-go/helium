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
	testChildXML      = "child.xml"
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
		{"relative against local base", testChildXML, testDocsMainXML, "/docs/child.xml"},
		// Root-relative ref against a URI base keeps scheme+authority.
		{"root-relative against uri base", "/other/d.xml", "mem:/docs/main.xml", "mem:/other/d.xml"},
		{"relative against uri base", testChildXML, "mem:/docs/main.xml", "mem:/docs/child.xml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveAgainstBaseURI(tc.uri, tc.base)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestEnsureFileURI verifies that ensureFileURI normalizes both POSIX- and
// Windows-absolute filesystem paths into "file:" URIs (so a later url.Parse
// never reads a Windows drive letter as a URI scheme), and leaves paths that
// already carry a scheme untouched. The Windows shapes are plain strings, so
// the Windows behavior is exercised on Linux CI.
func TestEnsureFileURI(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"posix absolute", "/a/b/main.xsl", "file:///a/b/main.xsl"},
		{"windows drive backslash", `D:\a\helium\main.xsl`, "file:///D:/a/helium/main.xsl"},
		{"windows drive forward slash", `C:/styles/main.xsl`, "file:///C:/styles/main.xsl"},
		{"windows unc", `\\host\share\main.xsl`, "file://host/share/main.xsl"},
		{"already file uri", "file:///a/b.xsl", "file:///a/b.xsl"},
		{"already http uri", "http://example.com/a.xsl", "http://example.com/a.xsl"},
		{"relative left alone", "child.xsl", "child.xsl"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ensureFileURI(tc.in))
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
			_, _, _ = loadParameterDocumentFromFile(context.Background(), nil, &OutputDef{}, tc.base, tc.href, loadBytes, false, false, 0)
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
		{"relative under uri base", testChildXML, "mem://pkg", "mem://pkg/child.xml"},
		{"root-relative under uri base", "/other.xml", "mem://pkg/sub", "mem://pkg/other.xml"},
		// Both local: historical filepath behavior preserved.
		{"local relative", testChildXML, testDocsDir, "/docs/child.xml"},
		{"local absolute", "/abs.xml", testDocsDir, "/abs.xml"},
		// A POSIX-shaped file: URI resolves to a forward-slash path on every OS;
		// the ToSlash normalization on the FileURIToPath result is what stops
		// Windows from emitting "\abs\x.xml" here. (A drive-letter file: URI is
		// GOOS-dependent via FileURIToPath, so it is not asserted in this
		// cross-OS table.)
		{"file triple slash", "file:///abs/x.xml", testDocsDir, "/abs/x.xml"},
		// A "file:////server/share" UNC URI is rejected by FileURIToPath; the
		// fallback must NOT strip "file://" (which would yield the UNC path
		// "//server/share/x.xml" and reach a remote SMB host on Windows). The
		// original file: URI is returned unchanged so a local-path loader rejects it.
		{"file unc rejected not stripped", "file:////server/share/x.xml", testDocsDir, "file:////server/share/x.xml"},
		// url.Parse percent-decodes u.Path, so a "%5C"/"%5c" encoded backslash
		// decodes to "/\server/share" — still a UNC path on Windows. FileURIToPath
		// rejects it, so the fallback must keep the original file: URI verbatim
		// rather than stripping "file://" into a bare UNC path.
		{"file unc encoded backslash not stripped", "file:///%5Cserver/share/x.xml", testDocsDir, "file:///%5Cserver/share/x.xml"},
		{"file unc encoded backslash lower not stripped", "file:///%5cserver/share/x.xml", testDocsDir, "file:///%5cserver/share/x.xml"},
		// Windows-shaped local base resolves with forward-slash output on any OS
		// (a plain string here, so the Windows behavior is exercised on Linux).
		{"windows base relative ref", testChildXML, `C:\docs`, "C:/docs/child.xml"},
		{"windows-absolute ref verbatim", `C:\abs.xml`, `C:\docs`, `C:\abs.xml`},
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

// TestRefDenotesDirectory verifies that directory-detection looks only at the
// PATH portion of a reference: a query-only/fragment-only ref has an empty path
// and is NOT directory-denoting (path.Base("") == "." must not be mistaken for
// a "." dot-segment), while a path that ends in '/' before a query still is.
func TestRefDenotesDirectory(t *testing.T) {
	for _, tc := range []struct {
		ref  string
		want bool
	}{
		{"", false},
		{"dir/", true},
		{"dir", false},
		{".", true},
		{"..", true},
		{"a/b/", true},
		{"a/b", false},
		// Query-only / fragment-only: empty path portion, not a directory.
		{"?v", false},
		{"#frag", false},
		{"?a=b", false},
		// Path ends in '/' even with a trailing query/fragment.
		{"dir/?v", true},
		{"dir/#f", true},
		// Path is a plain file even with a query.
		{"dir?v", false},
		{"file.xml#frag", false},
		// Dot-segment paths with a query.
		{"..?v", true},
		{"./#f", true},
	} {
		t.Run(tc.ref, func(t *testing.T) {
			require.Equal(t, tc.want, refDenotesDirectory(tc.ref))
		})
	}
}

// TestEnsureDirSlash verifies the trailing path slash is inserted before any
// query/fragment, never at the very end — so a directory base carrying a query
// ("…/dir?v") becomes "…/dir/?v", not the path-corrupting "…/dir?v/".
func TestEnsureDirSlash(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"a/b/dir", "a/b/dir/"},
		{"a/b/dir/", "a/b/dir/"},
		{"a/b/dir?v", "a/b/dir/?v"},
		{"a/b/dir?v=1&w=2", "a/b/dir/?v=1&w=2"},
		{"a/b/dir#f", "a/b/dir/#f"},
		{"a/b/dir/?v", "a/b/dir/?v"},
		{"a/b/dir/#f", "a/b/dir/#f"},
		{"mem://h/dir?v", "mem://h/dir/?v"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, ensureDirSlash(tc.in))
		})
	}
}

var errStopAfterResolve = stopAfterResolveError{}

type stopAfterResolveError struct{}

func (stopAfterResolveError) Error() string { return "stop after resolve" }
