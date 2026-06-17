package xslt3_test

import (
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

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
// through the resolver rather than read off the local filesystem.
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
// xs:include is denied through the resolver rather than read off disk.
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
