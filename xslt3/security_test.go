package xslt3_test

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

func TestFnDoc(t *testing.T) {
	t.Run("no file read by default", func(t *testing.T) {
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
	})

	t.Run("no network by default", func(t *testing.T) {
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
	})

	// Sanity: when an HTTPClient is explicitly configured, retrieval is allowed.
	// This guards against the helper accidentally rejecting opt-in callers.
	t.Run("HTTP client enables network", func(t *testing.T) {
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
	})

	// URI schemes are case-insensitive per RFC 3986. A URL spelled "HTTP://..."
	// must still be classified as HTTP and dispatched to the HTTPClient path —
	// otherwise an opt-in caller would silently fall through to "no URIResolver".
	t.Run("HTTP client handles uppercase scheme", func(t *testing.T) {
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
	})

	// A hostile or pathological resource must not be read in full: doc()/fn:doc
	// reads through a bounded reader capped at [xslt3.MaxResourceBytes]. The server
	// streams more than the cap; the transform must fail with an error, buffering no
	// whole body into memory. The handler tracks how many bytes it
	// actually wrote so we can confirm the client stopped reading near the cap
	// instead of draining the entire (effectively unbounded) stream.
	t.Run("over limit resource rejected", func(t *testing.T) {
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
	})

	// An over-cap doc() read must surface [xslt3.ErrResourceTooLarge] through the
	// XSLTError wrapper, as the public API documents, while still matching
	// [xslt3.ErrDynamicError]. The wrapped error previously discarded the cause
	// (it was formatted with %v), so errors.Is(err, ErrResourceTooLarge) was false.
	t.Run("over limit error is resource too large", func(t *testing.T) {
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
	})

	// A resource larger than the default cap is accepted when the per-invocation
	// cap is raised via Invocation.MaxResourceBytes. Exercises the full doc()
	// retrieval path, confirming the configured override actually threads through.
	t.Run("raised cap accepts large resource", func(t *testing.T) {
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
	})
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

// xxeResolver serves a fixed set of URIs from an in-memory map. It mimics a
// URIResolver / Compiler.URIResolver that a caller installs to permit external
// document / stylesheet loading.
type xxeResolver struct {
	files map[string]string
}

func (r *xxeResolver) Resolve(uri string) (io.ReadCloser, error) {
	body, ok := r.files[uri]
	if !ok {
		// The map is keyed on native paths (filepath.Join). helium's URI
		// resolution emits forward slashes on every OS, so on Windows the
		// resolved URI is "C:/dir/x" while the key is "C:\\dir\\x". Retry with
		// the native-separator form so the lookup matches on both platforms
		// (filepath.FromSlash is a no-op on POSIX).
		body, ok = r.files[filepath.FromSlash(uri)]
	}
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

func (r *xxeResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	return r.Resolve(uri)
}

// xxeRuntimeStylesheet loads an external document via doc() and copies it.
const xxeRuntimeStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:param name="url"/>
  <xsl:template match="/">
    <out><xsl:value-of select="doc($url)/payload"/></out>
  </xsl:template>
</xsl:stylesheet>`

func TestXXE(t *testing.T) {
	// A-001: runtime fn:doc / document() of a resolver-served doc whose XML
	// defines an external SYSTEM entity referencing a local file must NOT expand
	// that entity by default (XXE blocked).
	t.Run("runtime doc blocked by default", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("TOP-SECRET"), 0o600))
		docPath := filepath.Join(dir, "doc.xml")

		// The relative SYSTEM entity resolves against the document's on-disk URI;
		// under the legacy permissive parse this would expand to the secret file's
		// contents (see TestXXE_RuntimeDocAllowedWithOptIn). The default must block it.
		docBody := `<?xml version="1.0"?>
<!DOCTYPE payload [ <!ENTITY leak SYSTEM "secret.txt"> ]>
<payload>&leak;</payload>`

		resolver := &xxeResolver{files: map[string]string{docPath: docBody}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(xxeRuntimeStylesheet))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		out, err := ss.Transform(source).
			URIResolver(resolver).
			SetParameter("url", xpath3.SingleString(docPath)).
			Serialize(t.Context())
		// Either parsing fails, or the entity is not expanded. In neither case may
		// the secret leak into the output.
		if err == nil {
			require.NotContains(t, out, "TOP-SECRET",
				"external entity must not be expanded by default")
		}
	})

	// A-001 opt-in: AllowExternalEntities(true) restores external entity loading,
	// but the load is now ROUTED THROUGH the configured URIResolver (not the raw
	// filesystem). The external SYSTEM entity resolves against the document's base
	// URI and the resulting entity URI is served by the same resolver. This proves
	// opted-in entities go through the resolver-mediated, resource-limited channel,
	// and never the parser's default os.Open.
	t.Run("runtime doc allowed with opt in", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		docPath := filepath.Join(dir, "doc.xml")
		// The relative SYSTEM entity "secret.txt" resolves against the document's
		// base URI (docPath's directory); the resolver serves that resolved URI.
		secretURI := filepath.Join(dir, "secret.txt")

		docBody := `<?xml version="1.0"?>
<!DOCTYPE payload [ <!ENTITY leak SYSTEM "secret.txt"> ]>
<payload>&leak;</payload>`

		resolver := &xxeResolver{files: map[string]string{
			docPath:   docBody,
			secretURI: "LEGACY-VALUE",
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(xxeRuntimeStylesheet))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		out, err := ss.Transform(source).
			URIResolver(resolver).
			AllowExternalEntities(true).
			SetParameter("url", xpath3.SingleString(docPath)).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, "LEGACY-VALUE",
			"opt-in must restore legacy external entity expansion")
	})

	// A-002: xsl:include of a resolver-returned stylesheet module that defines an
	// external SYSTEM entity referencing a local file must NOT expand that entity
	// by default (compile-time XXE blocked).
	t.Run("stylesheet include blocked by default", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("INCLUDE-SECRET"), 0o600))
		incPath := filepath.Join(dir, "inc.xsl")

		includedXSL := `<?xml version="1.0"?>
<!DOCTYPE xsl:stylesheet [ <!ENTITY leak SYSTEM "secret.txt"> ]>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:template name="leaked"><val>&leak;</val></xsl:template>
</xsl:stylesheet>`

		mainXSL := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:include href="` + incPath + `"/>
  <xsl:template match="/"><out><xsl:call-template name="leaked"/></out></xsl:template>
</xsl:stylesheet>`

		resolver := &xxeResolver{files: map[string]string{incPath: includedXSL}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(mainXSL))
		require.NoError(t, err)

		ss, err := xslt3.NewCompiler().URIResolver(resolver).Compile(t.Context(), doc)
		if err != nil {
			// Compile may reject the included module outright when the external
			// entity is blocked; that is an acceptable secure outcome.
			return
		}

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		out, err := ss.Transform(source).Serialize(t.Context())
		if err == nil {
			require.NotContains(t, out, "INCLUDE-SECRET",
				"external entity in included stylesheet must not be expanded by default")
		}
	})

	// A-002 opt-in: Compiler.AllowExternalEntities(true) restores stylesheet-module
	// entity expansion, with the external entity load routed through the configured
	// URIResolver (not the raw filesystem). The resolver serves both the included
	// module and the resolved entity URI.
	t.Run("stylesheet include allowed with opt in", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		// The relative SYSTEM entity "secret.txt" resolves against the included
		// module's base URI; the resolver serves that resolved URI.
		secretURI := filepath.Join(dir, "secret.txt")
		incPath := filepath.Join(dir, "inc.xsl")

		includedXSL := `<?xml version="1.0"?>
<!DOCTYPE xsl:stylesheet [ <!ENTITY leak SYSTEM "secret.txt"> ]>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:template name="leaked"><val>&leak;</val></xsl:template>
</xsl:stylesheet>`

		mainXSL := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:include href="` + incPath + `"/>
  <xsl:template match="/"><out><xsl:call-template name="leaked"/></out></xsl:template>
</xsl:stylesheet>`

		resolver := &xxeResolver{files: map[string]string{
			incPath:   includedXSL,
			secretURI: "INCLUDE-LEGACY",
		}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(mainXSL))
		require.NoError(t, err)

		ss, err := xslt3.NewCompiler().
			URIResolver(resolver).
			AllowExternalEntities(true).
			Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		out, err := ss.Transform(source).AllowExternalEntities(true).Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, "INCLUDE-LEGACY",
			"opt-in must restore legacy stylesheet-module entity expansion")
	})

	// A-003 (regression): under the secure default (XXE blocked), fn:doc() must
	// still apply internal-subset DTD default attributes. The secure parser path
	// previously dropped the extraOpts (DefaultDTDAttributes), so the @kind default
	// vanished and the output was <out/>. Internal-subset DTD processing must keep
	// working while EXTERNAL DTD/entity/network stay blocked.
	t.Run("runtime doc internal DTD default attr", func(t *testing.T) {
		t.Parallel()

		docPath := "mem://doc.xml"
		docBody := `<?xml version="1.0"?>
<!DOCTYPE payload [ <!ATTLIST payload kind CDATA "defaulted"> ]>
<payload/>`

		resolver := &xxeResolver{files: map[string]string{docPath: docBody}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(xxeDocAttrStylesheet))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		out, err := ss.Transform(source).
			URIResolver(resolver).
			SetParameter("url", xpath3.SingleString(docPath)).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, "defaulted",
			"internal-subset DTD default attribute must apply under the secure default")
	})

	// A-004: opted-in external entities must be loaded THROUGH the configured
	// URIResolver, not the parser's raw filesystem. The secret file exists on disk
	// (where a raw-FS parse would read it) but the resolver does NOT serve its URI;
	// the entity must therefore fail to resolve and never leak the on-disk content.
	t.Run("runtime doc opt in uses resolver not raw FS", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("RAW-FS-SECRET"), 0o600))
		docPath := filepath.Join(dir, "doc.xml")

		docBody := `<?xml version="1.0"?>
<!DOCTYPE payload [ <!ENTITY leak SYSTEM "secret.txt"> ]>
<payload>&leak;</payload>`

		// Resolver serves only the document, NOT the entity's resolved URI.
		resolver := &xxeResolver{files: map[string]string{docPath: docBody}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(xxeRuntimeStylesheet))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		out, err := ss.Transform(source).
			URIResolver(resolver).
			AllowExternalEntities(true).
			SetParameter("url", xpath3.SingleString(docPath)).
			Serialize(t.Context())
		// Even opted-in, the entity load goes through the resolver, which does not
		// serve secret.txt; the raw filesystem must never be consulted.
		if err == nil {
			require.NotContains(t, out, "RAW-FS-SECRET",
				"opted-in external entity must load via resolver, not raw filesystem")
		}
	})

	// A-006 (regression): under the secure default (XXE blocked), fn:doc() must
	// still expand INTERNAL general entities defined in the document's internal
	// subset. The secure parser path previously dropped SubstituteEntities(true),
	// so &local; survived as an EntityRefNode and the XPath string-value of
	// /payload was empty, yielding <out/>. Internal-subset entity substitution must
	// keep working while EXTERNAL DTD/entity/network stay blocked.
	t.Run("runtime doc internal entity expands", func(t *testing.T) {
		t.Parallel()

		docPath := "mem://doc.xml"
		docBody := `<?xml version="1.0"?>
<!DOCTYPE payload [ <!ENTITY local "ok"> ]>
<payload>&local;</payload>`

		resolver := &xxeResolver{files: map[string]string{docPath: docBody}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(xxeDocEntityStylesheet))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		out, err := ss.Transform(source).
			URIResolver(resolver).
			SetParameter("url", xpath3.SingleString(docPath)).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, "ok",
			"internal-subset general entity must expand under the secure default")
	})

	// A-007 (regression): when the OUTER fn:transform invocation opts into external
	// entities, a nested fn:transform() must inherit that opt-in so its own doc()
	// loads (with an external SYSTEM entity, served via the resolver) are permitted.
	// The nested transformConfig previously did not inherit allowExternalEntities,
	// forcing the inner doc() back to the blocked posture even when the outer caller
	// opted in.
	t.Run("nested fn transform inherits opt in", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		// Inner stylesheet loaded by the outer fn:transform; it itself runs a doc()
		// whose XML defines an external SYSTEM entity that the resolver serves.
		innerLoc := filepath.Join(dir, "inner.xsl")
		dataPath := filepath.Join(dir, "data.xml")
		secretURI := filepath.Join(dir, "secret.txt")

		innerXSL := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:param name="data-url"/>
  <xsl:template match="/">
    <inner><xsl:value-of select="doc($data-url)/payload"/></inner>
  </xsl:template>
</xsl:stylesheet>`

		dataXML := `<?xml version="1.0"?>
<!DOCTYPE payload [ <!ENTITY leak SYSTEM "secret.txt"> ]>
<payload>&leak;</payload>`

		resolver := &xxeResolver{files: map[string]string{
			innerLoc:  innerXSL,
			dataPath:  dataXML,
			secretURI: "NESTED-LEGACY",
		}}

		outerXSL := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:param name="inner-loc"/>
  <xsl:param name="data-url"/>
  <xsl:template match="/">
    <xsl:variable name="result" select="transform(map{
      'stylesheet-location': $inner-loc,
      'stylesheet-params': map{ QName('','data-url'): $data-url },
      'delivery-format': 'serialized'
    })"/>
    <out><xsl:value-of select="$result('output')"/></out>
  </xsl:template>
