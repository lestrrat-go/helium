package xslt3_test

import (
	"io"
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

// A hostile or pathological resource must not be read in full: doc()/fn:doc
// reads through a bounded reader capped at [xslt3.MaxResourceBytes]. The server
// streams more than the cap; the transform must fail with an error rather than
// buffering the whole body into memory. The handler tracks how many bytes it
// actually wrote so we can confirm the client stopped reading near the cap
// instead of draining the entire (effectively unbounded) stream.
func TestFnDoc_OverLimitResourceRejected(t *testing.T) {
	t.Parallel()

	// Far larger than MaxResourceBytes so a successful full read would be obvious.
	const total = xslt3.MaxResourceBytes * 4

	var written atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Begin with a well-formed document opener so that, absent the bound,
		// the read would otherwise plausibly proceed; the padding never closes
		// it. The size — not the well-formedness — is what must trip the guard.
		buf := make([]byte, 64*1024)
		for i := range buf {
			buf[i] = 'a'
		}
		var sent int
		for sent < total {
			n := len(buf)
			if remaining := total - sent; remaining < n {
				n = remaining
			}
			m, err := w.Write(buf[:n])
			written.Add(int64(m))
			sent += m
			if err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)
	ss := compileFnDocStylesheet(t)

	_, err = ss.Transform(source).
		SetParameter("url", xpath3.SingleString(srv.URL+"/x")).
		HTTPClient(srv.Client()).
		Serialize(t.Context())
	require.Error(t, err, "an over-limit resource must be rejected, not fully read")
	require.Less(t, written.Load(), int64(total),
		"client must stop reading near the cap, not drain the whole stream")
}

// An over-cap doc() read must surface [xslt3.ErrResourceTooLarge] through the
// XSLTError wrapper, as the public API documents, while still matching
// [xslt3.ErrDynamicError]. The wrapped error previously discarded the cause
// (it was formatted with %v), so errors.Is(err, ErrResourceTooLarge) was false.
func TestFnDoc_OverLimitErrorIsResourceTooLarge(t *testing.T) {
	t.Parallel()

	const u = "http://example.invalid/big.xml"
	// A well-formed document comfortably larger than the default cap.
	body := "<root>" + strings.Repeat("a", int(xslt3.MaxResourceBytes)+1024) + "</root>"

	resolver := httpResolverFunc(func(uri string) (io.ReadCloser, error) {
		if uri != u {
			return nil, &xpath3.XPathError{Code: "FOUT1170", Message: "not found: " + uri}
		}
		return io.NopCloser(strings.NewReader(body)), nil
	})

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)
	ss := compileFnDocStylesheet(t)

	_, err = ss.Transform(source).
		SetParameter("url", xpath3.SingleString(u)).
		URIResolver(resolver).
		Serialize(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge,
		"over-cap read must remain matchable via errors.Is(err, ErrResourceTooLarge)")
	require.ErrorIs(t, err, xslt3.ErrDynamicError,
		"over-cap read must still match ErrDynamicError")
}

// A resource larger than the default cap is accepted when the per-invocation
// cap is raised via Invocation.MaxResourceBytes. Exercises the full doc()
// retrieval path, confirming the configured override actually threads through.
func TestFnDoc_RaisedCapAcceptsLargeResource(t *testing.T) {
	t.Parallel()

	// A well-formed XML document comfortably larger than the default cap.
	const padding = xslt3.MaxResourceBytes + (1 << 20) // > 10 MiB
	var b strings.Builder
	b.Grow(padding + 64)
	b.WriteString("<root>")
	for b.Len() < padding {
		b.WriteString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	}
	b.WriteString("</root>")
	body := b.String()
	require.Greater(t, len(body), int(xslt3.MaxResourceBytes))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)
	ss := compileFnDocStylesheet(t)

	// Default cap rejects it.
	_, err = ss.Transform(source).
		SetParameter("url", xpath3.SingleString(srv.URL+"/x")).
		HTTPClient(srv.Client()).
		Serialize(t.Context())
	require.Error(t, err, "default cap must reject the over-limit resource")

	// Raised cap accepts it.
	out, err := ss.Transform(source).
		SetParameter("url", xpath3.SingleString(srv.URL+"/x")).
		HTTPClient(srv.Client()).
		MaxResourceBytes(int64(len(body)) + 1).
		Serialize(t.Context())
	require.NoError(t, err, "raised cap must accept the resource")
	require.Contains(t, out, "<root>")
}

// Sanity: the bounded helper itself rejects an over-limit reader and accepts a
// reader exactly at the cap. Guards the bound logic independent of the HTTP path.
func TestReadResourceBounded_Limit(t *testing.T) {
	t.Parallel()

	_, err := xslt3.ReadResourceBoundedForTest(io.LimitReader(neverEndingReader{}, xslt3.MaxResourceBytes+1), 0)
	require.ErrorIs(t, err, xslt3.ErrResourceTooLarge)

	data, err := xslt3.ReadResourceBoundedForTest(io.LimitReader(neverEndingReader{}, xslt3.MaxResourceBytes), 0)
	require.NoError(t, err)
	require.Len(t, data, xslt3.MaxResourceBytes)

	// A raised cap accepts a reader larger than the default.
	data, err = xslt3.ReadResourceBoundedForTest(io.LimitReader(neverEndingReader{}, xslt3.MaxResourceBytes+1), xslt3.MaxResourceBytes+1)
	require.NoError(t, err)
	require.Len(t, data, xslt3.MaxResourceBytes+1)

	// A negative cap disables the bound entirely.
	data, err = xslt3.ReadResourceBoundedForTest(io.LimitReader(neverEndingReader{}, xslt3.MaxResourceBytes*2), -1)
	require.NoError(t, err)
	require.Len(t, data, xslt3.MaxResourceBytes*2)
}

// neverEndingReader yields an unbounded stream of 'a' bytes.
type neverEndingReader struct{}

func (neverEndingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	return len(p), nil
}
