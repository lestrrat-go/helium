package heliumcmd_test

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

func TestRunSchematronValidateVersion(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", "--version"})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Contains(t, stderr.String(), "helium version")
}

func TestSchematronValidateMissingSchemaArg(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate"})
	require.Equal(t, heliumcmd.ExitErr, code)
	require.Contains(t, stderr.String(), "schema is required")
}

func TestSchematronValidateUnknownOption(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", "--schema"})
	require.Equal(t, heliumcmd.ExitErr, code)
	require.Contains(t, stderr.String(), "unrecognized option --schema")
}

func TestSchematronValidateValid(t *testing.T) {
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

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitOK, code)
}

func TestSchematronValidateInvalid(t *testing.T) {
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

	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitValidation, code)
	require.Contains(t, stderr.String(), "fails to validate")
	require.Contains(t, stderr.String(), "child element is required")
	require.Contains(t, stderr.String(), xmlFile)
}

func TestSchematronValidateSchemaCompileError(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", filepath.Join(dir, "missing.sch"), xmlFile})
	require.Equal(t, heliumcmd.ExitSchemaComp, code)
}

func TestSchematronValidateFileReadError(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">child element is required</assert>
    </rule>
  </pattern>
</schema>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", schemaFile, filepath.Join(dir, "missing.xml")})
	require.Equal(t, heliumcmd.ExitReadFile, code)
}

func TestSchematronValidateParseError(t *testing.T) {
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

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitErr, code)
}

func TestSchematronValidateMultipleFiles(t *testing.T) {
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

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", schemaFile, validXML, invalidXML})
	require.Equal(t, heliumcmd.ExitValidation, code)
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

	ctx := heliumcmd.WithIO(
		t.Context(),
		strings.NewReader(`<?xml version="1.0"?><root><child/></root>`),
		io.Discard,
		io.Discard,
	)

	code := heliumcmd.Execute(ctx, []string{"schematron", "validate", schemaFile})
	require.Equal(t, heliumcmd.ExitOK, code)
}