</xsl:stylesheet>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(outerXSL))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().
			URIResolver(resolver).
			AllowExternalEntities(true).
			Compile(t.Context(), doc)
		require.NoError(t, err)

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)

		out, err := ss.Transform(source).
			URIResolver(resolver).
			AllowExternalEntities(true).
			SetParameter("inner-loc", xpath3.SingleString(innerLoc)).
			SetParameter("data-url", xpath3.SingleString(dataPath)).
			Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, out, "NESTED-LEGACY",
			"nested fn:transform must inherit the outer external-entity opt-in")
	})

	// A-005: imported XSD schemas are ALWAYS parsed XXE-blocked. Even with
	// Compiler.AllowExternalEntities(true), an external SYSTEM entity in an imported
	// schema must not be expanded — the entity opt-in does not extend to schemas.
	t.Run("import schema always blocked", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("SCHEMA-SECRET"), 0o600))
		schemaPath := filepath.Join(dir, "schema.xsd")

		// The schema documentation carries an external SYSTEM entity reference. If
		// the entity were expanded, SCHEMA-SECRET would enter the parsed schema.
		schemaBody := `<?xml version="1.0"?>
<!DOCTYPE xs:schema [ <!ENTITY leak SYSTEM "secret.txt"> ]>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="urn:xxe" xmlns:t="urn:xxe">
  <xs:element name="root" type="xs:string"/>
  <xs:annotation><xs:documentation>&leak;</xs:documentation></xs:annotation>
</xs:schema>`

		mainXSL := `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:t="urn:xxe" version="3.0">
  <xsl:import-schema namespace="urn:xxe" schema-location="` + schemaPath + `"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

		resolver := &xxeResolver{files: map[string]string{schemaPath: schemaBody}}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(mainXSL))
		require.NoError(t, err)

		// AllowExternalEntities(true) must NOT cause the schema's external entity to
		// be expanded. Compilation may succeed (entity skipped/unexpanded) or fail
		// (entity rejected); in neither case may the secret leak.
		ss, err := xslt3.NewCompiler().
			URIResolver(resolver).
			AllowExternalEntities(true).
			Compile(t.Context(), doc)
		if err != nil {
			require.NotContains(t, err.Error(), "SCHEMA-SECRET",
				"schema external entity must never be expanded")
			return
		}

		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)
		out, err := ss.Transform(source).Serialize(t.Context())
		if err == nil {
			require.NotContains(t, out, "SCHEMA-SECRET",
				"schema external entity must never be expanded even with opt-in")
		}
	})
}

// xxeDocAttrStylesheet loads an external document via doc() and emits the value
// of an attribute that is supplied solely by an internal-subset DTD default.
const xxeDocAttrStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:param name="url"/>
  <xsl:template match="/">
    <out><xsl:value-of select="doc($url)/payload/@kind"/></out>
  </xsl:template>
</xsl:stylesheet>`

