package catalog_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/catalog"
	"github.com/stretchr/testify/require"
)

// loadCatalogString writes xml to a temp file and loads it, returning the
// catalog and the slice of errors the loader's ErrorHandler collected.
func loadCatalogString(t *testing.T, xml string) (*catalog.Catalog, []error) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "catalog.xml")
	require.NoError(t, os.WriteFile(p, []byte(xml), 0o600))

	ec := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	cat, err := catalog.NewLoader().ErrorHandler(ec).Load(t.Context(), p)
	require.NoError(t, err)
	require.NoError(t, ec.Close())
	return cat, ec.Errors()
}

// TestZeroValueLoaderClone exercises Loader.clone() when cfg is nil (the
// zero-value Loader path) via MaxBytes, which calls clone internally.
func TestZeroValueLoaderClone(t *testing.T) {
	var l catalog.Loader // zero value: cfg is nil
	l2 := l.MaxBytes(4096)

	dir := t.TempDir()
	p := filepath.Join(dir, "catalog.xml")
	xml := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="sid" uri="out.dtd"/>
</catalog>`
	require.NoError(t, os.WriteFile(p, []byte(xml), 0o600))

	cat, err := l2.Load(t.Context(), p)
	require.NoError(t, err)
	require.NotNil(t, cat)
}

// TestZeroValueLoaderErrorHandlerClone exercises clone() with nil cfg through
// ErrorHandler.
func TestZeroValueLoaderErrorHandlerClone(t *testing.T) {
	var l catalog.Loader
	ec := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	l2 := l.ErrorHandler(ec)
	require.NotNil(t, l2)
}

// TestGroupNesting drives parseEntries recursion into <group> and the
// per-element prefer attribute.
func TestGroupNesting(t *testing.T) {
	cat, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <group prefer="public">
    <public publicId="-//Example//DTD//EN" uri="grouped.dtd"/>
  </group>
</catalog>`)
	require.Empty(t, errs)
	got := cat.Resolve(t.Context(), "-//Example//DTD//EN", "")
	require.True(t, strings.HasSuffix(got, "grouped.dtd"), "resolved %q", got)
}

// TestMissingURIAttr drives appendNameURLEntry -> catalogMissingAttr for the
// missing-url branch (val2 == "").
func TestMissingURIAttr(t *testing.T) {
	_, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="sid"/>
</catalog>`)
	require.NotEmpty(t, errs)
	require.Contains(t, errs[0].Error(), "uri")
}

// TestMissingNameAttr drives catalogMissingAttr for the missing-name branch
// (val1 == "").
func TestMissingNameAttr(t *testing.T) {
	_, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system uri="out.dtd"/>
</catalog>`)
	require.NotEmpty(t, errs)
	require.Contains(t, errs[0].Error(), "systemId")
}

// TestNextCatalogMissingAttr drives the nextCatalog branch where the catalog
// attribute is empty.
func TestNextCatalogMissingAttr(t *testing.T) {
	_, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <nextCatalog catalog=""/>
</catalog>`)
	require.NotEmpty(t, errs)
	require.Contains(t, errs[0].Error(), "catalog")
}

// TestXMLBaseAttribute drives the xml:base handling on both the root element
// and a child element in parseEntries.
func TestXMLBaseAttribute(t *testing.T) {
	cat, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog"
         xmlns:xml="http://www.w3.org/XML/1998/namespace"
         xml:base="http://example.com/base/">
  <uri name="u" uri="target.dtd" xml:base="sub/"/>
</catalog>`)
	require.Empty(t, errs)
	got := cat.ResolveURI(t.Context(), "u")
	require.Equal(t, "http://example.com/base/sub/target.dtd", got)
}

// TestNonCatalogNamespaceChildSkipped exercises the skip branch for child
// elements outside the catalog namespace.
func TestNonCatalogNamespaceChildSkipped(t *testing.T) {
	cat, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog"
         xmlns:other="http://example.com/other">
  <other:thing foo="bar"/>
  <system systemId="sid" uri="out.dtd"/>
</catalog>`)
	require.Empty(t, errs)
	got := cat.Resolve(t.Context(), "", "sid")
	require.True(t, strings.HasSuffix(got, "out.dtd"), "resolved %q", got)
}

// TestNonElementChildSkipped exercises the AsNode skip branch for text/comment
// children.
func TestNonElementChildSkipped(t *testing.T) {
	cat, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <!-- a comment -->
  text content
  <system systemId="sid" uri="out.dtd"/>
</catalog>`)
	require.Empty(t, errs)
	got := cat.Resolve(t.Context(), "", "sid")
	require.True(t, strings.HasSuffix(got, "out.dtd"), "resolved %q", got)
}

// TestNoRootElement covers documentElement returning nil when the document has
// no element child. A document that is only a comment + PI has no root element
// at parse time; helium rejects it, so use the loadFromBytes "no root" path via
// a parse that yields no element. Instead test the empty-root namespace error.
func TestWrongRootNamespace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "catalog.xml")
	xml := `<?xml version="1.0"?>
<catalog xmlns="http://example.com/not-catalog">
  <system systemId="sid" uri="out.dtd"/>
</catalog>`
	require.NoError(t, os.WriteFile(p, []byte(xml), 0o600))
	_, err := catalog.Load(t.Context(), p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "namespace")
}

// TestUnsupportedURIScheme covers catalogFilePath rejecting a non-file scheme.
func TestUnsupportedURIScheme(t *testing.T) {
	_, err := catalog.Load(t.Context(), "http://example.com/catalog.xml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported URI scheme")
}

// TestNonLocalFileURIHost covers catalogFilePath rejecting a remote host.
func TestNonLocalFileURIHost(t *testing.T) {
	_, err := catalog.Load(t.Context(), "file://remotehost/path/catalog.xml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-local file URI host")
}

// TestOpaqueFileURI covers catalogFilePath rejecting an opaque file: URI.
func TestOpaqueFileURI(t *testing.T) {
	_, err := catalog.Load(t.Context(), "file:next.xml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no local path")
}

// TestLocalhostFileURI covers the EqualFold "localhost" accept path plus
// fileURIPath conversion for a real file.
func TestLocalhostFileURI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "catalog.xml")
	xml := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="sid" uri="out.dtd"/>
</catalog>`
	require.NoError(t, os.WriteFile(p, []byte(xml), 0o600))

	// Build "file://localhost/<path>". On Windows the slashed path is
	// "C:/..." (no leading slash), so prepend one to avoid producing
	// "file://localhostC:/..." where "localhostC:" is read as the host.
	slashed := filepath.ToSlash(p)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	uri := "file://localhost" + slashed
	cat, err := catalog.Load(t.Context(), uri)
	require.NoError(t, err)
	got := cat.Resolve(t.Context(), "", "sid")
	require.True(t, strings.HasPrefix(got, "file:"), "resolved %q", got)
}

// TestNilContextLoad covers the ctx == nil branch in Loader.Load.
func TestNilContextLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "catalog.xml")
	xml := `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="sid" uri="out.dtd"/>
</catalog>`
	require.NoError(t, os.WriteFile(p, []byte(xml), 0o600))

	var nilCtx context.Context
	cat, err := catalog.NewLoader().Load(nilCtx, p)
	require.NoError(t, err)
	require.NotNil(t, cat)
}
