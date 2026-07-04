package xslt3_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// These tests pin the POSITIVE-TAG discipline of the xslt3 schema loaders: the
// demotion decision keys on a CONFIRMED benign resolution miss (an unresolvable
// schemaLocation / HTTP 404), NOT on a negative "not-fatal → miss" inference. So a
// reader obtained then failing during Read (post-open), an HTTP 401/403/5xx, or
// any untagged/ambiguous error is FATAL even under lax validation, while a genuine
// resolution miss / HTTP 404 still proceeds (demoted) under lax. This mirrors the
// xsd nested classifier (readNestedSchema / errSchemaFetchMiss / notDemotable).

// badReadCloser is obtained successfully (models a resolved-and-opened resource)
// but fails during Read — a POST-OPEN read failure, not a resolution miss.
type badReadCloser struct{ err error }

func (b badReadCloser) Read([]byte) (int, error) { return 0, b.err }
func (badReadCloser) Close() error               { return nil }

// postOpenFailResolver resolves every URI to a reader that then fails during
// Read, so the failure is unambiguously post-open (the reader WAS obtained).
type postOpenFailResolver struct{ readErr error }

func (r postOpenFailResolver) ResolveURI(string) (io.ReadCloser, error) {
	return badReadCloser{r.readErr}, nil
}

// TestSourceSchemaPostOpenReadFailFatalUnderLax verifies that a source
// xsi:schemaLocation whose reader is OBTAINED then fails during Read is FATAL
// under lax/default validation — a post-open read failure is not a resolution
// miss and must never be demoted.
func TestSourceSchemaPostOpenReadFailFatalUnderLax(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxSource))
	require.NoError(t, err)

	_, err = ss.Transform(src).
		URIResolver(postOpenFailResolver{readErr: io.ErrUnexpectedEOF}).
		Serialize(ctx)
	require.Error(t, err, "a post-open source-schema read failure must be fatal under lax validation")
}

// runHTTPSourceSchema drives a source-document schema load whose
// xsi:noNamespaceSchemaLocation is an ABSOLUTE http URL served by a stub that
// replies with the given status, so the load dispatches to the HTTPClient path
// (fetchHTTPBytes). It returns the transform error under lax validation.
func runHTTPSourceSchema(t *testing.T, status int) error {
	t.Helper()
	ctx := t.Context()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}))
	defer srv.Close()

	ss := compileTaxSheet(t)
	srcXML := `<?xml version="1.0"?>
<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:noNamespaceSchemaLocation="` + srv.URL + `/s.xsd"/>`
	src, err := helium.NewParser().Parse(ctx, []byte(srcXML))
	require.NoError(t, err)

	_, err = ss.Transform(src).HTTPClient(srv.Client()).Serialize(ctx)
	return err
}

// TestSourceSchemaHTTP403FatalUnderLax verifies an HTTP 403 (Forbidden) source
// schema fetch is FATAL under lax — a forbidden authoritative schema-location is
// not a benign miss.
func TestSourceSchemaHTTP403FatalUnderLax(t *testing.T) {
	err := runHTTPSourceSchema(t, http.StatusForbidden)
	require.Error(t, err, "an HTTP 403 source-schema fetch must be fatal under lax validation")
}

// TestSourceSchemaHTTP500FatalUnderLax verifies an HTTP 500 (server error) source
// schema fetch is FATAL under lax — a server error is not a confirmed not-found.
func TestSourceSchemaHTTP500FatalUnderLax(t *testing.T) {
	err := runHTTPSourceSchema(t, http.StatusInternalServerError)
	require.Error(t, err, "an HTTP 500 source-schema fetch must be fatal under lax validation")
}

// TestSourceSchemaHTTP404SkippedUnderLax verifies an HTTP 404 (Not Found) source
// schema fetch is a CONFIRMED resolution miss, demoted under lax so the transform
// still succeeds (schemaLocation is only a hint).
func TestSourceSchemaHTTP404SkippedUnderLax(t *testing.T) {
	ctx := t.Context()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ss := compileTaxSheet(t)
	srcXML := `<?xml version="1.0"?>
<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:noNamespaceSchemaLocation="` + srv.URL + `/s.xsd"/>`
	src, err := helium.NewParser().Parse(ctx, []byte(srcXML))
	require.NoError(t, err)

	out, err := ss.Transform(src).HTTPClient(srv.Client()).Serialize(ctx)
	require.NoError(t, err, "an HTTP 404 source-schema fetch is a resolution miss, demoted under lax")
	require.Contains(t, out, "out")
}

// strictSchemaSheet is a schema-aware stylesheet with default-validation="strict".
// Its inline import-schema declares a no-namespace element "out" of type
// xs:string so the LRE result <out>ok</out> validates, isolating the
// source-schema-miss classification as the only remaining failure source.
const strictSchemaSheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" default-validation="strict"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:import-schema>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
      <xs:element name="out" type="xs:string"/>
    </xs:schema>
  </xsl:import-schema>
  <xsl:template match="/"><out>ok</out></xsl:template>
</xsl:stylesheet>`

// TestSourceSchemaStrictFatalOnMiss verifies that under strict validation even a
// GENUINE resolution miss (the demotable case under lax) is fatal — strict never
// demotes a source-schema load failure.
func TestSourceSchemaStrictFatalOnMiss(t *testing.T) {
	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(strictSchemaSheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI("mem://sheets/main.xsl").Compile(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(taxSource))
	require.NoError(t, err)

	// Resolver present but lacking s.xsd: a genuine resolution miss, demotable
	// under lax but FATAL under strict.
	resolver := runtimeFileMapResolver{files: map[string]string{
		"mem://sheets/other.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"/>`,
	}}
	_, err = ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.Error(t, err, "a source-schema miss must be fatal under strict validation")
}
