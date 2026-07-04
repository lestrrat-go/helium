package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

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
