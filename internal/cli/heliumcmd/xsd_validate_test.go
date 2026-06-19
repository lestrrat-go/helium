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

func TestRunXSDValidateVersion(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, flagVersion})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Contains(t, stderr.String(), "using helium")
}

func TestXSDValidateMissingSchemaArg(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate})
	require.Equal(t, heliumcmd.ExitErr, code)
	require.Contains(t, stderr.String(), "schema is required")
}

func TestXSDValidateUnknownOption(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, "--schema"})
	require.Equal(t, heliumcmd.ExitErr, code)
	require.Contains(t, stderr.String(), "unrecognized option --schema")
}

func TestXSDValidateValid(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>ok</root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitOK, code)
}

func TestXSDValidateInvalid(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:integer"/>
</xs:schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>bad</root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitValidation, code)
}

func TestXSDValidateSchemaCompileError(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?><not-schema/>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitSchemaComp, code)
}

func TestXSDValidateDuplicateGlobalElementDiagnostic(t *testing.T) {
	// A duplicate global element declaration is a fatal compile error. The CLI
	// must compile with an ErrorHandler so the diagnostic detail (schema file +
	// the offending element) reaches stderr, not just a bare "failed to compile
	// schema" summary.
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>ok</root>`)

	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitSchemaComp, code)
	out := stderr.String()
	require.Contains(t, out, "schema.xsd", "diagnostic should name the schema file")
	require.Contains(t, out, "root", "diagnostic should name the duplicate element")
	require.Contains(t, out, "does already exist", "diagnostic detail should reach stderr")
}

func TestXSDValidateFileReadError(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, schemaFile, filepath.Join(dir, "missing.xml")})
	require.Equal(t, heliumcmd.ExitReadFile, code)
}

func TestXSDValidateParseError(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitErr, code)
}

func TestXSDValidateMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	validXML := writeFile(t, dir, "valid.xml", `<?xml version="1.0"?><root>ok</root>`)
	invalidXML := writeFile(t, dir, "invalid.xml", `<?xml version="1.0"?><root><child/></root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, schemaFile, validXML, invalidXML})
	require.Equal(t, heliumcmd.ExitValidation, code)
}

func TestXSDValidateMaxInputBytes(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
	xml := `<?xml version="1.0"?><root>ok</root>`

	t.Run("file over cap rejected", func(t *testing.T) {
		xmlFile := writeFile(t, dir, "over.xml", xml)
		var stderr bytes.Buffer
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
		ctx = heliumcmd.WithStdinTTY(ctx, true)

		code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, flagMaxInput, "5", schemaFile, xmlFile})
		require.Equal(t, heliumcmd.ExitReadFile, code)
		require.Contains(t, stderr.String(), "exceeds maximum size")
	})

	t.Run("stdin over cap rejected", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(xml), io.Discard, &stderr)

		code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, flagMaxInput, "5", schemaFile})
		require.Equal(t, heliumcmd.ExitReadFile, code)
		require.Contains(t, stderr.String(), "exceeds maximum size")
	})

	t.Run("within cap ok", func(t *testing.T) {
		xmlFile := writeFile(t, dir, "within.xml", xml)
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
		ctx = heliumcmd.WithStdinTTY(ctx, true)

		code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, flagMaxInput, "100000", schemaFile, xmlFile})
		require.Equal(t, heliumcmd.ExitOK, code)
	})
}

func TestXSDValidateStdIn(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.xsd", `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)

	ctx := heliumcmd.WithIO(
		t.Context(),
		strings.NewReader(`<?xml version="1.0"?><root>ok</root>`),
		io.Discard,
		io.Discard,
	)

	code := heliumcmd.Execute(ctx, []string{cmdXSD, cmdValidate, schemaFile})
	require.Equal(t, heliumcmd.ExitOK, code)
}