// xxeDocEntityStylesheet loads an external document via doc() and emits the
// string value of its root element, which contains an internal general entity
// reference.
const xxeDocEntityStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:param name="url"/>
  <xsl:template match="/">
    <out><xsl:value-of select="doc($url)/payload"/></out>
  </xsl:template>
</xsl:stylesheet>`

// osOpenResolver is an explicit opt-in compile-time URIResolver that reads
// modules straight off the local filesystem. Tests that load real files supply
// it explicitly because implicit filesystem access is no longer permitted.
type osOpenResolver struct{}

func (osOpenResolver) Resolve(uri string) (io.ReadCloser, error) {
	return os.Open(uri)
}

// fileMapResolver is an xslt3.URIResolver (method Resolve) that serves
// content from an in-memory map keyed by URI. Lookup falls back to matching by
// base name so the test does not depend on how xsl:import/include resolves the
// href (it uses filepath.Join, whose separators are OS-dependent).
type fileMapResolver struct {
	files map[string]string
}

func (r fileMapResolver) Resolve(uri string) (io.ReadCloser, error) {
	content, ok := r.files[uri]
	if !ok {
		want := baseName(uri)
		for k, v := range r.files {
			if baseName(k) == want {
				content, ok = v, true
				break
			}
		}
	}
	if !ok {
		return nil, &resolverNotFoundError{uri: uri}
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

// baseName returns the final path segment, treating both '/' and '\' as
// separators so the comparison is OS-independent.
func baseName(s string) string {
	if i := strings.LastIndexAny(s, `/\`); i >= 0 {
		return s[i+1:]
	}
	return s
}

// resolverNotFoundError models a well-behaved URIResolver reporting a genuine
// not-found: per the demotable-miss contract it satisfies fs.ErrNotExist (via
// Unwrap), so a schema loader may demote an ABSENT optional schema. An
// opaque/ambiguous resolver error that does NOT satisfy fs.ErrNotExist is fatal.
type resolverNotFoundError struct {
	uri string
}

func (e *resolverNotFoundError) Error() string { return "not found: " + e.uri }

func (*resolverNotFoundError) Unwrap() error { return fs.ErrNotExist }

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

func TestDefaultDenyFS(t *testing.T) {
	// TestImportIncludeDefaultDeny verifies that xsl:import and xsl:include of a
	// local module fail to compile when no Compiler.URIResolver is configured
	// (filesystem access is opt-in), and succeed when a resolver is supplied.
	t.Run("xsl:import and xsl:include are denied by default", func(t *testing.T) {
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
	})

	// TestFnTransformStylesheetLocationDefaultDeny verifies that fn:transform with
	// a stylesheet-location denies loading when no compile-time URIResolver is
	// configured, and succeeds when one is.
	t.Run("an fn:transform stylesheet location is denied by default", func(t *testing.T) {
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
	})

	// TestStaticFnTransformHonorsCompilerCap verifies that the compile-time
	// fn:transform used for static="yes" variables respects Compiler.MaxResourceBytes.
	// Before the fix, the temporary stylesheet / transform context built for static
	// evaluation ignored the compiler cap, so an over-cap inner stylesheet loaded
	// regardless. The compiler cap must now bound the static transform() read: an
	// inner stylesheet larger than the cap is refused, surfacing
	// [xslt3.ErrResourceTooLarge].
	t.Run("a static fn:transform honors the compiler cap", func(t *testing.T) {
		const outerURI = "mem://stylesheets/outer.xsl"
		const innerURI = "mem://stylesheets/inner.xsl"

		innerSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><inner>transformed</inner></xsl:template>
</xsl:stylesheet>`

		// A static variable whose select calls transform() at compile time.
		outerSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
  <xsl:variable name="r" static="yes" select="transform(map{
    'stylesheet-location': 'inner.xsl',
    'delivery-format': 'serialized'
  })('output')"/>
  <xsl:template match="/"><out><xsl:value-of select="$r"/></out></xsl:template>
