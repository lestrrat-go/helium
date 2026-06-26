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

	// "file:////server/share/x" parses to an empty host with path
	// "//server/share/x"; on Windows that would become the UNC path
	// \\server\share\x, reaching a remote SMB host despite the local-only
	// policy. It must be rejected on every platform.
	t.Run("UNC file URI rejected", func(t *testing.T) {
		t.Parallel()
		_, err := iofs.FileURIToPath("file:////server/share/inc.xml")
		require.Error(t, err)
	})

	// url.Parse percent-decodes u.Path, so a "%5C"/"%5c" encoded backslash after
	// the leading slash decodes to "/\server/share"; on Windows filepath.FromSlash
	// turns that into the UNC path \\server\share. The "//"-only check missed
	// these, so they must be rejected too.
	for _, ref := range []string{
		"file:///%5Cserver/share/inc.xml",
		"file:///%5cserver/share/inc.xml",
		"file:///%5C%5Cserver/share/inc.xml",
	} {
		t.Run("encoded-backslash UNC rejected: "+ref, func(t *testing.T) {
			t.Parallel()
			_, err := iofs.FileURIToPath(ref)
			require.Error(t, err)
		})
	}
}
