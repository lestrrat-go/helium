package xslt3_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// The source-document schema-load caller (execute_transform.go) applies the same
// fetch/content/denial taxonomy under lax/default validation as the strict path:
// a genuine FETCH MISS is skippable (best-effort validation), but a CONTENT error
// (malformed XML / invalid XSD) or a POLICY denial (no URIResolver) is fatal even
// under lax. These tests pin that a broken or policy-denied source-schema hint is
// no longer masked.

// taxSourceSheet is a minimal, non-schema-aware stylesheet (no xsl:import-schema)
// whose default validation is lax. The source-schema load still runs from the
// source document's xsi:schemaLocation, so its failure classification is exercised.
const taxSourceSheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

// taxSource is a source document referencing a no-namespace schema by hint.
const taxSource = `<?xml version="1.0"?>
<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:noNamespaceSchemaLocation="s.xsd"/>`

func compileTaxSheet(t *testing.T) *xslt3.Stylesheet {
	t.Helper()
	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(taxSourceSheet))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI("mem://sheets/main.xsl").Compile(ctx, doc)
	require.NoError(t, err)
	return ss
}

// TestSourceSchemaLoadMalformedXMLFatalUnderLax verifies that a source
// xsi:schemaLocation resolving to MALFORMED XML is fatal even under lax/default
// validation — a fetched-but-unparseable schema is a content error, not a miss.
func TestSourceSchemaLoadMalformedXMLFatalUnderLax(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxSource))
	require.NoError(t, err)

	resolver := runtimeFileMapResolver{files: map[string]string{
		"mem://sheets/s.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element`, // truncated, not well-formed
	}}
	_, err = ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.Error(t, err, "malformed source-schema XML must be fatal under lax validation")
}

// TestSourceSchemaLoadInvalidXSDFatalUnderLax verifies that a source
// xsi:schemaLocation resolving to well-formed XML that is an INVALID schema is
// fatal even under lax/default validation.
func TestSourceSchemaLoadInvalidXSDFatalUnderLax(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxSource))
	require.NoError(t, err)

	// Well-formed XML, invalid XSD: an element referencing an undefined type and a
	// duplicate global element — the xsd compiler reports construction errors and
	// Compile returns ErrCompilationFailed.
	resolver := runtimeFileMapResolver{files: map[string]string{
		"mem://sheets/s.xsd": `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="xs:string"/>
  <xs:element name="a" type="xs:string"/>
</xs:schema>`,
	}}
	_, err = ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.Error(t, err, "invalid source-schema XSD must be fatal under lax validation")
}

// TestSourceSchemaLoadPolicyDenialFatalUnderLax verifies that a source
// xsi:schemaLocation with NO URIResolver configured is a default-deny policy
// denial, fatal even under lax/default validation — it must not be masked as a
// benign fetch miss.
func TestSourceSchemaLoadPolicyDenialFatalUnderLax(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxSource))
	require.NoError(t, err)

	// No URIResolver on the invocation: the schema-location fetch is refused.
	_, err = ss.Transform(src).Serialize(ctx)
	require.Error(t, err, "a no-resolver source-schema load must be fatal under lax validation")
	require.Contains(t, err.Error(), "no URIResolver configured",
		"the denial must surface as a policy error")
}

