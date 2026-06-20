package catalog_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/catalog"
	"github.com/stretchr/testify/require"
)

const minimalCatalogXML = `<?xml version="1.0"?>
<catalog xmlns="urn:oasis:names:tc:entity:xmlns:xml:catalog">
  <system systemId="http://example.com/test.dtd" uri="test.dtd"/>
</catalog>`

// writeCatalog writes content to a catalog file in a fresh temp dir and returns
// its path.
func writeCatalog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.xml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadMaxBytes(t *testing.T) {
	t.Parallel()

	t.Run("over cap fails with ErrCatalogTooLarge", func(t *testing.T) {
		t.Parallel()
		// Pad the catalog past a small cap with trailing whitespace, which keeps
		// the XML well-formed so the failure is the size cap, not a parse error.
		padded := minimalCatalogXML + strings.Repeat(" ", 4096)
		path := writeCatalog(t, padded)

		_, err := catalog.NewLoader().MaxBytes(64).Load(t.Context(), path)
		require.Error(t, err)
		require.ErrorIs(t, err, catalog.ErrCatalogTooLarge)
	})

	t.Run("under cap still loads", func(t *testing.T) {
		t.Parallel()
		path := writeCatalog(t, minimalCatalogXML)

		cat, err := catalog.NewLoader().MaxBytes(64<<10).Load(t.Context(), path)
		require.NoError(t, err)
		got := cat.Resolve(t.Context(), "", "http://example.com/test.dtd")
		require.NotEqual(t, "", got, "expected the system entry to resolve")
	})

	t.Run("default cap loads a normal catalog", func(t *testing.T) {
		t.Parallel()
		path := writeCatalog(t, minimalCatalogXML)

		cat, err := catalog.Load(t.Context(), path)
		require.NoError(t, err)
		got := cat.Resolve(t.Context(), "", "http://example.com/test.dtd")
		require.NotEqual(t, "", got)
	})

	t.Run("at cap loads", func(t *testing.T) {
		t.Parallel()
		path := writeCatalog(t, minimalCatalogXML)
		info, err := os.Stat(path)
		require.NoError(t, err)

		cat, err := catalog.NewLoader().MaxBytes(int(info.Size())).Load(t.Context(), path)
		require.NoError(t, err)
		got := cat.Resolve(t.Context(), "", "http://example.com/test.dtd")
		require.NotEqual(t, "", got)
	})

	t.Run("MaxInt cap does not overflow into a zero read", func(t *testing.T) {
		t.Parallel()
		// MaxBytes(math.MaxInt) once made the over-cap probe (limit+1) overflow
		// to a negative value on 64-bit platforms, so io.LimitReader read zero
		// bytes and a valid catalog failed to parse. A huge cap must load a
		// normal catalog successfully.
		path := writeCatalog(t, minimalCatalogXML)

		cat, err := catalog.NewLoader().MaxBytes(math.MaxInt).Load(t.Context(), path)
		require.NoError(t, err)
		got := cat.Resolve(t.Context(), "", "http://example.com/test.dtd")
		require.NotEqual(t, "", got, "expected the system entry to resolve")
	})
}
