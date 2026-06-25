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

// TestLoaderClone exercises Loader.clone() when cfg is nil (the zero-value
// Loader path).
func TestLoaderClone(t *testing.T) {
	// zero value exercises clone() via MaxBytes, which calls clone internally.
	t.Run("zero value", func(t *testing.T) {
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
	})

	// error handler exercises clone() with nil cfg through ErrorHandler.
	t.Run("error handler", func(t *testing.T) {
		var l catalog.Loader
		ec := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		l2 := l.ErrorHandler(ec)
		require.NotNil(t, l2)
	})
}

// TestParseCatalog exercises catalog parsing branches.
func TestParseCatalog(t *testing.T) {
	// group nesting drives parseEntries recursion into <group> and the
	// per-element prefer attribute.
	t.Run("group nesting", func(t *testing.T) {
		cat, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <group prefer="public">
    <public publicId="-//Example//DTD//EN" uri="grouped.dtd"/>
  </group>
</catalog>`)
		require.Empty(t, errs)
		got := cat.Resolve(t.Context(), "-//Example//DTD//EN", "")
		require.True(t, strings.HasSuffix(got, "grouped.dtd"), "resolved %q", got)
	})

	// missing uri attr drives appendNameURLEntry -> catalogMissingAttr for the
	// missing-url branch (val2 == "").
	t.Run("missing uri attr", func(t *testing.T) {
		_, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="sid"/>
</catalog>`)
		require.NotEmpty(t, errs)
		require.Contains(t, errs[0].Error(), "uri")
	})

	// missing name attr drives catalogMissingAttr for the missing-name branch
	// (val1 == "").
	t.Run("missing name attr", func(t *testing.T) {
		_, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system uri="out.dtd"/>
</catalog>`)
		require.NotEmpty(t, errs)
		require.Contains(t, errs[0].Error(), "systemId")
	})

	// nextCatalog missing attr drives the nextCatalog branch where the catalog
	// attribute is empty.
	t.Run("nextCatalog missing attr", func(t *testing.T) {
		_, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <nextCatalog catalog=""/>
</catalog>`)
		require.NotEmpty(t, errs)
		require.Contains(t, errs[0].Error(), "catalog")
	})

	// xml:base drives the xml:base handling on both the root element and a
	// child element in parseEntries.
	t.Run("xml:base", func(t *testing.T) {
		cat, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog"
         xmlns:xml="http://www.w3.org/XML/1998/namespace"
         xml:base="http://example.com/base/">
  <uri name="u" uri="target.dtd" xml:base="sub/"/>
</catalog>`)
		require.Empty(t, errs)
		got := cat.ResolveURI(t.Context(), "u")
		require.Equal(t, "http://example.com/base/sub/target.dtd", got)
	})

	// non-catalog ns child skipped exercises the skip branch for child
	// elements outside the catalog namespace.
	t.Run("non-catalog ns child skipped", func(t *testing.T) {
		cat, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog"
         xmlns:other="http://example.com/other">
  <other:thing foo="bar"/>
  <system systemId="sid" uri="out.dtd"/>
</catalog>`)
		require.Empty(t, errs)
		got := cat.Resolve(t.Context(), "", "sid")
		require.True(t, strings.HasSuffix(got, "out.dtd"), "resolved %q", got)
	})

	// non-element child skipped exercises the AsNode skip branch for
	// text/comment children.
	t.Run("non-element child skipped", func(t *testing.T) {
		cat, errs := loadCatalogString(t, `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <!-- a comment -->
  text content
  <system systemId="sid" uri="out.dtd"/>
</catalog>`)
		require.Empty(t, errs)
		got := cat.Resolve(t.Context(), "", "sid")
		require.True(t, strings.HasSuffix(got, "out.dtd"), "resolved %q", got)
	})

	// wrong root namespace covers documentElement returning nil when the
	// document has no element child. A document that is only a comment + PI has
	// no root element at parse time; helium rejects it, so use the loadFromBytes
	// "no root" path via a parse that yields no element. Instead test the
	// empty-root namespace error.
	t.Run("wrong root namespace", func(t *testing.T) {
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
	})
}

// TestFileURI exercises catalogFilePath URI handling branches.
func TestFileURI(t *testing.T) {
	// unsupported scheme covers catalogFilePath rejecting a non-file scheme.
	t.Run("unsupported scheme", func(t *testing.T) {
		_, err := catalog.Load(t.Context(), "http://example.com/catalog.xml")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported URI scheme")
	})

	// non-local host covers catalogFilePath rejecting a remote host.
	t.Run("non-local host", func(t *testing.T) {
		_, err := catalog.Load(t.Context(), "file://remotehost/path/catalog.xml")
		require.Error(t, err)
		require.Contains(t, err.Error(), "non-local file URI host")
	})

	// opaque covers catalogFilePath rejecting an opaque file: URI.
	t.Run("opaque", func(t *testing.T) {
		_, err := catalog.Load(t.Context(), "file:next.xml")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no local path")
	})

	// localhost covers the EqualFold "localhost" accept path plus fileURIPath
	// conversion for a real file.
	t.Run("localhost", func(t *testing.T) {
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
	})
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
