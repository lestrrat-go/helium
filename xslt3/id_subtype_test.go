package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestAnnotateAttrRegistersIDSubtype(t *testing.T) {
	ss := compileStylesheetString(t, `
<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema>
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
      <xs:simpleType name="myID">
        <xs:restriction base="xs:ID"/>
      </xs:simpleType>
      <xs:complexType name="rootType">
        <xs:attribute name="id" type="myID"/>
      </xs:complexType>
      <xs:element name="root" type="rootType"/>
    </xs:schema>
  </xsl:import-schema>

  <xsl:template match="/">
    <result>
      <found><xsl:value-of select="boolean(id('alpha'))"/></found>
      <name><xsl:value-of select="id('alpha')/name()"/></name>
    </result>
  </xsl:template>
</xsl:stylesheet>`)

	source, err := helium.NewParser().Parse(t.Context(), []byte(`<root id="alpha"/>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)

	require.Contains(t, result, "<found>true</found>")
	require.Contains(t, result, "<name>root</name>")
}
