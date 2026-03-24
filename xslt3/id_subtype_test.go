package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

const idSubtypeStylesheet = `
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
    <out><xsl:value-of select="id('alpha')/name()"/></out>
  </xsl:template>
</xsl:stylesheet>`

func TestIDRecognizesSchemaValidatedIDSubtype(t *testing.T) {
	ss := compileStylesheetString(t, idSubtypeStylesheet)

	source, err := helium.Parse(t.Context(), []byte(`<root id="alpha"/>`))
	require.NoError(t, err)

	result, err := xslt3.TransformString(t.Context(), source, ss)
	require.NoError(t, err)
	require.Contains(t, result, ">root</out>")
}
