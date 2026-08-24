package xslt3_test

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

// exactURIResolver is a compile-time URIResolver (method Resolve) that serves
// content ONLY for the exact URI keys it is given — no base-name fallback. It
// records every URI it is asked for so a test can assert the precise canonical
// nested URI the schema loader requests.
type exactURIResolver struct {
	files map[string]string
	mu    sync.Mutex
	asked []string
}

func (r *exactURIResolver) Resolve(uri string) (io.ReadCloser, error) {
	r.mu.Lock()
	r.asked = append(r.asked, uri)
	r.mu.Unlock()
	content, ok := r.files[uri]
	if !ok {
		return nil, &resolverNotFoundError{uri: uri}
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (r *exactURIResolver) askedFor(uri string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.asked, uri)
}

// exactRuntimeURIResolver mirrors exactURIResolver but exposes the runtime
// xpath3.URIResolver shape (method ResolveURI).
type exactRuntimeURIResolver struct {
	files map[string]string
	mu    sync.Mutex
	asked []string
}

func (r *exactRuntimeURIResolver) ResolveURI(uri string) (io.ReadCloser, error) {
	r.mu.Lock()
	r.asked = append(r.asked, uri)
	r.mu.Unlock()
	content, ok := r.files[uri]
	if !ok {
		return nil, &resolverNotFoundError{uri: uri}
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (r *exactRuntimeURIResolver) askedFor(uri string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.asked, uri)
}

// TestImportSchemaNestedIncludeAbsoluteURLBase verifies that a compile-time
// xsl:import-schema whose schema-location is an absolute https:// URL routes
// the nested xs:include through the resolver under its CORRECT canonical URI
// (https://example.com/s/part.xsd) — not a filepath-collapsed
// https:/example.com/s/part.xsd.
func TestImportSchemaNestedIncludeAbsoluteURLBase(t *testing.T) {
	const baseURI = "https://example.com/style/main.xsl"
	const mainSchemaURI = "https://example.com/s/main.xsd"
	const partSchemaURI = "https://example.com/s/part.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="https://example.com/s/main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	resolver := &exactURIResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithInclude,
		partSchemaURI: ddPartSchemaXSD,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err, "nested xs:include over an absolute URL base must resolve through the resolver")
	require.True(t, resolver.askedFor(partSchemaURI),
		"resolver must be asked for the canonical nested URI %q; got %v", partSchemaURI, resolver.asked)

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// ddMainSchemaWithAbsoluteCrossHostInclude includes a part schema by an
// ABSOLUTE URI on a DIFFERENT host than the main schema. The xsd compiler must
// pass this absolute schema-location to the resolver UNCHANGED — never
// filepath.Join it onto the base (which would collapse "//" and drop the host,
// yielding the malformed "https:/cdn.example.com/part.xsd").
const ddMainSchemaWithAbsoluteCrossHostInclude = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:include schemaLocation="https://cdn.example.com/part.xsd"/>
  <xs:element name="root" type="s:rootType"/>
</xs:schema>`

// TestImportSchemaNestedIncludeAbsoluteCrossHost verifies that an absolute
// cross-host xs:include is requested from the resolver under its exact
// canonical URI, not a filepath-collapsed (host-dropped) spelling.
func TestImportSchemaNestedIncludeAbsoluteCrossHost(t *testing.T) {
	const baseURI = "https://example.com/style/main.xsl"
	const mainSchemaURI = "https://example.com/s/main.xsd"
	const partSchemaURI = "https://cdn.example.com/part.xsd"
	const collapsedURI = "https:/cdn.example.com/part.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="https://example.com/s/main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	resolver := &exactURIResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithAbsoluteCrossHostInclude,
		partSchemaURI: ddPartSchemaXSD,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err, "absolute cross-host xs:include must resolve through the resolver")
	require.True(t, resolver.askedFor(partSchemaURI),
		"resolver must be asked for the canonical cross-host URI %q; got %v", partSchemaURI, resolver.asked)
	require.False(t, resolver.askedFor(collapsedURI),
		"resolver must NOT be asked for the collapsed host-dropped URI %q; got %v", collapsedURI, resolver.asked)

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// TestImportSchemaNestedIncludeFileURIBase verifies that a file:// base URI is
// NOT corrupted by the round-2 string-repair (which turned file:/tmp/... into
// file://tmp/..., reinterpreting "tmp" as a URI host). The nested xs:include
// must resolve to the canonical three-slash file:///tmp/s/part.xsd.
func TestImportSchemaNestedIncludeFileURIBase(t *testing.T) {
	const baseURI = "file:///tmp/s/main.xsl"
	const mainSchemaURI = "file:///tmp/s/main.xsd"
	const partSchemaURI = "file:///tmp/s/part.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	resolver := &exactURIResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithInclude,
		partSchemaURI: ddPartSchemaXSD,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err, "nested xs:include over a file:// base must resolve through the resolver")
	require.True(t, resolver.askedFor(mainSchemaURI),
		"resolver must be asked for the canonical main URI %q; got %v", mainSchemaURI, resolver.asked)
	require.True(t, resolver.askedFor(partSchemaURI),
		"resolver must be asked for the canonical nested URI %q (three slashes, not file://tmp/...); got %v", partSchemaURI, resolver.asked)

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// TestImportSchemaNestedIncludeMemURIBase verifies that a compile-time
// xsl:import-schema whose schema-location is a no-authority single-slash
// "mem:/..." URI routes its nested relative xs:include through the resolver
// under the CANONICAL "mem:/schemas/part.xsd" — NOT the "mem:///schemas/..."
// form that net/url's ResolveReference would emit by dropping the base's
// OmitHost flag. An exact-match resolver keyed on the canonical URI must be
// asked for it.
func TestImportSchemaNestedIncludeMemURIBase(t *testing.T) {
	const mainSchemaURI = "mem:/schemas/main.xsd"
	const partSchemaURI = "mem:/schemas/part.xsd"
	const collapsedURI = "mem:///schemas/part.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="mem:/schemas/main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	resolver := &exactURIResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithInclude,
		partSchemaURI: ddPartSchemaXSD,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err, "nested xs:include over a mem:/ base must resolve through the resolver")
	require.True(t, resolver.askedFor(partSchemaURI),
		"resolver must be asked for the canonical mem:/ nested URI %q; got %v", partSchemaURI, resolver.asked)
	require.False(t, resolver.askedFor(collapsedURI),
		"resolver must NOT be asked for the OmitHost-dropped URI %q; got %v", collapsedURI, resolver.asked)

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// ddMainSchemaWithSubdirInclude pulls in a part from a subdirectory, exercising
// multi-segment relative nested-include resolution. (The xsd compiler forbids a
// nested reference that climbs above its schema's own directory via "../", by
// design, as a path-traversal guard; the pure "../" URI rule is covered by the
// resolveSchemaURI unit test below.)
const ddMainSchemaWithSubdirInclude = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:include schemaLocation="sub/part.xsd"/>
  <xs:element name="root" type="s:rootType"/>
</xs:schema>`

const ddSubdirPartSchemaXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:simpleType name="rootType">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`

// TestImportSchemaNestedIncludeSubdirRelative verifies that a relative nested
// include into a subdirectory resolves to its canonical URL per RFC 3986 rules
// — the resolver is asked for the precise nested URI, not a filepath-collapsed
// spelling.
func TestImportSchemaNestedIncludeSubdirRelative(t *testing.T) {
	const baseURI = "https://example.com/style/main.xsl"
	const mainSchemaURI = "https://example.com/s/main.xsd"
	const partSchemaURI = "https://example.com/s/sub/part.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="https://example.com/s/main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	resolver := &exactURIResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithSubdirInclude,
		partSchemaURI: ddSubdirPartSchemaXSD,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err, "a relative subdir nested include must resolve via URI rules")
	require.True(t, resolver.askedFor(partSchemaURI),
		"resolver must be asked for the subdir URI %q; got %v", partSchemaURI, resolver.asked)

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// TestImportSchemaXMLBaseFileURIBase verifies that an xml:base attribute on the
// xsl:import-schema element is folded into the effective base URI without
// collapsing a canonical file:/// base to a bare local path. With base URI
// "file:///tmp/styles/main.xsl" and xml:base="schemas/", the schema-location
// "main.xsd" must resolve to "file:///tmp/styles/schemas/main.xsd" — not the
// scheme-dropped "/tmp/styles/schemas/main.xsd" that helium.BuildURI produced
// by filepath.Join'ing the file: scheme away.
func TestImportSchemaXMLBaseFileURIBase(t *testing.T) {
	const baseURI = "file:///tmp/styles/main.xsl"
	const wantSchemaURI = "file:///tmp/styles/schemas/main.xsd"
	const collapsedURI = "/tmp/styles/schemas/main.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xml="http://www.w3.org/XML/1998/namespace"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="main.xsd" xml:base="schemas/"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	resolver := &exactURIResolver{files: map[string]string{
		wantSchemaURI: ddSelfContainedSchema,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err, "xml:base folded over a file:/// base must keep the file: scheme so the resolver can serve it")
	require.True(t, resolver.askedFor(wantSchemaURI),
		"resolver must be asked for the canonical file:/// URI %q; got %v", wantSchemaURI, resolver.asked)
	require.False(t, resolver.askedFor(collapsedURI),
		"resolver must NOT be asked for the scheme-dropped local path %q; got %v", collapsedURI, resolver.asked)

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// ddMainSchemaMissingType references a type ("s:rootType") that is never
// declared (the would-be nested include is absent), so the xsd compiler cannot
// resolve it. The compiler reports a fatal unresolved-type diagnostic and
// installs a recovery placeholder; the file-backed xsl:import-schema path must
// surface this as XTSE0220, succeeding with no invalid
// schema.
const ddMainSchemaMissingType = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:element name="root" type="s:rootType"/>
</xs:schema>`

// TestImportSchemaFileBackedMissingTypeIsFatal verifies that a file-backed
// xsl:import-schema whose schema references an undeclared type fails the
// stylesheet compile with XTSE0220, instead of discarding the xsd compiler's
// fatal unresolved-type diagnostic and compiling successfully with a recovery
// placeholder.
func TestImportSchemaFileBackedMissingTypeIsFatal(t *testing.T) {
	const baseURI = "https://example.com/style/main.xsl"
	const mainSchemaURI = "https://example.com/s/main.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="https://example.com/s/main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	resolver := &exactURIResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaMissingType,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.Error(t, err, "file-backed import-schema with an unresolved referenced type must fail compilation")
	require.Contains(t, err.Error(), "XTSE0220",
		"fatal schema-construction error must surface as XTSE0220; got %v", err)
}

// ddSelfContainedSchema is a single-file schema with no nested include, used to
// exercise top-level schema-location resolution in isolation.
const ddSelfContainedSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

// TestImportSchemaAbsoluteOpaqueURILocalBase verifies that a compile-time
// xsl:import-schema whose schema-location is an absolute URI WITHOUT a "//"
// authority (opaque or single-slash: mem:/..., urn:..., file:/...) is passed to
// the resolver VERBATIM even when the stylesheet base is a LOCAL filesystem
// path. The buggy "://"-only detection treated these as relative and
// filepath-joined them onto the local base (e.g. "/work/mem:/schemas/s.xsd"),
// so the resolver was asked for the wrong URI and the load failed.
func TestImportSchemaAbsoluteOpaqueURILocalBase(t *testing.T) {
	for _, tc := range []struct {
		name      string
		schemaLoc string
	}{
		{"mem single-slash", "mem:/schemas/s.xsd"},
		{"urn opaque", "urn:schemas:s"},
		{"file single-slash", "file:/tmp/s.xsd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const baseURI = "/work/main.xsl"

			mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="` + tc.schemaLoc + `"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

			ctx := t.Context()

			resolver := &exactURIResolver{files: map[string]string{
				tc.schemaLoc: ddSelfContainedSchema,
			}}
			doc, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
			require.NoError(t, err)
			ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
			require.NoError(t, err, "absolute opaque/single-slash URI schema-location must resolve through the resolver verbatim")
			require.True(t, resolver.askedFor(tc.schemaLoc),
				"resolver must be asked for the exact URI %q (not filepath-joined onto the local base); got %v", tc.schemaLoc, resolver.asked)
			require.False(t, resolver.askedFor(filepath.Join("/work", tc.schemaLoc)),
				"resolver must NOT be asked for a filepath-joined URI; got %v", resolver.asked)

			src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
			require.NoError(t, err)
			out, err := ss.Transform(src).Serialize(ctx)
			require.NoError(t, err)
			require.Contains(t, out, "out")
		})
	}
}

// TestSourceSchemaLocationAbsoluteOpaqueURILocalBase is the runtime
// (xsi:schemaLocation) analogue: an absolute opaque/single-slash URI in the
// source document's xsi:schemaLocation, with a LOCAL source base URI, must be
// requested from the invocation resolver verbatim — never filepath-joined.
func TestSourceSchemaLocationAbsoluteOpaqueURILocalBase(t *testing.T) {
	const sourceURI = "/work/input.xml"
	const schemaLoc = "mem:/schemas/s.xsd"

	styleSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s"
    default-validation="strict">
  <xsl:import-schema namespace="http://example.com/s"/>
  <xsl:template match="/">
    <s:root>text</s:root>
  </xsl:template>
</xsl:stylesheet>`

	sourceSrc := `<?xml version="1.0"?>
<root xmlns="http://example.com/s"
      xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
      xsi:schemaLocation="http://example.com/s mem:/schemas/s.xsd">text</root>`

	ctx := t.Context()

	ssDoc, err := helium.NewParser().Parse(ctx, []byte(styleSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
	require.NoError(t, err)

	resolver := &exactRuntimeURIResolver{files: map[string]string{
		schemaLoc: ddSelfContainedSchema,
	}}
	src, err := helium.NewParser().Parse(ctx, []byte(sourceSrc))
	require.NoError(t, err)
	src.SetURL(sourceURI)
	out, err := ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.NoError(t, err, "runtime absolute opaque URI schema-location over a local base must resolve through the resolver verbatim")
	require.True(t, resolver.askedFor(schemaLoc),
		"resolver must be asked for the exact URI %q; got %v", schemaLoc, resolver.asked)
	require.False(t, resolver.askedFor(filepath.Join("/work", schemaLoc)),
		"resolver must NOT be asked for a filepath-joined URI; got %v", resolver.asked)
	require.Contains(t, out, "root")
}

// TestSourceSchemaLocationNestedIncludeAbsoluteURLBase verifies the runtime
// (xsi:schemaLocation) path resolves the nested xs:include over an absolute
// URL base to its canonical URI through the invocation resolver.
func TestSourceSchemaLocationNestedIncludeAbsoluteURLBase(t *testing.T) {
	const sourceURI = "https://example.com/docs/input.xml"
	const mainSchemaURI = "https://example.com/s/main.xsd"
	const partSchemaURI = "https://example.com/s/part.xsd"

	styleSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s"
    default-validation="strict">
  <xsl:import-schema namespace="http://example.com/s"/>
  <xsl:template match="/">
    <s:root>text</s:root>
  </xsl:template>
</xsl:stylesheet>`

	sourceSrc := `<?xml version="1.0"?>
<root xmlns="http://example.com/s"
      xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
      xsi:schemaLocation="http://example.com/s https://example.com/s/main.xsd">text</root>`

	ctx := t.Context()

	ssDoc, err := helium.NewParser().Parse(ctx, []byte(styleSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
	require.NoError(t, err)

	resolver := &exactRuntimeURIResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithInclude,
		partSchemaURI: ddPartSchemaXSD,
	}}
	src, err := helium.NewParser().Parse(ctx, []byte(sourceSrc))
	require.NoError(t, err)
	src.SetURL(sourceURI)
	out, err := ss.Transform(src).URIResolver(resolver).Serialize(ctx)
	require.NoError(t, err, "runtime nested xs:include over an absolute URL base must resolve through the resolver")
	require.True(t, resolver.askedFor(partSchemaURI),
		"resolver must be asked for the canonical nested URI %q; got %v", partSchemaURI, resolver.asked)
	require.Contains(t, out, "root")
}

// runtimeFileMapResolver is an xpath3.URIResolver (method ResolveURI) serving
// content from an in-memory map keyed by URI, with base-name fallback so the
// test does not depend on the exact resolved-URI spelling.
type runtimeFileMapResolver struct {
	files map[string]string
}

func (r runtimeFileMapResolver) ResolveURI(uri string) (io.ReadCloser, error) {
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

const ddSchemaXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

// TestImportSchemaDefaultDeny verifies that xsl:import-schema with a
// schema-location refuses to read the schema file when no Compiler.URIResolver
// is configured (no implicit os.ReadFile), and loads it when one is supplied.
func TestImportSchemaDefaultDeny(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"
	const schemaURI = "mem:/stylesheets/s.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="s.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	// Without a resolver: default-deny. The import-schema must not read s.xsd
	// off the local filesystem.
	docDeny, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().BaseURI(baseURI).Compile(ctx, docDeny)
	require.Error(t, err, "import-schema must fail without a URIResolver")
	require.Contains(t, err.Error(), "no URIResolver configured",
		"error should explain that filesystem access is opt-in")

	// With a resolver: success.
	resolver := fileMapResolver{files: map[string]string{
		schemaURI: ddSchemaXSD,
	}}
	docAllow, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, docAllow)
	require.NoError(t, err, "import-schema must succeed with a URIResolver")

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// ddMainSchemaWithInclude is a main schema that pulls in part.xsd via
// xs:include. The nested include must be resolved through the same resolver
// that supplied the main schema, not via the xsd compiler's default os.Open.
const ddMainSchemaWithInclude = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:include schemaLocation="part.xsd"/>
  <xs:element name="root" type="s:rootType"/>
</xs:schema>`

const ddPartSchemaXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:simpleType name="rootType">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`

// TestImportSchemaNestedIncludeThroughResolver verifies that a top-level
// xsl:import-schema fetched through the compile-time URIResolver routes its
// nested xs:include through the SAME resolver, instead of falling back to the
// xsd compiler's default os.Open (which would bypass the default-deny policy
// and fail for in-memory/HTTP-backed schemas).
func TestImportSchemaNestedIncludeThroughResolver(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"
	const mainSchemaURI = "mem:/stylesheets/main.xsd"
	const partSchemaURI = "mem:/stylesheets/part.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	// The resolver supplies BOTH the main schema and the nested part. The
	// nested xs:include must resolve through it; only then does the main
	// schema compile (s:rootType is defined in part.xsd).
	resolver := fileMapResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithInclude,
		partSchemaURI: ddPartSchemaXSD,
	}}
	docAllow, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, docAllow)
	require.NoError(t, err, "nested xs:include must resolve through the compile-time resolver")

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// TestImportSchemaNestedIncludeDenied verifies that when the resolver supplies
// the main schema but NOT its nested include, the nested xs:include is denied
// through the resolver, and never read off the local filesystem.
func TestImportSchemaNestedIncludeDenied(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"
	const mainSchemaURI = "mem:/stylesheets/main.xsd"

	mainSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="main.xsd"/>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	ctx := t.Context()

	// Only the main schema is resolvable; part.xsd is not. The nested include
	// must fail through the resolver (s:rootType stays unresolved) — it must
	// not silently succeed by reading part.xsd from disk.
	resolver := fileMapResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithInclude,
	}}
	docDeny, err := helium.NewParser().Parse(ctx, []byte(mainSrc))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, docDeny)
	require.Error(t, err, "nested xs:include must be denied when the resolver does not supply it")
}

