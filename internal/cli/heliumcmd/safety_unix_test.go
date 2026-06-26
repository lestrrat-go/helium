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

func TestXSLTConfinedFSBlocksSymlinkEscape(t *testing.T) {
	// A symlink living INSIDE the stylesheet directory but pointing OUTSIDE it
	// must not let --noent exfiltrate the linked-to file. A purely lexical
	// containment check passes the symlink (its own path is inside the root) and
	// os.Open would then follow it; os.Root confinement rejects the escaping
	// link, so the secret never reaches the output.
	const secret = "TOPSECRETSYMLINK"

	outsideDir := t.TempDir()
	secretFile := writeFile(t, outsideDir, "secret.txt", secret)

	dir := t.TempDir()
	link := filepath.Join(dir, "leak")
	require.NoError(t, os.Symlink(secretFile, link))

	ssFile := writeFile(t, dir, "main.xsl", `<?xml version="1.0"?>
<!DOCTYPE xsl:stylesheet [ <!ENTITY x SYSTEM "leak"> ]>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root"><out>&x;</out></xsl:template>
</xsl:stylesheet>`)
	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root/>`)

	out, _, _ := executeArgs(t, strings.NewReader(""),
		"xslt", "--noent", ssFile, xmlFile)
	require.NotContains(t, out, secret,
		"a symlink inside the stylesheet directory must not escape the confined FS")
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

func TestLintOutputThroughDanglingSymlink(t *testing.T) {
	// --output onto a symlink whose target does NOT exist must write through to
	// the resolved target path (creating it), exactly as os.Create would.
	// filepath.EvalSymlinks rejects a dangling link, so the resolver must follow
	// it with os.Readlink instead.
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>x</root>`)

	missing := filepath.Join(dir, "missing.xml")
	link := filepath.Join(dir, "link.xml")
	require.NoError(t, os.Symlink(missing, link))

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--output", link, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)

	// The previously-missing target now exists and holds the output.
	got, err := os.ReadFile(missing)
	require.NoError(t, err)
	require.Contains(t, string(got), "<root>x</root>")

	// link.xml is still a symlink pointing at the resolved target.
	fi, err := os.Lstat(link)
	require.NoError(t, err)
	require.NotZero(t, fi.Mode()&os.ModeSymlink, "link.xml should remain a symlink")
	dest, err := os.Readlink(link)
	require.NoError(t, err)
	require.Equal(t, missing, dest)
}

func TestLintOutputExistingFileStickyBitPreserved(t *testing.T) {
	// A pre-existing output file with the sticky bit set must keep it: mode
	// preservation must include sticky/setuid/setgid, not just the permission
	// bits.
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>x</root>`)
	outFile := filepath.Join(dir, "out.xml")
	require.NoError(t, os.WriteFile(outFile, []byte("old"), 0o640))
	require.NoError(t, os.Chmod(outFile, 0o640|os.ModeSticky))

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--output", outFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)

	fi, err := os.Stat(outFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), fi.Mode().Perm())
	require.NotZero(t, fi.Mode()&os.ModeSticky, "sticky bit should be preserved")
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

func TestLintOutputReadOnlyExistingFileRejected(t *testing.T) {
	// A pre-existing read-only (0444) --output target must NOT be silently
	// overwritten via os.Rename. The command must fail with a non-zero exit and
	// the original file's content must be left intact. (root bypasses the file
	// mode check, so skip there.)
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permission checks")
	}

	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>x</root>`)
	outFile := filepath.Join(dir, "out.xml")
	require.NoError(t, os.WriteFile(outFile, []byte("KEEP"), 0o444))

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--output", outFile, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)
	require.Contains(t, errOut, "cannot write to")

	// The read-only file must be untouched.
	got, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Equal(t, "KEEP", string(got))
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
