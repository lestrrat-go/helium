package iofs_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/stretchr/testify/require"
)

// A non-file-scheme absolute URI is not a local filesystem path: PermissiveRoot
// must classify a failed local os.Open attempt as a resolution MISS
// (fs.ErrNotExist) rather than returning a platform-dependent errno (ENOENT on
// Linux, EINVAL on macOS/Windows) that a demotion classifier cannot treat
// consistently. A local path (no scheme, or a "file:" URI) still reaches os.Open.
func TestPermissiveRootNonFileURI(t *testing.T) {
	t.Parallel()

	for _, uri := range []string{
		"http://example.com/missing.xsd",
		"https://host:8080/x.xsd",
		"urn:example:schema",
		"ftp://host/x.xsd",
	} {
		_, err := iofs.PermissiveRoot{}.Open(uri)
		require.Errorf(t, err, "non-file URI %q must not open", uri)
		require.Truef(t, errors.Is(err, fs.ErrNotExist),
			"non-file URI %q must classify as fs.ErrNotExist (resolution miss), got %v", uri, err)
	}

	// A real local file still opens through os.Open (no scheme interception).
	dir := t.TempDir()
	p := filepath.Join(dir, "real.xsd")
	require.NoError(t, os.WriteFile(p, []byte("<x/>"), 0o600))
	f, err := iofs.PermissiveRoot{}.Open(p)
	require.NoError(t, err, "a genuine local path must open via os.Open")
	b, _ := io.ReadAll(f)
	_ = f.Close()
	require.Equal(t, "<x/>", string(b))

	// A missing local path still yields os.Open's own ErrNotExist (not intercepted
	// as a URI), so a local malformed/missing path stays fatal/normal.
	_, err = iofs.PermissiveRoot{}.Open(filepath.Join(dir, "missing.xsd"))
	require.True(t, errors.Is(err, fs.ErrNotExist))
}

func TestPermissiveRootOpensURILikeLocalFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows path components cannot contain ':'; URI-shaped local filename coverage is POSIX-only")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	const name = "urn:cache-key"
	require.NoError(t, os.WriteFile(name, []byte("cached"), 0o600))

	f, err := iofs.PermissiveRoot{}.Open(name)
	require.NoError(t, err, "a local filename that looks like a URI must still open via os.Open")
	defer f.Close()

	b, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, "cached", string(b))
}

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