// TestSourceSchemaLoadFetchMissSkippedUnderLax verifies that a genuine fetch miss
// (a configured resolver that simply lacks the target) is best-effort skipped
// under lax/default validation — the transform still succeeds, since
// schemaLocation is only a hint.
func TestSourceSchemaLoadFetchMissSkippedUnderLax(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxSource))
	require.NoError(t, err)

	// Resolver present but does not serve s.xsd: an unresolvable hint, demotable.
	resolver := runtimeFileMapResolver{files: map[string]string{
		"mem://sheets/other.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"/>`,
	}}
	out, err := ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.NoError(t, err, "a genuine fetch miss must be skipped under lax validation")
	require.Contains(t, out, "out")
}

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

// opaqueRuntimeResolver is an xpath3.URIResolver whose ResolveURI returns an
// OPAQUE error (a bare "HTTP 403", NOT satisfying fs.ErrNotExist) for every URI.
// Per the demotable-miss contract such an error is FATAL, never demoted.
type opaqueRuntimeResolver struct{}

func (opaqueRuntimeResolver) ResolveURI(string) (io.ReadCloser, error) {
	return nil, errors.New("HTTP 403 Forbidden")
}

// TestSourceSchemaOpaqueResolverErrorFatalUnderLax verifies that an OPAQUE source
// schema-location resolver error (not fs.ErrNotExist) is FATAL even under lax —
// only a CONFIRMED not-found (fs.ErrNotExist / HTTP 404) is a demotable miss.
func TestSourceSchemaOpaqueResolverErrorFatalUnderLax(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxSource))
	require.NoError(t, err)

	_, err = ss.Transform(src).URIResolver(opaqueRuntimeResolver{}).Serialize(ctx)
	require.Error(t, err, "an opaque source-schema resolver error (not fs.ErrNotExist) must be fatal under lax")
}

// The source-document schema-load loop (source_schema.go loadSchemasFromSchemaLocation)
// aggregates over every xsi:schemaLocation entry: a demotable fetch miss on an
// earlier entry must NOT short-circuit the loop, so a LATER entry's authoritative
// content error stays fatal under lax and a LATER valid schema still loads. These
// tests pin that invariant.

// taxMultiSource references TWO no-namespace-mapped schemas via xsi:schemaLocation.
// The first (missing.xsd) is a genuine fetch miss; the second (s.xsd) is served.
const taxMultiSource = `<?xml version="1.0"?>
<doc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
     xsi:schemaLocation="urn:a missing.xsd urn:b s.xsd"/>`

// memSheetsSXSDURI is the resolved URI of the second (served) schema entry.
const memSheetsSXSDURI = "mem://sheets/s.xsd"

// TestSourceSchemaMultiEntryLaterMalformedFatal verifies that when an earlier
// xsi:schemaLocation entry is a genuine fetch miss and a LATER entry resolves to
// MALFORMED XML, the load is fatal even under lax/default validation — the earlier
// miss no longer masks the later content error.
func TestSourceSchemaMultiEntryLaterMalformedFatal(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxMultiSource))
	require.NoError(t, err)

	// missing.xsd is absent (fetch miss); s.xsd is truncated, not well-formed.
	resolver := runtimeFileMapResolver{files: map[string]string{
		memSheetsSXSDURI: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element`,
	}}
	_, err = ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.Error(t, err, "a later malformed source-schema entry must stay fatal after an earlier fetch miss")
}

// TestSourceSchemaMultiEntryLaterInvalidXSDFatal verifies the same for a LATER
// entry that is well-formed XML but an INVALID schema.
func TestSourceSchemaMultiEntryLaterInvalidXSDFatal(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxMultiSource))
	require.NoError(t, err)

	// missing.xsd is absent (fetch miss); s.xsd is valid XML but an invalid schema
	// (duplicate global element → xsd construction error).
	resolver := runtimeFileMapResolver{files: map[string]string{
		memSheetsSXSDURI: `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="xs:string"/>
  <xs:element name="a" type="xs:string"/>
</xs:schema>`,
	}}
	_, err = ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.Error(t, err, "a later invalid source-schema entry must stay fatal after an earlier fetch miss")
}

// TestSourceSchemaMultiEntryLaterValidLoads verifies that when an earlier entry is
// a pure fetch miss and a LATER entry is a VALID schema, the transform succeeds
// (the miss is demoted under lax) and the valid schema is loaded best-effort.
func TestSourceSchemaMultiEntryLaterValidLoads(t *testing.T) {
	ctx := t.Context()
	ss := compileTaxSheet(t)

	src, err := helium.NewParser().Parse(ctx, []byte(taxMultiSource))
	require.NoError(t, err)

	// missing.xsd is absent (fetch miss); s.xsd is a valid no-namespace schema.
	resolver := runtimeFileMapResolver{files: map[string]string{
		memSheetsSXSDURI: `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc" type="xs:string"/>
</xs:schema>`,
	}}
	out, err := ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.NoError(t, err, "a pure earlier miss is demoted and the later valid schema loads under lax")
	require.Contains(t, out, "out")
}
