package iofs_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/stretchr/testify/require"
)

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
		"file URI": "file://" + filepath.ToSlash(abs),
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

// A non-file URI scheme is refused so the FS never reaches the network.
func TestConfinedDirRefusesNonFileScheme(t *testing.T) {
	t.Parallel()

	c := iofs.NewConfinedDir(t.TempDir())
	for _, uri := range []string{
		"http://example.com/x.dtd",
		"https://host/x.dtd",
		"ftp://host/x.dtd",
	} {
		_, err := c.Open(uri)
		require.ErrorIs(t, err, fs.ErrPermission, "%s must be refused", uri)
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
