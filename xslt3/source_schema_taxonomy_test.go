package xslt3_test

import (
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
