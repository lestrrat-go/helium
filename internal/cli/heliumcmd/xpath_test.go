package heliumcmd

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestXPathCommand() *xpathCommand {
	return &xpathCommand{
		prog:     "helium xpath",
		stdin:    strings.NewReader(""),
		stdout:   io.Discard,
		stderr:   io.Discard,
		stdinTTY: true,
	}
}

func TestRunXPathVersion(t *testing.T) {
	require.Equal(t, ExitOK, Execute(newExecuteTestContext(), []string{"xpath", "--version"}))
}

func TestParseXPathArgsDefaults(t *testing.T) {
	cmd := newTestXPathCommand()

	cfg, files := cmd.parseArgs([]string{"//book", "doc.xml"})
	require.NotNil(t, cfg)
	require.Equal(t, "3", cfg.engine)
	require.Equal(t, "//book", cfg.expr)
	require.Equal(t, []string{"doc.xml"}, files)
}

func TestParseXPathArgsEngine1(t *testing.T) {
	cmd := newTestXPathCommand()

	cfg, files := cmd.parseArgs([]string{"--engine", "1", "count(//book)", "doc.xml"})
	require.NotNil(t, cfg)
	require.Equal(t, "1", cfg.engine)
	require.Equal(t, "count(//book)", cfg.expr)
	require.Equal(t, []string{"doc.xml"}, files)
}

func TestParseXPathArgsMissingExpr(t *testing.T) {
	cmd := newTestXPathCommand()

	cfg, files := cmd.parseArgs(nil)
	require.Nil(t, cfg)
	require.Nil(t, files)
}

func TestParseXPathArgsInvalidEngine(t *testing.T) {
	cmd := newTestXPathCommand()

	cfg, files := cmd.parseArgs([]string{"--engine", "2", "//book"})
	require.Nil(t, cfg)
	require.Nil(t, files)
}

func TestXPathEngine1File(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><book>one</book><book>two</book></root>`)
	var out strings.Builder

	cmd := &xpathCommand{
		prog:     "helium xpath",
		stdin:    strings.NewReader(""),
		stdout:   &out,
		stderr:   io.Discard,
		stdinTTY: true,
	}

	code := cmd.runContext(context.Background(), []string{"--engine", "1", "count(//book)", xmlFile})
	require.Equal(t, ExitOK, code)
	require.Equal(t, "2\n", out.String())
}

func TestXPathEngine3DefaultFile(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><book>one</book><book>two</book></root>`)
	var out strings.Builder

	cmd := &xpathCommand{
		prog:     "helium xpath",
		stdin:    strings.NewReader(""),
		stdout:   &out,
		stderr:   io.Discard,
		stdinTTY: true,
	}

	code := cmd.runContext(context.Background(), []string{"count(//book)", xmlFile})
	require.Equal(t, ExitOK, code)
	require.Equal(t, "2\n", out.String())
}

func TestXPathEngine3StdInXML(t *testing.T) {
	var out strings.Builder
	cmd := &xpathCommand{
		prog:     "helium xpath",
		stdin:    strings.NewReader(`<?xml version="1.0"?><root><book>one</book></root>`),
		stdout:   &out,
		stderr:   io.Discard,
		stdinTTY: false,
	}

	code := cmd.runContext(context.Background(), []string{"//book"})
	require.Equal(t, ExitOK, code)
	require.Contains(t, out.String(), "<book>one</book>")
}

func TestXPathEngine3AtomicSequence(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)
	var out strings.Builder

	cmd := &xpathCommand{
		prog:     "helium xpath",
		stdin:    strings.NewReader(""),
		stdout:   &out,
		stderr:   io.Discard,
		stdinTTY: true,
	}

	code := cmd.runContext(context.Background(), []string{"(1,2,3)", xmlFile})
	require.Equal(t, ExitOK, code)
	require.Equal(t, "1\n2\n3\n", out.String())
}

func TestXPathInvalidExpression(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	cmd := newTestXPathCommand()
	code := cmd.runContext(context.Background(), []string{"///invalid[[[", xmlFile})
	require.Equal(t, ExitXPath, code)
}

func TestXPathReadFileError(t *testing.T) {
	cmd := newTestXPathCommand()
	code := cmd.runContext(context.Background(), []string{"//book", "/missing.xml"})
	require.Equal(t, ExitReadFile, code)
}

func TestXPathParseError(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<root>`)

	cmd := newTestXPathCommand()
	code := cmd.runContext(context.Background(), []string{"//book", xmlFile})
	require.Equal(t, ExitErr, code)
}