// ddInlineMainXSL builds a stylesheet whose xsl:import-schema contains an INLINE
// xs:schema with a nested xs:include="part.xsd". The inline schema references
// s:rootType, which is only defined in part.xsd, so the include must resolve
// for compilation to succeed.
func ddInlineMainXSL() string {
	return `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s">
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
               targetNamespace="http://example.com/s"
               xmlns:s="http://example.com/s"
               elementFormDefault="qualified">
      <xs:include schemaLocation="part.xsd"/>
      <xs:element name="root" type="s:rootType"/>
    </xs:schema>
  </xsl:import-schema>
  <xsl:template match="/">
    <out/>
  </xsl:template>
</xsl:stylesheet>`
}

// TestImportSchemaInlineNestedIncludeThroughResolver verifies that an INLINE
// xsl:import-schema whose inline xs:schema contains a nested xs:include routes
// that include through the compile-time URIResolver — the same default-deny FS
// the schema-location path uses — instead of the xsd compiler's default os.Open.
func TestImportSchemaInlineNestedIncludeThroughResolver(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"
	const partSchemaURI = "mem:/stylesheets/part.xsd"

	ctx := t.Context()

	// The resolver supplies the nested part.xsd; the inline schema bytes are
	// already in-memory. Only then does s:rootType resolve and compilation pass.
	resolver := fileMapResolver{files: map[string]string{
		partSchemaURI: ddPartSchemaXSD,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(ddInlineMainXSL()))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.NoError(t, err, "inline schema's nested xs:include must resolve through the compile-time resolver")

	src, err := helium.NewParser().Parse(ctx, []byte(`<dummy/>`))
	require.NoError(t, err)
	out, err := ss.Transform(src).Serialize(ctx)
	require.NoError(t, err)
	require.Contains(t, out, "out")
}

// TestImportSchemaInlineNestedIncludeDefaultDeny verifies that an INLINE
// xsl:import-schema whose inline xs:schema contains a nested xs:include does NOT
// read that include off the local filesystem when no URIResolver is configured.
// The nested load must hit the default-deny error, not os.Open.
func TestImportSchemaInlineNestedIncludeDefaultDeny(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"

	ctx := t.Context()

	doc, err := helium.NewParser().Parse(ctx, []byte(ddInlineMainXSL()))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().BaseURI(baseURI).Compile(ctx, doc)
	require.Error(t, err, "inline schema's nested xs:include must be denied without a URIResolver")
	require.Contains(t, err.Error(), "no URIResolver configured",
		"error should explain that filesystem access is opt-in")
}

// TestImportSchemaInlineNestedIncludeDenied verifies that when a resolver is
// configured but does NOT supply the nested include, the inline schema's
// xs:include is denied through the resolver, and never read off disk.
func TestImportSchemaInlineNestedIncludeDenied(t *testing.T) {
	const baseURI = "mem://stylesheets/main.xsl"

	ctx := t.Context()

	// Resolver has no part.xsd entry. The nested include must fail through the
	// resolver (s:rootType stays unresolved) — it must not silently succeed by
	// reading part.xsd from disk.
	resolver := fileMapResolver{files: map[string]string{
		"mem:/stylesheets/other.xsd": ddSchemaXSD,
	}}
	doc, err := helium.NewParser().Parse(ctx, []byte(ddInlineMainXSL()))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().BaseURI(baseURI).URIResolver(resolver).Compile(ctx, doc)
	require.Error(t, err, "inline schema's nested xs:include must be denied when the resolver does not supply it")
}

// TestSourceSchemaLocationNestedIncludeThroughResolver verifies the runtime
// (xsi:schemaLocation) path also routes nested xs:include loads through the
// invocation's URIResolver instead of the xsd compiler's default os.Open.
func TestSourceSchemaLocationNestedIncludeThroughResolver(t *testing.T) {
	const sourceURI = "mem://docs/input.xml"
	const mainSchemaURI = "mem:/docs/main.xsd"
	const partSchemaURI = "mem:/docs/part.xsd"

	styleSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s"
    default-validation="strict">
  <xsl:import-schema namespace="http://example.com/s"/>
  <xsl:template match="/">
    <s:root>text</s:root>
  </xsl:template>
</xsl:stylesheet>`

	sourceSrc := `<?xml version="1.0"?>
<root xmlns="http://example.com/s"
      xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
      xsi:schemaLocation="http://example.com/s main.xsd">text</root>`

	ctx := t.Context()

	ssDoc, err := helium.NewParser().Parse(ctx, []byte(styleSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
	require.NoError(t, err)

	// Resolver supplies both the main schema and its nested include.
	resolver := runtimeFileMapResolver{files: map[string]string{
		mainSchemaURI: ddMainSchemaWithInclude,
		partSchemaURI: ddPartSchemaXSD,
	}}
	srcAllow, err := helium.NewParser().Parse(ctx, []byte(sourceSrc))
	require.NoError(t, err)
	srcAllow.SetURL(sourceURI)
	out, err := ss.Transform(srcAllow).URIResolver(resolver).Serialize(ctx)
	require.NoError(t, err, "runtime nested xs:include must resolve through the resolver")
	require.Contains(t, out, "root")
}

// TestSourceSchemaLocationDefaultDeny verifies that a source document's
// xsi:schemaLocation does not read schema files off the local filesystem at
// runtime unless an Invocation.URIResolver permits it.
func TestSourceSchemaLocationDefaultDeny(t *testing.T) {
	const sourceURI = "mem://docs/input.xml"
	const schemaURI = "mem:/docs/s.xsd"

	// default-validation="strict" so source schema-location load failures
	// surface as transform errors (otherwise they are swallowed). A
	// namespace-only xsl:import-schema (no schema-location) satisfies the
	// XTSE0020 requirement that strict validation imports a schema without
	// itself needing a resolver.
	styleSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s"
    default-validation="strict">
  <xsl:import-schema namespace="http://example.com/s"/>
  <xsl:template match="/">
    <s:root>text</s:root>
  </xsl:template>
</xsl:stylesheet>`

	sourceSrc := `<?xml version="1.0"?>
<root xmlns="http://example.com/s"
      xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
      xsi:schemaLocation="http://example.com/s s.xsd">text</root>`

	ctx := t.Context()

	ssDoc, err := helium.NewParser().Parse(ctx, []byte(styleSrc))
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().Compile(ctx, ssDoc)
	require.NoError(t, err)

	// Without a resolver: the runtime schema-location load is denied.
	srcDeny, err := helium.NewParser().Parse(ctx, []byte(sourceSrc))
	require.NoError(t, err)
	srcDeny.SetURL(sourceURI)
	_, err = ss.Transform(srcDeny).Serialize(ctx)
	require.Error(t, err, "source schema-location must be denied without a resolver")
	require.Contains(t, err.Error(), "no URIResolver configured")

	// With a resolver: the schema loads.
	resolver := runtimeFileMapResolver{files: map[string]string{
		schemaURI: ddSchemaXSD,
	}}
	srcAllow, err := helium.NewParser().Parse(ctx, []byte(sourceSrc))
	require.NoError(t, err)
	srcAllow.SetURL(sourceURI)
	out, err := ss.Transform(srcAllow).URIResolver(resolver).Serialize(ctx)
	require.NoError(t, err, "source schema-location must load with a resolver")
	require.Contains(t, out, "root")
}

// These tests pin the xslt3->xsd BOUNDARY translation (schemaResolverFS.Open):
// a NESTED xs:include/xs:import/xs:redefine reached while an xsl:import-schema
// compiles is loaded through the xslt3 resolver, and the resolver's error must be
// re-expressed in XSD's fs.FS vocabulary so xsd's readNestedSchema classifies it
// the SAME as the xslt3 side. A CONFIRMED resolution miss (a resolver reporting
// not-found, even one that does NOT itself satisfy fs.ErrNotExist) is demoted so
// an OPTIONAL nested include is skipped; a permission denial or a post-open read
// failure stays FATAL.

// boundaryImportSheet imports a schema by schema-location. The imported main.xsd
// carries a nested xs:include whose target the resolver controls.
const boundaryImportSheet = `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:s="http://example.com/s">
  <xsl:import-schema namespace="http://example.com/s" schema-location="main.xsd"/>
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`

// boundaryMainXSD is self-sufficient once its nested include is skipped: root is
// typed by a builtin, so a demoted (skipped) include leaves a valid schema.
const boundaryMainXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/s"
           xmlns:s="http://example.com/s"
           elementFormDefault="qualified">
  <xs:include schemaLocation="missing.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

// nestedFailResolver is a compile-time URIResolver (method Resolve) that serves
// main.xsd from a map but fails the nested include (basename "missing.xsd") in a
// chosen way: a Resolve error (resolution phase) or a reader that fails on Read
// (post-open). It models a resolver whose not-found error does NOT satisfy
// fs.ErrNotExist — the over-rejection the boundary translation fixes.
type nestedFailResolver struct {
	main       string // content served for main.xsd
	failBase   string // basename that fails
	resolveErr error  // if non-nil, Resolve returns this for failBase
	readErr    error  // else Resolve returns a reader that fails Read with this
}

func (r nestedFailResolver) Resolve(uri string) (io.ReadCloser, error) {
	if baseName(uri) == r.failBase {
		if r.resolveErr != nil {
			return nil, r.resolveErr
		}
		return badReadCloser{r.readErr}, nil
	}
	if baseName(uri) == "main.xsd" {
		return io.NopCloser(strings.NewReader(r.main)), nil
	}
	return nil, &resolverNotFoundError{uri: uri}
}

func compileBoundaryImport(t *testing.T, resolver nestedFailResolver) error {
	t.Helper()
	ctx := t.Context()
	doc, err := helium.NewParser().Parse(ctx, []byte(boundaryImportSheet))
	require.NoError(t, err)
	_, err = xslt3.NewCompiler().
		BaseURI("mem://stylesheets/main.xsl").
		URIResolver(resolver).
		Compile(ctx, doc)
	return err
}

// TestNestedIncludeResolverNotFoundDemoted is the repro: the resolver serves
// main.xsd but reports a bare (non-fs.ErrNotExist) "not found" for the nested
// include. The optional include must be skipped and the stylesheet must compile,
// not be over-rejected as "schema content invalid".
func TestNestedIncludeResolverNotFoundDemoted(t *testing.T) {
	err := compileBoundaryImport(t, nestedFailResolver{
		main:       boundaryMainXSD,
		failBase:   "missing.xsd",
		resolveErr: &resolverNotFoundError{uri: "missing.xsd"},
	})
	require.NoError(t, err, "a resolver not-found for an optional nested include must warn/skip, not fail compile")
}

// TestNestedIncludeResolverPermissionFatal verifies a PERMISSION denial on the
// nested include stays FATAL through the boundary (not demoted as a miss).
func TestNestedIncludeResolverPermissionFatal(t *testing.T) {
	err := compileBoundaryImport(t, nestedFailResolver{
		main:       boundaryMainXSD,
		failBase:   "missing.xsd",
		resolveErr: fs.ErrPermission,
	})
	require.Error(t, err, "a permission denial on a nested include must be fatal, not demoted")
}

// TestNestedIncludePostOpenReadFatal verifies a POST-OPEN read failure on the
// nested include (reader obtained then Read fails) stays FATAL through the
// boundary — it is not a resolution miss.
func TestNestedIncludePostOpenReadFatal(t *testing.T) {
	err := compileBoundaryImport(t, nestedFailResolver{
		main:     boundaryMainXSD,
		failBase: "missing.xsd",
		readErr:  io.ErrUnexpectedEOF,
	})
	require.Error(t, err, "a post-open read failure on a nested include must be fatal, not demoted")
}
