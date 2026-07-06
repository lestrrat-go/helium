package xslt3_test

import (
	"errors"
	"io"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// These regressions pin the top-level xsl:import-schema fail-closed taxonomy:
// a schema-location that is FETCHED but whose CONTENT is unusable (malformed
// XML, or a well-formed-but-invalid XSD) must fail compilation fatally and must
// NOT be masked by a matching pre-compiled Compiler.ImportSchemas entry — only a
// genuine FETCH MISS (the location could not be loaded) may fall back. This is
// the same fetch-miss / content / denial classification the nested-schema path
// uses.

const isNS = "http://example.com/s"

// isValidPrecompiledSchema is a small, valid schema for isNS, registered as the
// pre-compiled fallback. If content-error masking regressed, compilation would
// silently succeed against this schema instead of failing.
const isValidPrecompiledSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="` + isNS + `"
           elementFormDefault="qualified">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

const isValidOtherNamespaceSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/other"
           elementFormDefault="qualified">
  <xs:element name="other" type="xs:string"/>
</xs:schema>`

const isImportSchemaStylesheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="` + isNS + `">
  <xsl:import-schema namespace="` + isNS + `" schema-location="s.xsd"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

func isPrecompiledSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	preDoc, err := helium.NewParser().Parse(t.Context(), []byte(isValidPrecompiledSchema))
	require.NoError(t, err)
	precompiled, err := xsd.NewCompiler().Compile(t.Context(), preDoc)
	require.NoError(t, err)
	return precompiled
}

// A schema-location that is fetched but whose bytes are MALFORMED XML must FAIL
// compilation fatally — a post-fetch content error must not be papered over by a
// registered pre-compiled schema for the same namespace.
func TestImportSchemaMalformedXMLNotMaskedByPrecompiledFallback(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"
	const schemaURI = "mem:/stylesheets/s.xsd"

	// Well-formed enough to fetch, but not well-formed XML: unclosed root.
	const malformed = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + isNS + `">
  <xs:element name="root" type="xs:string"/>`

	ctx := t.Context()
	resolver := fileMapResolver{files: map[string]string{schemaURI: malformed}}
	doc, err := helium.NewParser().Parse(ctx, []byte(isImportSchemaStylesheet))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		ImportSchemas(isPrecompiledSchema(t)).
		Compile(ctx, doc)
	require.Error(t, err,
		"a fetched-but-malformed schema-location must fail even with a pre-compiled fallback registered")
	require.Contains(t, err.Error(), "cannot",
		"the content error must surface, not a silent fallback to ImportSchemas")
}

// A schema-location that is fetched and is well-formed XML but an INVALID XSD
// (here: a top-level element referencing an undefined type — a schema
// construction failure) must FAIL compilation fatally and must not be masked by
// a registered pre-compiled schema for the same namespace.
func TestImportSchemaInvalidXSDNotMaskedByPrecompiledFallback(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"
	const schemaURI = "mem:/stylesheets/s.xsd"

	// Well-formed XML, but the referenced type s:missing is never declared, so
	// schema construction fails (ErrCompilationFailed / XTSE0220).
	const invalidXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           xmlns:s="` + isNS + `"
           targetNamespace="` + isNS + `"
           elementFormDefault="qualified">
  <xs:element name="root" type="s:missing"/>
</xs:schema>`

	ctx := t.Context()
	resolver := fileMapResolver{files: map[string]string{schemaURI: invalidXSD}}
	doc, err := helium.NewParser().Parse(ctx, []byte(isImportSchemaStylesheet))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		ImportSchemas(isPrecompiledSchema(t)).
		Compile(ctx, doc)
	require.Error(t, err,
		"a fetched-but-invalid XSD schema-location must fail even with a pre-compiled fallback registered")
}

// A top-level complexType missing its @name is an invalid XSD (schema
// representation error). Fetched but content-invalid, it must fail fatally, not
// fall back to the pre-compiled schema.
func TestImportSchemaMissingComplexTypeNameNotMaskedByPrecompiledFallback(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"
	const schemaURI = "mem:/stylesheets/s.xsd"

	const invalidXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="` + isNS + `"
           elementFormDefault="qualified">
  <xs:complexType>
    <xs:sequence/>
  </xs:complexType>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

	ctx := t.Context()
	resolver := fileMapResolver{files: map[string]string{schemaURI: invalidXSD}}
	doc, err := helium.NewParser().Parse(ctx, []byte(isImportSchemaStylesheet))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		ImportSchemas(isPrecompiledSchema(t)).
		Compile(ctx, doc)
	require.Error(t, err,
		"a fetched schema whose top-level complexType lacks @name must fail even with a pre-compiled fallback registered")
}

