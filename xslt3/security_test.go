package xslt3_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// Mirrors the xpath3 security tests landed in #417 for the xslt3 internal
// loadDocument path. XSLT 3.0's document() / fn:doc() now refuse to fall
// back to os.ReadFile or http.DefaultClient. A caller must opt in by
// installing a URIResolver and/or HTTPClient on Invocation.

const fnDocStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:template match="/">
    <out>
      <xsl:copy-of select="doc($url)"/>
    </out>
  </xsl:template>
  <xsl:param name="url"/>
</xsl:stylesheet>`

func compileFnDocStylesheet(t *testing.T) *xslt3.Stylesheet {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(fnDocStylesheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)
	return ss
}

func TestFnDoc_NoFileReadByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.xml")
	require.NoError(t, os.WriteFile(path, []byte("<root>secret</root>"), 0o644))

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	ss := compileFnDocStylesheet(t)

	_, err = ss.Transform(source).
		SetParameter("url", xpath3.SingleString(path)).
		Serialize(t.Context())
	require.Error(t, err, "default-deny: doc() must refuse filesystem access without URIResolver")
	require.True(t, strings.Contains(err.Error(), "no URIResolver"),
		"error must explain that a URIResolver is required, got: %v", err)
}

func TestFnDoc_NoNetworkByDefault(t *testing.T) {
	t.Parallel()
	// If any request reaches the test server, hits > 0 and the test fails.
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("<root/>"))
	}))
	defer srv.Close()

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	ss := compileFnDocStylesheet(t)

	_, err = ss.Transform(source).
		SetParameter("url", xpath3.SingleString(srv.URL+"/x")).
		Serialize(t.Context())
	require.Error(t, err, "default-deny: doc() must refuse network access without HTTPClient/URIResolver")
	require.Zero(t, hits.Load(), "no HTTP request should reach the test server")
}

// Sanity: when an HTTPClient is explicitly configured, retrieval is allowed.
// This guards against the helper accidentally rejecting opt-in callers.
func TestFnDoc_HTTPClientEnablesNetwork(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<root>fetched</root>"))
	}))
	defer srv.Close()

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)
	ss := compileFnDocStylesheet(t)

	out, err := ss.Transform(source).
		SetParameter("url", xpath3.SingleString(srv.URL+"/x")).
		HTTPClient(srv.Client()).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "fetched", "doc() should have fetched and embedded the document")
}

// URI schemes are case-insensitive per RFC 3986. A URL spelled "HTTP://..."
// must still be classified as HTTP and dispatched to the HTTPClient path —
// otherwise an opt-in caller would silently fall through to "no URIResolver".
func TestFnDoc_HTTPClientHandlesUppercaseScheme(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<root>fetched</root>"))
	}))
	defer srv.Close()

	// Uppercase scheme: HTTP://host:port/x
	upper := "HTTP" + strings.TrimPrefix(srv.URL, "http") + "/x"

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)
	ss := compileFnDocStylesheet(t)

	out, err := ss.Transform(source).
		SetParameter("url", xpath3.SingleString(upper)).
		HTTPClient(srv.Client()).
		Serialize(t.Context())
	require.NoError(t, err)
	require.Contains(t, out, "fetched")
}
