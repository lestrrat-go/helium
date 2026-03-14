package heliumcmd

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newExecuteTestContext() context.Context {
	ctx := WithIO(context.Background(), strings.NewReader(""), io.Discard, io.Discard)
	return WithStdinTTY(ctx, true)
}

func TestRunNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), nil))
}

func TestRunUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), []string{"xslt"}))
}

func TestRunLintVersion(t *testing.T) {
	require.Equal(t, ExitOK, Execute(newExecuteTestContext(), []string{"lint", "--version"}))
}

func TestRunXPathNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), []string{"xpath"}))
}

func TestRunRelaxNGNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), []string{"relaxng"}))
}

func TestRunSchematronNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), []string{"schematron"}))
}

func TestRunXSDNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), []string{"xsd"}))
}

func TestRunRelaxNGUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), []string{"relaxng", "compile"}))
}

func TestRunSchematronUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), []string{"schematron", "compile"}))
}

func TestRunXSDUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, Execute(newExecuteTestContext(), []string{"xsd", "compile"}))
}