</xsl:stylesheet>`

		ctx := t.Context()
		resolver := fileMapResolver{files: map[string]string{innerURI: innerSrc}}

		// MaxResourceBytes(1): the inner stylesheet is far larger than 1 byte, so
		// the static transform() read is rejected.
		doc, err := helium.NewParser().Parse(ctx, []byte(outerSrc))
		require.NoError(t, err)
		ss, err := xslt3.NewCompiler().
			BaseURI(outerURI).
			URIResolver(resolver).
			MaxResourceBytes(1).
			Compile(ctx, doc)
		require.NoError(t, err)
		src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
		require.NoError(t, err)
		_, err = ss.Transform(src).Serialize(ctx)
		require.Error(t, err, "MaxResourceBytes(1) must reject the over-cap static transform read")
		require.ErrorIs(t, err, xslt3.ErrResourceTooLarge)

		// Sanity: with the default cap the same static transform succeeds.
		doc2, err := helium.NewParser().Parse(ctx, []byte(outerSrc))
		require.NoError(t, err)
		ss2, err := xslt3.NewCompiler().BaseURI(outerURI).URIResolver(resolver).Compile(ctx, doc2)
		require.NoError(t, err)
		src2, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
		require.NoError(t, err)
		out, err := ss2.Transform(src2).Serialize(ctx)
		require.NoError(t, err)
		require.Contains(t, out, "transformed")
	})

	// TestSourceDocumentDefaultDeny verifies xsl:source-document does NOT read the
	// host filesystem when no URIResolver is installed: even though the file
	// physically exists, retrieval must be denied with FODC0002.
	t.Run("a source document is denied by default", func(t *testing.T) {
		path := writeTempXML(t, `<data v="hello"/>`)

		ss := compileStylesheetString(t, strings.ReplaceAll(sourceDocStylesheet, "%HREF%", path))
		source := parseTransformSource(t)

		_, err := ss.Transform(source).Serialize(t.Context())
		require.Error(t, err, "source-document must default-deny without a URIResolver")
		require.Contains(t, err.Error(), "FODC0002")
	})

	// TestSourceDocumentRoutesThroughResolver verifies that with a recording
	// resolver installed, xsl:source-document retrieves its document through the
	// resolver (receiving the resolved URI), and never via os.ReadFile.
	t.Run("a source document routes through the resolver", func(t *testing.T) {
		path := writeTempXML(t, `<data v="hello"/>`)

		resolver := &recordingURIResolver{files: map[string][]byte{
			path: []byte(`<data v="hello"/>`),
		}}

		ss := compileStylesheetString(t, strings.ReplaceAll(sourceDocStylesheet, "%HREF%", path))
		source := parseTransformSource(t)

		result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, result, "<out>hello</out>")
		require.True(t, resolver.seen(path), "resolver should have been asked to resolve %q; got %v", path, resolver.requests)
	})

	// TestStreamAvailableDefaultDeny verifies fn:stream-available returns false
	// (stat-ing no host filesystem) when no URIResolver is installed,
	// even though the referenced file exists on disk.
	t.Run("stream-available is denied by default", func(t *testing.T) {
		path := writeTempXML(t, `<data/>`)

		ss := compileStylesheetString(t, strings.ReplaceAll(streamAvailableStylesheet, "%HREF%", path))
		source := parseTransformSource(t)

		result, err := ss.Transform(source).Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, result, "<out>false</out>", "stream-available must report false without a resolver")
	})

	// TestStreamAvailableRoutesThroughResolver verifies fn:stream-available probes
	// availability via the installed URIResolver and returns true for an XML
	// resource it can retrieve.
	t.Run("stream-available routes through the resolver", func(t *testing.T) {
		path := writeTempXML(t, `<data/>`)

		resolver := &recordingURIResolver{files: map[string][]byte{
			path: []byte(`<data/>`),
		}}

		ss := compileStylesheetString(t, strings.ReplaceAll(streamAvailableStylesheet, "%HREF%", path))
		source := parseTransformSource(t)

		result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, result, "<out>true</out>")
		require.True(t, resolver.seen(path), "resolver should have been asked to resolve %q; got %v", path, resolver.requests)
	})

	// TestMergeDefaultDeny verifies xsl:merge with for-each-source does NOT read
	// the host filesystem when no URIResolver is installed: even though the file
	// physically exists, retrieval must be denied with FODC0002.
	t.Run("xsl:merge is denied by default", func(t *testing.T) {
		path := writeTempXML(t, `<data><row k="a"/></data>`)

		ss := compileStylesheetString(t, strings.ReplaceAll(mergeStylesheet, "%HREF%", path))
		source := parseTransformSource(t)

		_, err := ss.Transform(source).Serialize(t.Context())
		require.Error(t, err, "xsl:merge must default-deny without a URIResolver")
		require.Contains(t, err.Error(), "FODC0002")
	})

	// TestMergeRoutesThroughResolver verifies that with a recording resolver
	// installed, xsl:merge retrieves its merge-source document through the
	// resolver (receiving the resolved URI), and never via os.ReadFile.
	t.Run("xsl:merge routes through the resolver", func(t *testing.T) {
		path := writeTempXML(t, `<data><row k="a"/></data>`)

		resolver := &recordingURIResolver{files: map[string][]byte{
			path: []byte(`<data><row k="a"/></data>`),
		}}

		ss := compileStylesheetString(t, strings.ReplaceAll(mergeStylesheet, "%HREF%", path))
		source := parseTransformSource(t)

		result, err := ss.Transform(source).URIResolver(resolver).Serialize(t.Context())
		require.NoError(t, err)
		require.Contains(t, result, "a")
		require.True(t, resolver.seen(path), "resolver should have been asked to resolve %q; got %v", path, resolver.requests)
	})
}

// recordingURIResolver records every URI it is asked to resolve and serves
// the bytes registered for that URI. It lets a test prove that a runtime XSLT
// instruction routed its retrieval through Invocation.URIResolver instead of
// reading the host filesystem directly.
type recordingURIResolver struct {
	mu       sync.Mutex
	requests []string
	files    map[string][]byte
}

func (r *recordingURIResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, uri)
	data, ok := r.files[uri]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (r *recordingURIResolver) seen(uri string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.requests, uri)
}

// writeTempXML writes an XML file into the test's temp dir and returns its
// absolute path.
func writeTempXML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.xml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

const sourceDocStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:source-document href="%HREF%">
      <out><xsl:value-of select="/data/@v"/></out>
    </xsl:source-document>
  </xsl:template>
</xsl:stylesheet>`

const streamAvailableStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out><xsl:value-of select="stream-available('%HREF%')"/></out>
  </xsl:template>
</xsl:stylesheet>`

const mergeStylesheet = `
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <out>
      <xsl:merge>
        <xsl:merge-source for-each-source="'%HREF%'" select="/data/row">
          <xsl:merge-key select="@k"/>
        </xsl:merge-source>
        <xsl:merge-action>
          <xsl:value-of select="current-merge-group()/@k"/>
        </xsl:merge-action>
      </xsl:merge>
    </out>
  </xsl:template>
</xsl:stylesheet>`

// injectionResolver serves a fixed set of URIs from an in-memory map.
type injectionResolver struct {
	files map[string]string
}

func (r *injectionResolver) Resolve(uri string) (io.ReadCloser, error) {
	body, ok := r.files[uri]
	if !ok {
		body, ok = r.files[filepath.FromSlash(uri)]
	}
	if !ok {
		return nil, io.EOF
	}
	return io.NopCloser(strings.NewReader(body)), nil
}

func (r *injectionResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	return r.Resolve(uri)
}

// TestParserInjectionGovernsRuntimeParse proves that a helium.Parser injected
// via Compiler.Parser governs the parse policy of the runtime document parse
// performed by fn:doc: a parser with MaxNameLength(8) rejects a document whose
// element name exceeds 8 bytes, while the default compiler accepts it.
func TestParserInjectionGovernsRuntimeParse(t *testing.T) {
	t.Parallel()

	const stylesheet = `<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:param name="url"/>
  <xsl:template match="/">
    <out><xsl:value-of select="doc($url)/*/local-name()"/></out>
  </xsl:template>