// A schema-location that is a genuine FETCH MISS (the resolver has no such
// resource) with a matching pre-compiled Compiler.ImportSchemas entry still
// falls back cleanly: compilation succeeds and the stylesheet transforms. This
// is the ONE benign case the fail-closed taxonomy preserves.
func TestImportSchemaFetchMissFallsBackToPrecompiled(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"

	ctx := t.Context()
	// The resolver serves NOTHING for s.xsd (a genuine fetch miss), but a
	// pre-compiled schema for the namespace is registered.
	resolver := fileMapResolver{files: map[string]string{}}
	doc, err := helium.NewParser().Parse(ctx, []byte(isImportSchemaStylesheet))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		ImportSchemas(isPrecompiledSchema(t)).
		Compile(ctx, doc)
	require.NoError(t, err,
		"a genuine fetch miss must fall back to the pre-compiled ImportSchemas entry")

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// A schema-location that is fetched and compiled, but whose target namespace
// does not satisfy the xsl:import-schema/@namespace declaration, may still use
// a registered pre-compiled schema for the requested namespace. The fetched
// schema was content-valid, so this does not mask a malformed/invalid schema;
// it only declines to use a valid schema for the wrong namespace.
func TestImportSchemaNamespaceMismatchFallsBackToPrecompiled(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"
	const schemaURI = "mem:/stylesheets/s.xsd"

	ctx := t.Context()
	resolver := fileMapResolver{files: map[string]string{schemaURI: isValidOtherNamespaceSchema}}
	doc, err := helium.NewParser().Parse(ctx, []byte(isImportSchemaStylesheet))
	require.NoError(t, err)

	ss, err := xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		ImportSchemas(isPrecompiledSchema(t)).
		Compile(ctx, doc)
	require.NoError(t, err,
		"a content-valid schema-location for another namespace should not block a matching pre-compiled schema")

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// If a content-valid schema-location names the wrong target namespace and no
// pre-compiled schema satisfies the requested namespace, the original namespace
// mismatch remains a static error.
func TestImportSchemaNamespaceMismatchWithoutPrecompiledErrors(t *testing.T) {
	t.Parallel()

	const baseURI = "mem://stylesheets/main.xsl"
	const schemaURI = "mem:/stylesheets/s.xsd"

	ctx := t.Context()
	resolver := fileMapResolver{files: map[string]string{schemaURI: isValidOtherNamespaceSchema}}
	doc, err := helium.NewParser().Parse(ctx, []byte(isImportSchemaStylesheet))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().
		BaseURI(baseURI).
		URIResolver(resolver).
		Compile(ctx, doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match schema targetNamespace")
}

// opaqueResolveErrorResolver is a compile-time URIResolver whose Resolve returns
// an OPAQUE error (a bare "HTTP 403", NOT satisfying fs.ErrNotExist) for every
// URI — modeling a resolver that could not fetch for a reason OTHER than a
// confirmed not-found. Per the demotable-miss contract such an error is fatal.
type opaqueResolveErrorResolver struct{}

func (opaqueResolveErrorResolver) Resolve(string) (io.ReadCloser, error) {
	return nil, errors.New("HTTP 403 Forbidden")
}

// An OPAQUE resolver error (not fs.ErrNotExist) on the top-level import-schema
// path is FATAL and must NOT fall back to a matching pre-compiled ImportSchemas
// entry — only a CONFIRMED not-found (fs.ErrNotExist) is a demotable miss.
func TestImportSchemaOpaqueResolverErrorNotMaskedByPrecompiledFallback(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(isImportSchemaStylesheet))
	require.NoError(t, err)

	_, err = xslt3.NewCompiler().
		BaseURI("mem://stylesheets/main.xsl").
		URIResolver(opaqueResolveErrorResolver{}).
		ImportSchemas(isPrecompiledSchema(t)).
		Compile(ctx, doc)
	require.Error(t, err,
		"an opaque resolver error (not fs.ErrNotExist) must be fatal, not masked by the precompiled fallback")
}
