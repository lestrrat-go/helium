package xslt3_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// xxeResolver serves a fixed set of URIs from an in-memory map. It mimics a
// URIResolver / Compiler.URIResolver that a caller installs to permit external
// document / stylesheet loading.
type xxeResolver struct {
	files map[string]string
}

func (r *xxeResolver) Resolve(uri string) (io.ReadCloser, error) {
	body, ok := r.files[uri]
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

// A-001: runtime fn:doc / document() of a resolver-served doc whose XML
// defines an external SYSTEM entity referencing a local file must NOT expand
// that entity by default (XXE blocked).
func TestXXE_RuntimeDocBlockedByDefault(t *testing.T) {
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
}

// A-001 opt-in: AllowExternalEntities(true) restores external entity loading,
// but the load is now ROUTED THROUGH the configured URIResolver (not the raw
// filesystem). The external SYSTEM entity resolves against the document's base
// URI and the resulting entity URI is served by the same resolver. This proves
// opted-in entities go through the resolver-mediated, resource-limited channel
// rather than the parser's default os.Open.
func TestXXE_RuntimeDocAllowedWithOptIn(t *testing.T) {
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
}

// A-002: xsl:include of a resolver-returned stylesheet module that defines an
// external SYSTEM entity referencing a local file must NOT expand that entity
// by default (compile-time XXE blocked).
func TestXXE_StylesheetIncludeBlockedByDefault(t *testing.T) {
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
}

// A-002 opt-in: Compiler.AllowExternalEntities(true) restores stylesheet-module
// entity expansion, with the external entity load routed through the configured
// URIResolver (not the raw filesystem). The resolver serves both the included
// module and the resolved entity URI.
func TestXXE_StylesheetIncludeAllowedWithOptIn(t *testing.T) {
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

// A-003 (regression): under the secure default (XXE blocked), fn:doc() must
// still apply internal-subset DTD default attributes. The secure parser path
// previously dropped the extraOpts (DefaultDTDAttributes), so the @kind default
// vanished and the output was <out/>. Internal-subset DTD processing must keep
// working while EXTERNAL DTD/entity/network stay blocked.
func TestXXE_RuntimeDocInternalDTDDefaultAttr(t *testing.T) {
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
}

// A-004: opted-in external entities must be loaded THROUGH the configured
// URIResolver, not the parser's raw filesystem. The secret file exists on disk
// (where a raw-FS parse would read it) but the resolver does NOT serve its URI;
// the entity must therefore fail to resolve and never leak the on-disk content.
func TestXXE_RuntimeDocOptInUsesResolverNotRawFS(t *testing.T) {
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
}

// A-005: imported XSD schemas are ALWAYS parsed XXE-blocked. Even with
// Compiler.AllowExternalEntities(true), an external SYSTEM entity in an imported
// schema must not be expanded — the entity opt-in does not extend to schemas.
func TestXXE_ImportSchemaAlwaysBlocked(t *testing.T) {
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
}
