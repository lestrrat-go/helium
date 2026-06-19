//go:build !windows && !plan9

package heliumcmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

func TestLintOutputSelfTruncateRejectedViaSymlink(t *testing.T) {
	dir := t.TempDir()
	content := `<?xml version="1.0"?><root>data</root>`
	xmlFile := writeFile(t, dir, "doc.xml", content)

	// A symlink pointing at the input is a genuine same-file alias whose
	// lexical path differs: os.SameFile (inode/device) must catch it so the
	// collision check fires rather than some incidental open failure.
	link := filepath.Join(dir, "alias.xml")
	require.NoError(t, os.Symlink(xmlFile, link))

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--output", link, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)
	require.Contains(t, errOut, "overwrite input")

	got, err := os.ReadFile(xmlFile)
	require.NoError(t, err)
	require.Equal(t, content, string(got))
}

func TestLintOutputThroughSymlink(t *testing.T) {
	// --output onto a symlink must write THROUGH to the real target: the linked
	// file's content is updated and the link stays a link. os.Rename would
	// otherwise replace the symlink with a regular file and leave the real
	// target untouched.
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>x</root>`)

	realTarget := filepath.Join(dir, "real.xml")
	require.NoError(t, os.WriteFile(realTarget, []byte("old"), 0o640))
	link := filepath.Join(dir, "link.xml")
	require.NoError(t, os.Symlink(realTarget, link))

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--output", link, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)

	// The real target received the output.
	got, err := os.ReadFile(realTarget)
	require.NoError(t, err)
	require.Contains(t, string(got), "<root>x</root>")

	// link.xml is still a symlink pointing at the real target.
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	require.NotZero(t, fi.Mode()&os.ModeSymlink, "link.xml should remain a symlink")
	dest, err := os.Readlink(link)
	require.NoError(t, err)
	require.Equal(t, realTarget, dest)

	// The resolved target kept its original mode (preserved, not replaced).
	rfi, err := os.Stat(realTarget)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), rfi.Mode().Perm())
}

func TestLintOutputNewFileModeRespectsUmask(t *testing.T) {
	// os.CreateTemp makes the temp 0600; a NEW --output destination must end up
	// with the usual os.Create mode (0666 masked by umask), not 0600.
	old := syscall.Umask(0o022)
	defer syscall.Umask(old)

	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>x</root>`)
	outFile := filepath.Join(dir, "out.xml")

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--output", outFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)

	fi, err := os.Stat(outFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), fi.Mode().Perm())
}

func TestLintOutputExistingFileModePreserved(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>x</root>`)
	outFile := filepath.Join(dir, "out.xml")
	require.NoError(t, os.WriteFile(outFile, []byte("old"), 0o640))

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--output", outFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)

	fi, err := os.Stat(outFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), fi.Mode().Perm())
}
