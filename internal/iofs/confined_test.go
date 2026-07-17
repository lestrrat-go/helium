package iofs_test

import (
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/stretchr/testify/require"
)

// fileURIFromPath builds a "file:" URI from a local filesystem path that is
// correct on every OS. On Windows filepath.ToSlash yields "C:/..." which,
// serialized WITHOUT a leading slash, becomes "file://C:/..." — "C:" is parsed
// as the URI host and FileURIToPath rejects it as non-local. Adding one leading
// slash makes it "file:///C:/..." (and keeps "file:///tmp/x" on POSIX). Every
// test that turns a t.TempDir() path into a file URI must use this.
func fileURIFromPath(path string) string {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

// ConfinedDir serves an in-root file by BOTH its absolute path and a relative
// name, and by an in-root "file:" URI. The absolute-path form is the value-add
// over a bare os.DirFS (which rejects a non-fs.ValidPath absolute name).
func TestConfinedDirServesInRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "sub.dtd"), []byte("payload"), 0o600))
	c := iofs.NewConfinedDir(root)

	abs := filepath.Join(root, "sub.dtd")
	for name, form := range map[string]string{
		"relative": "sub.dtd",
		"absolute": abs,
		"file URI": fileURIFromPath(abs),
	} {
		t.Run(name, func(t *testing.T) {
			f, err := c.Open(form)
			require.NoError(t, err, "%s name must open an in-root file", name)
			b, err := io.ReadAll(f)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			require.Equal(t, "payload", string(b))
		})
	}
}

// A name that resolves outside root — an absolute out-of-root path or a "../"
// escape — is refused with fs.ErrPermission, even though the target file exists
// and is readable. The refusal is the confinement guard, not a missing file.
func TestConfinedDirRefusesEscape(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret")
	require.NoError(t, os.WriteFile(secret, []byte("secret"), 0o600))

	root := t.TempDir()
	c := iofs.NewConfinedDir(root)

	for name, form := range map[string]string{
		"absolute out-of-root": secret,
		"dot-dot escape":       filepath.Join("..", filepath.Base(outside), "secret"),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := c.Open(form)
			require.ErrorIs(t, err, fs.ErrPermission, "%s must be refused", name)
		})
	}
}

// A non-file URI scheme is refused so the FS never reaches the network. A
// ONE-letter scheme ("x:dtd") is refused too: it is a valid RFC 3986 scheme, and
// letting it through would open an in-root file named "x:dtd" (or, with a
// matching name, reach a resource the caller meant a scheme to gate). This is
// the one-letter-scheme bypass that a two-char-minimum scheme test misses.
func TestConfinedDirRefusesNonFileScheme(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// A file literally named "x:dtd" exists in root; refusal must be the scheme
	// guard firing, not the file being absent.
	require.NoError(t, os.WriteFile(filepath.Join(root, "x:dtd"), []byte("payload"), 0o600))
	c := iofs.NewConfinedDir(root)
	for _, uri := range []string{
		"http://example.com/x.dtd",
		"https://host/x.dtd",
		"ftp://host/x.dtd",
		"urn:isbn:0",
		"x:dtd", // one-letter scheme
		"a:b/c", // one-letter scheme with a path
	} {
		_, err := c.Open(uri)
		require.ErrorIs(t, err, fs.ErrPermission, "%s must be refused", uri)
	}
}

// An opaque or empty-path "file:" URI ("file:inside", "file:") is served as the
// root directory rather than rejected, matching the private confinedDirFS this
// shared type replaces (localFilePath converted such a URI to an empty path, so
// Open joined it onto root and returned the root dir — an "is a directory" read
// error at the CLI). FileURIToPath rejects the opaque form, so ConfinedDir must
// NOT route "file:" inputs through it. Do not change this without checking the
// CLI's observable behavior for a "file:inside" external entity.
func TestConfinedDirOpaqueFileURIServesRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	c := iofs.NewConfinedDir(root)
	for _, uri := range []string{"file:inside", "file:"} {
		f, err := c.Open(uri)
		require.NoError(t, err, "%s must open the root directory (empty local path)", uri)
		// Reading the root directory as a file fails, exactly as origin/main's
		// "read <root>/.: is a directory" — the parity we preserve.
		_, rerr := io.ReadAll(f)
		require.Error(t, rerr, "%s resolves to the root directory, whose read fails", uri)
		require.NoError(t, f.Close())
	}
}

// os.DirFS follows an in-root symlink that points outside its root; ConfinedDir
// uses os.Root and refuses it. This is the symlink-sandbox guarantee that a bare
// os.DirFS does not provide.
func TestConfinedDirRefusesEscapingSymlink(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret")
	require.NoError(t, os.WriteFile(secret, []byte("secret"), 0o600))

	root := t.TempDir()
	link := filepath.Join(root, "leak")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	// os.DirFS follows the escaping symlink and leaks the out-of-root file.
	data, err := fs.ReadFile(os.DirFS(root), "leak")
	require.NoError(t, err, "os.DirFS follows an in-root symlink out of the root")
	require.Equal(t, "secret", string(data))

	// ConfinedDir refuses the same open.
	_, err = iofs.NewConfinedDir(root).Open("leak")
	require.Error(t, err, "ConfinedDir must refuse an open that escapes root via a symlink")
	require.False(t, errors.Is(err, fs.ErrNotExist),
		"the refusal is a confinement error, not a missing file")
}
