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

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, flagVersion})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Contains(t, stderr.String(), "using helium")
}

func TestSchematronValidateMissingSchemaArg(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate})
	require.Equal(t, heliumcmd.ExitErr, code)
	require.Contains(t, stderr.String(), "schema is required")
}

func TestSchematronValidateUnknownOption(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, "--schema"})
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

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, schemaFile, xmlFile})
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

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, schemaFile, xmlFile})
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

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, filepath.Join(dir, "missing.sch"), xmlFile})
	require.Equal(t, heliumcmd.ExitSchemaComp, code)
}

func TestSchematronValidateCompileDiagnostics(t *testing.T) {
	dir := t.TempDir()
	// A pattern with no rule element triggers a fatal compile diagnostic.
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
  </pattern>
</schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitSchemaComp, code)
	require.Contains(t, stderr.String(), "Pattern has no rule element")
	require.Contains(t, stderr.String(), "failed to compile schema")
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

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, schemaFile, filepath.Join(dir, "missing.xml")})
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

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, schemaFile, xmlFile})
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

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, schemaFile, validXML, invalidXML})
	require.Equal(t, heliumcmd.ExitValidation, code)
}

func TestSchematronValidateMaxInputBytes(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.sch", `<?xml version="1.0"?>
<schema xmlns="http://www.ascc.net/xml/schematron">
  <pattern>
    <rule context="root">
      <assert test="child">child element is required</assert>
    </rule>
  </pattern>
</schema>`)
	xml := `<?xml version="1.0"?><root><child/></root>`

	t.Run("file over cap rejected", func(t *testing.T) {
		xmlFile := writeFile(t, dir, "over.xml", xml)
		var stderr bytes.Buffer
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
		ctx = heliumcmd.WithStdinTTY(ctx, true)

		code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, flagMaxInput, "5", schemaFile, xmlFile})
		require.Equal(t, heliumcmd.ExitReadFile, code)
		require.Contains(t, stderr.String(), "exceeds maximum size")
	})

	t.Run("stdin over cap rejected", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(xml), io.Discard, &stderr)

		code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, flagMaxInput, "5", schemaFile})
		require.Equal(t, heliumcmd.ExitReadFile, code)
		require.Contains(t, stderr.String(), "exceeds maximum size")
	})

	t.Run("within cap ok", func(t *testing.T) {
		xmlFile := writeFile(t, dir, "within.xml", xml)
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
		ctx = heliumcmd.WithStdinTTY(ctx, true)

		code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, flagMaxInput, "100000", schemaFile, xmlFile})
		require.Equal(t, heliumcmd.ExitOK, code)
	})
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

	code := heliumcmd.Execute(ctx, []string{cmdSchematron, cmdValidate, schemaFile})
	require.Equal(t, heliumcmd.ExitOK, code)
}
