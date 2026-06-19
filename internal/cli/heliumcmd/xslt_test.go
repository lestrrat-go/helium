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
