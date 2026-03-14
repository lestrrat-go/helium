package heliumcmd

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeExitCode(t *testing.T) {
	require.Equal(t, ExitOK, mergeExitCode(ExitOK, ExitOK))
	require.Equal(t, ExitValidation, mergeExitCode(ExitOK, ExitValidation))
	require.Equal(t, ExitReadFile, mergeExitCode(ExitReadFile, ExitValidation))
}

func TestXPathMultipleFilesUsesHighestExitCode(t *testing.T) {
	dir := t.TempDir()
	badXML := writeFile(t, dir, "bad.xml", `<root>`)

	cmd := &xpathCommand{
		prog:     "helium xpath",
		stdin:    strings.NewReader(""),
		stdout:   io.Discard,
		stderr:   io.Discard,
		stdinTTY: true,
	}

	code := cmd.runContext(context.Background(), []string{"//book", filepath.Join(dir, "missing.xml"), badXML})
	require.Equal(t, ExitReadFile, code)
}

func TestXSDValidateMultipleFilesUsesHighestExitCode(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	invalidXML := writeFile(t, dir, "invalid.xml", `<?xml version="1.0"?><root><child/></root>`)

	cmd := &xsdValidateCommand{
		prog:     "helium xsd validate",
		stdin:    strings.NewReader(""),
		stderr:   io.Discard,
		stdinTTY: true,
	}

	code := cmd.runContext(context.Background(), []string{schemaFile, filepath.Join(dir, "missing.xml"), invalidXML})
	require.Equal(t, ExitReadFile, code)
}
