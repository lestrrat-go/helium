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

func TestRunRelaxNGValidateVersion(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", "--version"})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Contains(t, stderr.String(), "using helium")
}

func TestRelaxNGValidateMissingSchemaArg(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate"})
	require.Equal(t, heliumcmd.ExitErr, code)
	require.Contains(t, stderr.String(), "schema is required")
}

func TestRelaxNGValidateUnknownOption(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", "--schema"})
	require.Equal(t, heliumcmd.ExitErr, code)
	require.Contains(t, stderr.String(), "unrecognized option --schema")
}

func TestRelaxNGValidateValid(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>ok</root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitOK, code)
}

func TestRelaxNGValidateInvalid(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root><child/></root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitValidation, code)
}

func TestRelaxNGValidateSchemaCompileError(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", filepath.Join(dir, "missing.rng"), xmlFile})
	require.Equal(t, heliumcmd.ExitSchemaComp, code)
}

func TestRelaxNGValidateFileReadError(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", schemaFile, filepath.Join(dir, "missing.xml")})
	require.Equal(t, heliumcmd.ExitReadFile, code)
}

func TestRelaxNGValidateParseError(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitErr, code)
}

func TestRelaxNGValidateMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	validXML := writeFile(t, dir, "valid.xml", `<?xml version="1.0"?><root>ok</root>`)
	invalidXML := writeFile(t, dir, "invalid.xml", `<?xml version="1.0"?><root><child/></root>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", schemaFile, validXML, invalidXML})
	require.Equal(t, heliumcmd.ExitValidation, code)
}

func TestRelaxNGValidateStdIn(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)

	ctx := heliumcmd.WithIO(
		t.Context(),
		strings.NewReader(`<?xml version="1.0"?><root>ok</root>`),
		io.Discard,
		io.Discard,
	)

	code := heliumcmd.Execute(ctx, []string{"relaxng", "validate", schemaFile})
	require.Equal(t, heliumcmd.ExitOK, code)
}