</xsl:stylesheet>`

	// The runtime document's root element name "longelementname" is 15 bytes,
	// exceeding MaxNameLength(8) but accepted by the default parser.
	const docBody = `<?xml version="1.0"?><longelementname/>`
	docPath := filepath.FromSlash("/docs/in.xml")

	resolver := &injectionResolver{files: map[string]string{docPath: docBody}}

	compileAndRun := func(t *testing.T, c xslt3.Compiler) (string, error) {
		t.Helper()
		ssDoc, err := helium.NewParser().Parse(t.Context(), []byte(stylesheet))
		require.NoError(t, err)
		ss, err := c.Compile(t.Context(), ssDoc)
		require.NoError(t, err)
		source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
		require.NoError(t, err)
		return ss.Transform(source).
			URIResolver(resolver).
			SetParameter("url", xpath3.SingleString(docPath)).
			Serialize(t.Context())
	}

	t.Run("default parser accepts long name", func(t *testing.T) {
		t.Parallel()
		out, err := compileAndRun(t, xslt3.NewCompiler())
		require.NoError(t, err)
		require.Contains(t, out, "longelementname")
	})

	t.Run("injected parser MaxNameLength enforced", func(t *testing.T) {
		t.Parallel()
		c := xslt3.NewCompiler().Parser(helium.NewParser().MaxNameLength(8))
		out, err := compileAndRun(t, c)
		// The injected limit must reach the runtime fn:doc parse: either the parse
		// fails outright, or the over-limit name never surfaces in the output.
		if err == nil {
			require.NotContains(t, out, "longelementname",
				"injected MaxNameLength must govern the runtime fn:doc parse")
		}
	})
}
