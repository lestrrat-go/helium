package xslt3_test

import (
	"io"
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
