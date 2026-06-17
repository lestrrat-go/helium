package xslt3_test

import (
	"io"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

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
// surface this as XTSE0220 rather than silently succeeding with an invalid
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
