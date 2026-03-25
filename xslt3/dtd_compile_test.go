package xslt3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestCompileFileLoadsDTDDefinedExternalEntityInIncludedStylesheet(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.xsl"), []byte(`<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:include href="child.xsl"/>
  <xsl:template match="/">
    <out value="{$var}"/>
  </xsl:template>
</xsl:stylesheet>`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "child.xsl"), []byte(`<?xml version="1.0"?>
<!DOCTYPE xsl:stylesheet SYSTEM "child.dtd">
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  &inject;
</xsl:stylesheet>`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "child.dtd"), []byte(`<!ENTITY inject SYSTEM "inject.xsl">`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "inject.xsl"), []byte(`<?xml version="1.0"?>
<xsl:variable xmlns:xsl="http://www.w3.org/1999/XSL/Transform" name="var" select="'from-dtd-entity'"/>`), 0o644))

	mainPath := filepath.Join(tmpDir, "main.xsl")
	p := helium.NewParser().DTDLoad(true).NoEnt(true).BaseURI(mainPath)
	mainData, err := os.ReadFile(mainPath)
	require.NoError(t, err)
	doc, err := p.Parse(t.Context(), mainData)
	require.NoError(t, err)
	ss, err := xslt3.NewCompiler().BaseURI(mainPath).Compile(t.Context(), doc)
	require.NoError(t, err)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<doc/>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, result, `value="from-dtd-entity"`)
}
