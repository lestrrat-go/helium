package main

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestXSDValidateCommand() *xsdValidateCommand {
	return &xsdValidateCommand{
		prog:     "helium xsd validate",
		stdin:    strings.NewReader(""),
		stderr:   io.Discard,
		stdinTTY: true,
	}
}

func TestRunXSDValidateVersion(t *testing.T) {
	require.Equal(t, ExitOK, run([]string{"xsd", "validate", "--version"}))
}

func TestParseXSDValidateArgs(t *testing.T) {
	cmd := newTestXSDValidateCommand()

	cfg, files := cmd.parseArgs([]string{"--schema", "test.xsd", "one.xml", "two.xml"})
	require.NotNil(t, cfg)
	require.Equal(t, "test.xsd", cfg.schemaFile)
	require.Equal(t, []string{"one.xml", "two.xml"}, files)
}

func TestParseXSDValidateArgsMissingSchema(t *testing.T) {
	cmd := newTestXSDValidateCommand()

	cfg, files := cmd.parseArgs([]string{"doc.xml"})
	require.Nil(t, cfg)
	require.Nil(t, files)
}

func TestXSDValidateValid(t *testing.T) {
	cmd := newTestXSDValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>ok</root>`)

	code := cmd.run([]string{"--schema", schemaFile, xmlFile})
	require.Equal(t, ExitOK, code)
}

func TestXSDValidateInvalid(t *testing.T) {
	cmd := newTestXSDValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:integer"/>
</xs:schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>bad</root>`)

	code := cmd.run([]string{"--schema", schemaFile, xmlFile})
	require.Equal(t, ExitValidation, code)
}

func TestXSDValidateSchemaCompileError(t *testing.T) {
	cmd := newTestXSDValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?><not-schema/>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	code := cmd.run([]string{"--schema", schemaFile, xmlFile})
	require.Equal(t, ExitSchemaComp, code)
}

func TestXSDValidateFileReadError(t *testing.T) {
	cmd := newTestXSDValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)

	code := cmd.run([]string{"--schema", schemaFile, filepath.Join(dir, "missing.xml")})
	require.Equal(t, ExitReadFile, code)
}

func TestXSDValidateParseError(t *testing.T) {
	cmd := newTestXSDValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<root>`)

	code := cmd.run([]string{"--schema", schemaFile, xmlFile})
	require.Equal(t, ExitErr, code)
}

func TestXSDValidateMultipleFiles(t *testing.T) {
	cmd := newTestXSDValidateCommand()
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	validXML := writeFile(t, dir, "valid.xml", `<?xml version="1.0"?><root>ok</root>`)
	invalidXML := writeFile(t, dir, "invalid.xml", `<?xml version="1.0"?><root><child/></root>`)

	code := cmd.run([]string{"--schema", schemaFile, validXML, invalidXML})
	require.Equal(t, ExitValidation, code)
}

func TestXSDValidateVersionWritesToStderr(t *testing.T) {
	var errOut strings.Builder
	cmd := &xsdValidateCommand{
		prog:     "helium xsd validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.run([]string{"--version"})
	require.Equal(t, ExitOK, code)
	require.Contains(t, errOut.String(), "helium version")
}

func TestXSDValidateStdIn(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)

	cmd := &xsdValidateCommand{
		prog:     "helium xsd validate",
		stdin:    strings.NewReader(`<?xml version="1.0"?><root>ok</root>`),
		stderr:   io.Discard,
		stdinTTY: false,
	}

	code := cmd.run([]string{"--schema", schemaFile})
	require.Equal(t, ExitOK, code)
}

func TestXSDValidateMissingSchemaArg(t *testing.T) {
	var errOut strings.Builder
	cmd := &xsdValidateCommand{
		prog:     "helium xsd validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.run([]string{"doc.xml"})
	require.Equal(t, ExitErr, code)
	require.Contains(t, errOut.String(), "--schema is required")
}

func TestXSDValidateMissingSchemaValue(t *testing.T) {
	var errOut strings.Builder
	cmd := &xsdValidateCommand{
		prog:     "helium xsd validate",
		stdin:    strings.NewReader(""),
		stderr:   &errOut,
		stdinTTY: true,
	}

	code := cmd.run([]string{"--schema"})
	require.Equal(t, ExitErr, code)
	require.Contains(t, errOut.String(), "--schema requires an argument")
}
