<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <!-- Test of pattern matching with function id() including a variable reference -->

<xsl:variable name="temp" select="'elem_d'"/>

<xsl:template match="doc">
	<xsl:apply-templates/>
</xsl:template>

<xsl:template match="id($temp)">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()"/>

<xsl:template match="*"/>

</xsl:stylesheet>
