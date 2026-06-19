package heliumcmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/internal/cli/heliumcmd"
	"github.com/stretchr/testify/require"
)

func TestXSLTIncludeResolves(t *testing.T) {
	dir := t.TempDir()

	// Included module defines the template that produces output.
	writeFile(t, dir, "mod.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root"><out><xsl:value-of select="."/></out></xsl:template>
</xsl:stylesheet>`)

	ssFile := writeFile(t, dir, "main.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="mod.xsl"/>
</xsl:stylesheet>`)

	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root>hello</root>`)

	out, errOut, code := executeArgs(t, strings.NewReader(""), "xslt", ssFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
	require.Contains(t, out, "<out>hello</out>")
}

func TestXSLTIncludeFileURIResolves(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "mod.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root"><out><xsl:value-of select="."/></out></xsl:template>
</xsl:stylesheet>`)

	// A file: URI href must resolve to the local module rather than being
	// handed verbatim to os.Open.
	modURI := "file://" + filepath.ToSlash(filepath.Join(dir, "mod.xsl"))
	ssFile := writeFile(t, dir, "main.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="`+modURI+`"/>
</xsl:stylesheet>`)

	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root>hello</root>`)

	out, errOut, code := executeArgs(t, strings.NewReader(""), "xslt", ssFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
	require.Contains(t, out, "<out>hello</out>")
}

func TestXSLTIncludeRespectsMaxInputBytes(t *testing.T) {
	// An oversized xsl:include module must be rejected by --max-input-bytes:
	// the module is loaded through the URIResolver, which xslt3 drains with
	// io.ReadAll, so the cap has to be enforced inside the resolver itself.
	mod := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="root"><out><xsl:value-of select="."/></out></xsl:template>
  <!-- ` + strings.Repeat("padding ", 300) + ` -->
</xsl:stylesheet>`

	main := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:include href="mod.xsl"/>
</xsl:stylesheet>`

	xml := `<?xml version="1.0"?><root>hello</root>`

	t.Run("oversized include rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "mod.xsl", mod)
		ssFile := writeFile(t, dir, "main.xsl", main)
		xmlFile := writeFile(t, dir, "in.xml", xml)

		require.Greater(t, len(mod), 500, "module must exceed the cap for the test to be meaningful")
		require.Less(t, len(main), 500, "main stylesheet must be within the cap")

		_, errOut, code := executeArgs(t, strings.NewReader(""),
			"xslt", "--max-input-bytes", "500", ssFile, xmlFile)
		require.NotEqual(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
		require.Contains(t, errOut, "exceeds maximum size")
	})

	t.Run("within-cap include works", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "mod.xsl", mod)
		ssFile := writeFile(t, dir, "main.xsl", main)
		xmlFile := writeFile(t, dir, "in.xml", xml)

		out, errOut, code := executeArgs(t, strings.NewReader(""),
			"xslt", "--max-input-bytes", "100000", ssFile, xmlFile)
		require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
		require.Contains(t, out, "<out>hello</out>")
	})

	t.Run("unlimited include works", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "mod.xsl", mod)
		ssFile := writeFile(t, dir, "main.xsl", main)
		xmlFile := writeFile(t, dir, "in.xml", xml)

		out, errOut, code := executeArgs(t, strings.NewReader(""),
			"xslt", "--max-input-bytes", "0", ssFile, xmlFile)
		require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)
		require.Contains(t, out, "<out>hello</out>")
	})
}

func TestXSLTOutputNoOutRejected(t *testing.T) {
	dir := t.TempDir()
	ssFile := writeFile(t, dir, "main.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)
	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root/>`)
	outFile := filepath.Join(dir, "out.xml")
	require.NoError(t, os.WriteFile(outFile, []byte("KEEP"), 0o600))

	_, errOut, code := executeArgs(t, strings.NewReader(""), "xslt", "--noout", "--output", outFile, ssFile, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)
	require.Contains(t, errOut, "noout")

	got, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Equal(t, "KEEP", string(got))
}

func TestXSLTOutputOverInputRejected(t *testing.T) {
	dir := t.TempDir()
	ssFile := writeFile(t, dir, "main.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)
	content := `<?xml version="1.0"?><root>keepme</root>`
	xmlFile := writeFile(t, dir, "in.xml", content)

	_, errOut, code := executeArgs(t, strings.NewReader(""), "xslt", "--output", xmlFile, ssFile, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)
	require.Contains(t, errOut, "overwrite input")

	got, err := os.ReadFile(xmlFile)
	require.NoError(t, err)
	require.Equal(t, content, string(got))
}

func TestXSLTOutputOverRuntimeReadStylesheetSucceeds(t *testing.T) {
	// --output points at a stylesheet that the main transform loads at runtime
	// via fn:transform(map{'stylesheet-location':...}), i.e. AFTER the output
	// target is opened. The pre-flight collision check cannot catch this (the
	// inner stylesheet is not an input arg), so the temp-file-then-rename
	// scheme must keep inner.xsl intact until its runtime read completes.
	dir := t.TempDir()

	innerFile := writeFile(t, dir, "inner.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><inner><xsl:value-of select="/root"/></inner></xsl:template>
</xsl:stylesheet>`)

	ssFile := writeFile(t, dir, "main.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:fn="http://www.w3.org/2005/xpath-functions">
  <xsl:template match="/">
    <xsl:variable name="r" select="fn:transform(map{'stylesheet-location':'inner.xsl','source-node':/})"/>
    <xsl:copy-of select="$r?output"/>
  </xsl:template>
</xsl:stylesheet>`)

	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root>hello</root>`)

	_, errOut, code := executeArgs(t, strings.NewReader(""),
		"xslt", "--output", innerFile, ssFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)

	// inner.xsl was read intact at transform time, so the transform produced
	// real output, which was then published over inner.xsl.
	got, err := os.ReadFile(innerFile)
	require.NoError(t, err)
	require.Contains(t, string(got), "<inner>hello</inner>")
}

func TestXSLTOutputErrorLeavesTargetIntact(t *testing.T) {
	// A transform error must leave the pre-existing output target untouched:
	// the temp file is discarded and never renamed onto it.
	dir := t.TempDir()
	ssFile := writeFile(t, dir, "main.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:template match="/"><xsl:value-of select="1 div xs:integer('notanumber')"/></xsl:template>
</xsl:stylesheet>`)
	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root/>`)
	outFile := filepath.Join(dir, "out.xml")
	require.NoError(t, os.WriteFile(outFile, []byte("KEEP"), 0o600))

	_, _, code := executeArgs(t, strings.NewReader(""), "xslt", "--output", outFile, ssFile, xmlFile)
	require.NotEqual(t, heliumcmd.ExitOK, code)

	got, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Equal(t, "KEEP", string(got))
}

func TestXSLTOutputWritesToFile(t *testing.T) {
	dir := t.TempDir()
	ssFile := writeFile(t, dir, "main.xsl", `<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`)
	xmlFile := writeFile(t, dir, "in.xml", `<?xml version="1.0"?><root/>`)
	outFile := filepath.Join(dir, "result.xml")

	_, errOut, code := executeArgs(t, strings.NewReader(""), "xslt", "--output", outFile, ssFile, xmlFile)
	require.Equal(t, heliumcmd.ExitOK, code, "stderr: %s", errOut)

	got, err := os.ReadFile(outFile)
	require.NoError(t, err)
	require.Contains(t, string(got), "<out")
}
