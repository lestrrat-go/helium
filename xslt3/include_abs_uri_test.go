package xslt3_test

import (
	"bytes"
	"io"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

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

const childModule = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:template match="/data">
    <out value="{@v}"/>
  </xsl:template>
</xsl:stylesheet>`

// TestIncludeAbsoluteURIHrefPassedThrough verifies that an xsl:include /
// xsl:import href that is an absolute URI reference (with a scheme but no "://"
// authority, e.g. "urn:shared" or "file:/modules/child.xsl") is handed to the
// URIResolver unchanged — not filepath.Join'ed onto the stylesheet base.
func TestIncludeAbsoluteURIHrefPassedThrough(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		decl string // include or import
		base string // compiler base URI
		href string
	}{
		{"include urn", "include", "/styles/main.xsl", "urn:shared"},
		{"import urn", "import", "/styles/main.xsl", "urn:shared"},
		{"include file scheme single slash", "include", "/styles/main.xsl", "file:/modules/child.xsl"},
		{"import data scheme", "import", "/styles/main.xsl", "data:application/xslt+xml,child"},
		{"include http scheme", "include", "/styles/main.xsl", "http://example.com/modules/child.xsl"},
		// Windows drive-letter paths are filesystem paths, not URIs. With a URI
		// base they used to fall through to RFC 3986 resolution and be
		// lowercased / dot-segment-mangled; they must reach the resolver
		// verbatim. A URI base is what triggers the corruption, so use one here.
		{"include windows drive forward slash", "include", "mem:/styles/main.xsl", "C:/modules/child.xsl"},
		{"import windows drive back slash", "import", "mem:/styles/main.xsl", `C:\modules\child.xsl`},
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
}

// TestIncludeRelativeHrefResolvedAgainstBase verifies that a relative href is
// still resolved against the (URI) stylesheet base, not passed through bare.
func TestIncludeRelativeHrefResolvedAgainstBase(t *testing.T) {
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
}
