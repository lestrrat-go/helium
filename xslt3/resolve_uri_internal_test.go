package xslt3

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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
		{"urn opaque", "urn:shared", "/docs/main.xml", "urn:shared"},
		{"file single slash", "file:/docs/d.xml", "/docs/main.xml", "file:/docs/d.xml"},
		{"http authority", "http://example.com/d.xml", "/docs/main.xml", "http://example.com/d.xml"},
		{"relative against local base", "child.xml", "/docs/main.xml", "/docs/child.xml"},
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
		{"urn opaque", "/styles/main.xsl", "urn:params", "urn:params"},
		{"file single slash", "/styles/main.xsl", "file:/params/p.xml", "file:/params/p.xml"},
		{"http authority", "/styles/main.xsl", "http://example.com/p.xml", "http://example.com/p.xml"},
		{"relative against local base", "/styles/main.xsl", "p.xml", "/styles/p.xml"},
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
			_ = loadParameterDocumentFromFile(context.Background(), &OutputDef{}, tc.base, tc.href, loadBytes)
			require.Equal(t, tc.want, seen)
		})
	}
}

var errStopAfterResolve = stopAfterResolveError{}

type stopAfterResolveError struct{}

func (stopAfterResolveError) Error() string { return "stop after resolve" }
