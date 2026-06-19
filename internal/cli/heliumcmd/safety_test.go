package heliumcmd_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

func executeArgs(t *testing.T, stdin io.Reader, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), stdin, &outBuf, &errBuf)
	ctx = heliumcmd.WithStdinTTY(ctx, true)
	exit := heliumcmd.Execute(ctx, args)
	return outBuf.String(), errBuf.String(), exit
}

func TestLintOutputSelfTruncateRejected(t *testing.T) {
	dir := t.TempDir()
	content := `<?xml version="1.0"?><root>data</root>`
	xmlFile := writeFile(t, dir, "doc.xml", content)

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--output", xmlFile, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)
	require.Contains(t, errOut, "overwrite input")

	// The input file must remain intact.
	got, err := os.ReadFile(xmlFile)
	require.NoError(t, err)
	require.Equal(t, content, string(got))
}

func TestLintOutputSelfTruncateRejectedViaRelPath(t *testing.T) {
	dir := t.TempDir()
	content := `<?xml version="1.0"?><root>data</root>`
	xmlFile := writeFile(t, dir, "doc.xml", content)

	// Same file referenced via a "./"-prefixed path: os.SameFile must catch it.
	rel := "./" + xmlFile

	_, _, code := executeArgs(t, strings.NewReader(""), "lint", "--output", rel, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)

	got, err := os.ReadFile(xmlFile)
	require.NoError(t, err)
	require.Equal(t, content, string(got))
}

func TestLintOutputNoOutRejected(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)
	outFile := filepath.Join(dir, "out.xml")
	require.NoError(t, os.WriteFile(outFile, []byte("KEEP"), 0o600))

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--noout", "--output", outFile, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)
	require.Contains(t, errOut, "noout")

	// out.xml must not have been truncated.
	got, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Equal(t, "KEEP", string(got))
}

func TestLintOutputWritesToFile(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>x</root>`)
	outFile := filepath.Join(dir, "out.xml")

	_, _, code := executeArgs(t, strings.NewReader(""), "lint", "--output", outFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code)

	got, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Contains(t, string(got), "<root>x</root>")
}

func TestLintOutputOverLaterReadDTDSucceeds(t *testing.T) {
	// --output points at a DTD that is resolved via --path and read DURING
	// validation, i.e. AFTER the output target is opened. The pre-flight
	// collision check cannot catch this (the DTD path is not an input arg), so
	// the temp-file-then-rename scheme must keep the DTD intact until its read
	// completes. The run must succeed rather than truncate the DTD first.
	dir := t.TempDir()
	dtdDir := filepath.Join(dir, "dtd")
	require.NoError(t, os.Mkdir(dtdDir, 0o755))
	dtdFile := filepath.Join(dtdDir, "note.dtd")
	require.NoError(t, os.WriteFile(dtdFile, []byte("<!ELEMENT note (to)>\n<!ELEMENT to (#PCDATA)>\n"), 0o600))

	xmlFile := writeFile(t, dir, "doc.xml",
		"<?xml version=\"1.0\"?>\n<!DOCTYPE note SYSTEM \"note.dtd\">\n<note><to>x</to></note>")

	out, errOut, code := executeArgs(t, strings.NewReader(""),
		"lint", "--loaddtd", "--valid", "--path", dtdDir, "--output", dtdFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)

	// The output was published to the DTD path only after the DTD read
	// completed during validation, so validation succeeded.
	got, err := os.ReadFile(dtdFile)
	require.NoError(t, err)
	require.Contains(t, string(got), "<note>")
	require.Empty(t, out)
}

func TestLintOutputErrorLeavesTargetIntact(t *testing.T) {
	// A processing error (malformed XML) must leave the pre-existing output
	// target untouched: the temp file is discarded and never renamed onto it.
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><unclosed></root>`)
	outFile := filepath.Join(dir, "out.xml")
	require.NoError(t, os.WriteFile(outFile, []byte("KEEP"), 0o600))

	_, _, code := executeArgs(t, strings.NewReader(""), "lint", "--output", outFile, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)

	got, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Equal(t, "KEEP", string(got))
}

func TestLintMaxInputBytesExceeded(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>aaaaaaaaaa</root>`)

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--max-input-bytes", "10", xmlFile)
	require.Equal(t, heliumcmd.ExitReadFile, code)
	require.Contains(t, errOut, "exceeds maximum size")
}

func TestLintMaxInputBytesStdinExceeded(t *testing.T) {
	big := "<?xml version=\"1.0\"?><root>" + strings.Repeat("a", 1000) + "</root>"
	var outBuf, errBuf bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(big), &outBuf, &errBuf)
	ctx = heliumcmd.WithStdinTTY(ctx, false)
	code := heliumcmd.Execute(ctx, []string{"lint", "--max-input-bytes", "50"})
	require.Equal(t, heliumcmd.ExitReadFile, code)
	require.Contains(t, errBuf.String(), "exceeds maximum size")
}

func TestLintMaxInputBytesUnlimited(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>x</root>`)

	_, _, code := executeArgs(t, strings.NewReader(""), "lint", "--max-input-bytes", "0", xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code)
}

func TestLintQuietSuppressesTiming(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	_, errOut, code := executeArgs(t, strings.NewReader(""), "lint", "--quiet", "--timing", xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code)
	require.NotContains(t, errOut, "took")
}

func TestXPathMaxInputBytesExceeded(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><book/></root>`)

	_, errOut, code := executeArgs(t, strings.NewReader(""), "xpath", "--max-input-bytes", "10", "count(//book)", xmlFile)
	require.Equal(t, heliumcmd.ExitReadFile, code)
	require.Contains(t, errOut, "exceeds maximum size")
}
