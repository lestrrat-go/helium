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

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, flagVersion})
	require.Equal(t, heliumcmd.ExitOK, code)
	require.Contains(t, stderr.String(), "using helium")
}

func TestRelaxNGValidateMissingSchemaArg(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate})
	require.Equal(t, heliumcmd.ExitErr, code)
	require.Contains(t, stderr.String(), "schema is required")
}

func TestRelaxNGValidateUnknownOption(t *testing.T) {
	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, "--schema"})
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

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, schemaFile, xmlFile})
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

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitValidation, code)
}

func TestRelaxNGValidateSchemaCompileError(t *testing.T) {
	dir := t.TempDir()
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root/>`)

	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, filepath.Join(dir, "missing.rng"), xmlFile})
	require.Equal(t, heliumcmd.ExitSchemaComp, code)
}

func TestRelaxNGValidateOversizedIncludeIsSchemaCompFailure(t *testing.T) {
	// An <include> whose target exceeds the per-resource byte cap is a fatal
	// compile diagnostic. The RELAX NG compiler may still return a (grammar,
	// nil) with a poisoned notAllowed grammar in that case; the CLI must
	// install an ErrorHandler so this is reported as a schema-compilation
	// failure (ExitSchemaComp) rather than being silently discarded and then
	// misreported as a per-input validation failure (ExitValidation).
	dir := t.TempDir()

	// defaultMaxResourceBytes in the relaxng package is 10 MiB; write an
	// included grammar that comfortably exceeds it.
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?>` + "\n")
	b.WriteString(`<grammar xmlns="http://relaxng.org/ns/structure/1.0">` + "\n")
	b.WriteString("<!-- ")
	b.WriteString(strings.Repeat("x", 11<<20))
	b.WriteString(" -->\n")
	b.WriteString(`<start><element name="root"><text/></element></start>` + "\n")
	b.WriteString(`</grammar>` + "\n")
	writeFile(t, dir, "big.rng", b.String())

	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <include href="big.rng"/>
</grammar>`)
	xmlFile := writeFile(t, dir, "doc.xml", `<?xml version="1.0"?><root>ok</root>`)

	var stderr bytes.Buffer
	ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
	ctx = heliumcmd.WithStdinTTY(ctx, true)

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, schemaFile, xmlFile})
	require.Equal(t, heliumcmd.ExitSchemaComp, code)
	out := stderr.String()
	require.Contains(t, out, "exceeds the maximum resource size", "fatal compile diagnostic should reach stderr")
	require.Contains(t, out, "failed to compile schema", "CLI should report schema-compilation failure")
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

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, schemaFile, filepath.Join(dir, "missing.xml")})
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

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, schemaFile, xmlFile})
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

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, schemaFile, validXML, invalidXML})
	require.Equal(t, heliumcmd.ExitValidation, code)
}

func TestRelaxNGValidateMaxInputBytes(t *testing.T) {
	dir := t.TempDir()
	schemaFile := writeFile(t, dir, "schema.rng", `<?xml version="1.0"?>
<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root"><text/></element>
  </start>
</grammar>`)
	xml := `<?xml version="1.0"?><root>ok</root>`

	t.Run("file over cap rejected", func(t *testing.T) {
		xmlFile := writeFile(t, dir, "over.xml", xml)
		var stderr bytes.Buffer
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, &stderr)
		ctx = heliumcmd.WithStdinTTY(ctx, true)

		code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, flagMaxInput, "5", schemaFile, xmlFile})
		require.Equal(t, heliumcmd.ExitReadFile, code)
		require.Contains(t, stderr.String(), "exceeds maximum size")
	})

	t.Run("stdin over cap rejected", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(xml), io.Discard, &stderr)

		code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, flagMaxInput, "5", schemaFile})
		require.Equal(t, heliumcmd.ExitReadFile, code)
		require.Contains(t, stderr.String(), "exceeds maximum size")
	})

	t.Run("within cap ok", func(t *testing.T) {
		xmlFile := writeFile(t, dir, "within.xml", xml)
		ctx := heliumcmd.WithIO(t.Context(), strings.NewReader(""), io.Discard, io.Discard)
		ctx = heliumcmd.WithStdinTTY(ctx, true)

		code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, flagMaxInput, "100000", schemaFile, xmlFile})
		require.Equal(t, heliumcmd.ExitOK, code)
	})
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

	code := heliumcmd.Execute(ctx, []string{cmdRelaxNG, cmdValidate, schemaFile})
	require.Equal(t, heliumcmd.ExitOK, code)
}
