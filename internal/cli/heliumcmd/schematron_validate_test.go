package heliumcmd

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestSchematronValidateCommand() *schematronValidateCommand {
	return &schematronValidateCommand{
		prog:     "helium schematron validate",
		stdin:    strings.NewReader(""),
		stderr:   io.Discard,
		stdinTTY: true,
	}
}

func TestRunSchematronValidateVersion(t *testing.T) {
	require.Equal(t, ExitOK, Execute(context.Background(), []string{"schematron", "validate", "--version"}))
}

func TestParseSchematronValidateArgs(t *testing.T) {
	cmd := newTestSchematronValidateCommand()

	cfg, files := cmd.parseArgs([]string{"test.sch", "one.xml", "two.xml"})
	require.NotNil(t, cfg)
	require.Equal(t, "test.sch", cfg.schemaFile)
	require.Equal(t, []string{"one.xml", "two.xml"}, files)
}

func TestParseSchematronValidateArgsMissingSchema(t *testing.T) {
	cmd := newTestSchematronValidateCommand()

	cfg, files := cmd.parseArgs([]string{})
	require.Nil(t, cfg)
	require.Nil(t, files)
}

func TestSchematronValidateValid(t *testing.T) {
	cmd := newTestSchematronValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">child element is required</assert>
    </rule>
  </pattern>
</schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><child/></root>`)

	code := cmd.run([]string{schemaFile, xmlFile})
	require.Equal(t, ExitOK, code)
}

func TestSchematronValidateInvalid(t *testing.T) {
	var errOut strings.Builder
	cmd := &schematronValidateCommand{
		prog:     "helium schematron validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">child element is required</assert>
    </rule>
  </pattern>
</schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	code := cmd.run([]string{schemaFile, xmlFile})
	require.Equal(t, ExitValidation, code)
	require.Contains(t, errOut.String(), "fails to validate")
	require.Contains(t, errOut.String(), "child element is required")
	require.Contains(t, errOut.String(), xmlFile)
}

func TestSchematronValidateSchemaCompileError(t *testing.T) {
	cmd := newTestSchematronValidateCommand()
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	code := cmd.run([]string{filepath.Join(dir, "missing.sch"), xmlFile})
	require.Equal(t, ExitSchemaComp, code)
}

func TestSchematronValidateFileReadError(t *testing.T) {
	cmd := newTestSchematronValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">child element is required</assert>
    </rule>
  </pattern>
</schema>`)

	code := cmd.run([]string{schemaFile, filepath.Join(dir, "missing.xml")})
	require.Equal(t, ExitReadFile, code)
}

func TestSchematronValidateParseError(t *testing.T) {
	cmd := newTestSchematronValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">child element is required</assert>
    </rule>
  </pattern>
</schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<root>`)

	code := cmd.run([]string{schemaFile, xmlFile})
	require.Equal(t, ExitErr, code)
}

func TestSchematronValidateMultipleFiles(t *testing.T) {
	cmd := newTestSchematronValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">child element is required</assert>
    </rule>
  </pattern>
</schema>`)
	validXML := writeFile(t, dir, "valid.xml", `<?xml version="1.0"?><root><child/></root>`)
	invalidXML := writeFile(t, dir, "invalid.xml", `<?xml version="1.0"?><root/>`)

	code := cmd.run([]string{schemaFile, validXML, invalidXML})
	require.Equal(t, ExitValidation, code)
}

func TestSchematronValidateVersionWritesToStderr(t *testing.T) {
	var errOut strings.Builder
	cmd := &schematronValidateCommand{
		prog:     "helium schematron validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.run([]string{"--version"})
	require.Equal(t, ExitOK, code)
	require.Contains(t, errOut.String(), "helium version")
}

func TestSchematronValidateStdIn(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">child element is required</assert>
    </rule>
  </pattern>
</schema>`)

	cmd := &schematronValidateCommand{
		prog:     "helium schematron validate",
		stdin:    strings.NewReader(`<?xml version="1.0"?><root><child/></root>`),
		stderr:   io.Discard,
		stdinTTY: false,
	}

	code := cmd.run([]string{schemaFile})
	require.Equal(t, ExitOK, code)
}

func TestSchematronValidateMissingSchemaArg(t *testing.T) {
	var errOut strings.Builder
	cmd := &schematronValidateCommand{
		prog:     "helium schematron validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.run(nil)
	require.Equal(t, ExitErr, code)
	require.Contains(t, errOut.String(), "schema is required")
}

func TestSchematronValidateUnknownOption(t *testing.T) {
	var errOut strings.Builder
	cmd := &schematronValidateCommand{
		prog:     "helium schematron validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.run([]string{"--schema"})
	require.Equal(t, ExitErr, code)
	require.Contains(t, errOut.String(), "unrecognized option --schema")
}
