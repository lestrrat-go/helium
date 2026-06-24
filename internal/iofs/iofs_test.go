package iofs_test

import (
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/stretchr/testify/require"
)

func TestFileURIToPath(t *testing.T) {
	t.Parallel()

	// FileURIToPath returns a NATIVE path (via filepath.FromSlash), so the
	// expected value is wrapped in filepath.FromSlash: "/tmp/inc.xml" on POSIX,
	// "\\tmp\\inc.xml" on Windows.
	t.Run("absolute file URI", func(t *testing.T) {
		t.Parallel()
		p, err := iofs.FileURIToPath("file:///tmp/inc.xml")
		require.NoError(t, err)
		require.Equal(t, filepath.FromSlash("/tmp/inc.xml"), p)
	})

	t.Run("localhost host", func(t *testing.T) {
		t.Parallel()
		p, err := iofs.FileURIToPath("file://localhost/tmp/inc.xml")
		require.NoError(t, err)
		require.Equal(t, filepath.FromSlash("/tmp/inc.xml"), p)
	})

	t.Run("percent-decoded path", func(t *testing.T) {
		t.Parallel()
		p, err := iofs.FileURIToPath("file:///tmp/a%20b.xml")
		require.NoError(t, err)
		require.Equal(t, filepath.FromSlash("/tmp/a b.xml"), p)
	})

	t.Run("non-local host rejected", func(t *testing.T) {
		t.Parallel()
		_, err := iofs.FileURIToPath("file://remote/tmp/inc.xml")
		require.Error(t, err)
	})

	t.Run("opaque file URI rejected", func(t *testing.T) {
		t.Parallel()
		_, err := iofs.FileURIToPath("file:next.xml")
		require.Error(t, err)
	})

	t.Run("non-file scheme rejected", func(t *testing.T) {
		t.Parallel()
		_, err := iofs.FileURIToPath("http://example.com/inc.xml")
		require.Error(t, err)
	})
}
