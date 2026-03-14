package heliumcmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), nil))
}

func TestRunUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), []string{"xslt"}))
}

func TestRunLintVersion(t *testing.T) {
	require.Equal(t, ExitOK, Execute(context.Background(), []string{"lint", "--version"}))
}

func TestRunXPathNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), []string{"xpath"}))
}

func TestRunRelaxNGNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), []string{"relaxng"}))
}

func TestRunSchematronNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), []string{"schematron"}))
}

func TestRunXSDNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), []string{"xsd"}))
}

func TestRunRelaxNGUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), []string{"relaxng", "compile"}))
}

func TestRunSchematronUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), []string{"schematron", "compile"}))
}

func TestRunXSDUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, Execute(context.Background(), []string{"xsd", "compile"}))
}
