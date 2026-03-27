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

func TestMergeExitCode(t *testing.T) {
	// mergeExitCode takes the highest exit code.
	// Test indirectly via Execute with multiple files where one is missing and one is bad XML.
	// missing file → ExitReadFile(4), bad XML → ExitErr(1).
	// The highest (ExitReadFile=4) should win.
	dir := t.TempDir()
	badXML := filepath.Join(dir, "bad.xml")
	require.NoError(t, os.WriteFile(badXML, []byte(`<root>`), 0o600))

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"xpath", "//book", filepath.Join(dir, "missing.xml"), badXML})
	require.Equal(t, heliumcmd.ExitReadFile, code)
}

func TestXSDValidateMultipleFilesUsesHighestExitCode(t *testing.T) {
	dir := t.TempDir()
	schemaFile := filepath.Join(dir, "schema.xsd")
	require.NoError(t, os.WriteFile(schemaFile, []byte(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`), 0o600))

	invalidXML := filepath.Join(dir, "invalid.xml")
	require.NoError(t, os.WriteFile(invalidXML, []byte(`<?xml version="1.0"?><root><child/></root>`), 0o600))

	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"xsd", "validate", schemaFile, filepath.Join(dir, "missing.xml"), invalidXML})
	require.Equal(t, heliumcmd.ExitReadFile, code)
}
