package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, run(nil))
}

func TestRunUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, run([]string{"xslt"}))
}

func TestRunLintVersion(t *testing.T) {
	require.Equal(t, ExitOK, run([]string{"lint", "--version"}))
}

func TestRunXPathNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, run([]string{"xpath"}))
}

func TestRunRelaxNGNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, run([]string{"relaxng"}))
}

func TestRunXSDNoArgs(t *testing.T) {
	require.Equal(t, ExitErr, run([]string{"xsd"}))
}

func TestRunRelaxNGUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, run([]string{"relaxng", "compile"}))
}

func TestRunXSDUnknownSubcommand(t *testing.T) {
	require.Equal(t, ExitErr, run([]string{"xsd", "compile"}))
}
