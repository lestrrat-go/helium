package heliumcmd

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestRelaxNGValidateCommand() *relaxNGValidateCommand {
	return &relaxNGValidateCommand{
		prog:     "helium relaxng validate",
		stdin:    strings.NewReader(""),
		stderr:   io.Discard,
		stdinTTY: true,
	}
}

func TestRunRelaxNGValidateVersion(t *testing.T) {
	require.Equal(t, ExitOK, Execute(newExecuteTestContext(), []string{"relaxng", "validate", "--version"}))
}

func TestParseRelaxNGValidateArgs(t *testing.T) {
	cmd := newTestRelaxNGValidateCommand()

	cfg, files := cmd.parseArgs([]string{"test.rng", "one.xml", "two.xml"})
	require.NotNil(t, cfg)
	require.Equal(t, "test.rng", cfg.schemaFile)
	require.Equal(t, []string{"one.xml", "two.xml"}, files)
}

func TestParseRelaxNGValidateArgsMissingSchema(t *testing.T) {
	cmd := newTestRelaxNGValidateCommand()

	cfg, files := cmd.parseArgs([]string{})
	require.Nil(t, cfg)
	require.Nil(t, files)
}

func TestRelaxNGValidateValid(t *testing.T) {
	cmd := newTestRelaxNGValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>ok</root>`)

	code := cmd.runContext(context.Background(), []string{schemaFile, xmlFile})
	require.Equal(t, ExitOK, code)
}

func TestRelaxNGValidateInvalid(t *testing.T) {
	cmd := newTestRelaxNGValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><child/></root>`)

	code := cmd.runContext(context.Background(), []string{schemaFile, xmlFile})
	require.Equal(t, ExitValidation, code)
}

func TestRelaxNGValidateSchemaCompileError(t *testing.T) {
	cmd := newTestRelaxNGValidateCommand()
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	code := cmd.runContext(context.Background(), []string{filepath.Join(dir, "missing.rng"), xmlFile})
	require.Equal(t, ExitSchemaComp, code)
}

func TestRelaxNGValidateFileReadError(t *testing.T) {
	cmd := newTestRelaxNGValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)

	code := cmd.runContext(context.Background(), []string{schemaFile, filepath.Join(dir, "missing.xml")})
	require.Equal(t, ExitReadFile, code)
}

func TestRelaxNGValidateParseError(t *testing.T) {
	cmd := newTestRelaxNGValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<root>`)

	code := cmd.runContext(context.Background(), []string{schemaFile, xmlFile})
	require.Equal(t, ExitErr, code)
}

func TestRelaxNGValidateMultipleFiles(t *testing.T) {
	cmd := newTestRelaxNGValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	validXML := writeFile(t, dir, "valid.xml", `<?xml version="1.0"?><root>ok</root>`)
	invalidXML := writeFile(t, dir, "invalid.xml", `<?xml version="1.0"?><root><child/></root>`)

	code := cmd.runContext(context.Background(), []string{schemaFile, validXML, invalidXML})
	require.Equal(t, ExitValidation, code)
}

func TestRelaxNGValidateVersionWritesToStderr(t *testing.T) {
	var errOut strings.Builder
	cmd := &relaxNGValidateCommand{
		prog:     "helium relaxng validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.runContext(context.Background(), []string{"--version"})
	require.Equal(t, ExitOK, code)
	require.Contains(t, errOut.String(), "helium version")
}

func TestRelaxNGValidateStdIn(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)

	cmd := &relaxNGValidateCommand{
		prog:     "helium relaxng validate",
		stdin:    strings.NewReader(`<?xml version="1.0"?><root>ok</root>`),
		stderr:   io.Discard,
		stdinTTY: false,
	}

	code := cmd.runContext(context.Background(), []string{schemaFile})
	require.Equal(t, ExitOK, code)
}

func TestRelaxNGValidateMissingSchemaArg(t *testing.T) {
	var errOut strings.Builder
	cmd := &relaxNGValidateCommand{
		prog:     "helium relaxng validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.runContext(context.Background(), nil)
	require.Equal(t, ExitErr, code)
	require.Contains(t, errOut.String(), "schema is required")
}

func TestRelaxNGValidateUnknownOption(t *testing.T) {
	var errOut strings.Builder
	cmd := &relaxNGValidateCommand{
		prog:     "helium relaxng validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.runContext(context.Background(), []string{"--schema"})
	require.Equal(t, ExitErr, code)
	require.Contains(t, errOut.String(), "unrecognized option --schema")
}
