package heliumcmd_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

func TestRunXPathVersion(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, flagVersion})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Contains(t, stderr.String(), "using helium")
}

func TestXPathMissingExpr(t *testing.T) {
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath})
	require.Equal(t, heliumcmd.ExitErr, code)
}

func TestXPathInvalidEngine(t *testing.T) {
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, "--engine", "2", xpathBook})
	require.Equal(t, heliumcmd.ExitErr, code)
}

func TestXPathEngine1File(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><book>one</book><book>two</book></root>`)

	var stdout bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), &stdout, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, "--engine", "1", "count(//book)", xmlFile})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Equal(t, "2\n", stdout.String())
}

func TestXPathEngine3DefaultFile(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><book>one</book><book>two</book></root>`)

	var stdout bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), &stdout, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, "count(//book)", xmlFile})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Equal(t, "2\n", stdout.String())
}

func TestXPathEngine3StdInXML(t *testing.T) {
	var stdout bytes.Buffer
	ctx := heliumcmd.WithIO(
		t.Context(),
		strings.NewReader(`<?xml version="1.0"?><root><book>one</book></root>`),
		&stdout,
		io.Discard,
	)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, xpathBook})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Contains(t, stdout.String(), "<book>one</book>")
}

func TestXPathEngine3AtomicSequence(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	var stdout bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), &stdout, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, "(1,2,3)", xmlFile})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Equal(t, "1\n2\n3\n", stdout.String())
}

func TestXPathInvalidExpression(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, "///invalid[[[", xmlFile})
	require.Equal(t, heliumcmd.ExitXPath, code)
}

func TestXPathReadFileError(t *testing.T) {
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, xpathBook, "/missing.xml"})
	require.Equal(t, heliumcmd.ExitReadFile, code)
}

func TestXPathParseError(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, xpathBook, xmlFile})
	require.Equal(t, heliumcmd.ExitErr, code)
}
