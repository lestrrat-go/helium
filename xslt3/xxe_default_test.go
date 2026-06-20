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

// A-001 opt-in: AllowExternalEntities(true) restores the legacy
// resolver-mediated external entity loading behavior. The document is served
// from an on-disk URI so that the relative SYSTEM entity resolves against the
// filesystem (the same legacy behavior W3C tests such as base-uri-051 rely on).
func TestXXE_RuntimeDocAllowedWithOptIn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("LEGACY-VALUE"), 0o600))
	docPath := filepath.Join(dir, "doc.xml")

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

// A-002 opt-in: Compiler.AllowExternalEntities(true) restores legacy
// stylesheet-module entity expansion.
func TestXXE_StylesheetIncludeAllowedWithOptIn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("INCLUDE-LEGACY"), 0o600))
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
