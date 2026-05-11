package heliumcmd_test

import (
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

func executeDiscard(t *testing.T, args []string) int {
	t.Helper()
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)
	return heliumcmd.Execute(ctx, args)
}

func TestRunNoArgs(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, nil))
}

func TestRunUnknownSubcommand(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, []string{"xslt"}))
}

func TestRunLintVersion(t *testing.T) {
	require.Equal(t, heliumcmd.ExitOK, executeDiscard(t, []string{"lint", flagVersion}))
}

func TestRunXPathNoArgs(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, []string{cmdXPath}))
}

func TestRunRelaxNGNoArgs(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, []string{cmdRelaxNG}))
}

func TestRunSchematronNoArgs(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, []string{cmdSchematron}))
}

func TestRunXSDNoArgs(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, []string{cmdXSD}))
}

func TestRunRelaxNGUnknownSubcommand(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, []string{cmdRelaxNG, "compile"}))
}

func TestRunSchematronUnknownSubcommand(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, []string{cmdSchematron, "compile"}))
}

func TestRunXSDUnknownSubcommand(t *testing.T) {
	require.Equal(t, heliumcmd.ExitErr, executeDiscard(t, []string{cmdXSD, "compile"}))
}

func TestExecuteWithInjectedStdinDefaultsToNonTTY(t *testing.T) {
	var stdout strings.Builder
	ctx := heliumcmd.WithIO(
		t.Context(),
		strings.NewReader(`<?xml version="1.0"?><root><book/></root>`),
		&stdout,
		io.Discard,
	)

	code := heliumcmd.Execute(ctx, []string{cmdXPath, "count(//book)"})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Equal(t, "1\n", stdout.String())
}
