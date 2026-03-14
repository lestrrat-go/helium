<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- Purpose: Test of pattern matching with operator "lt" apperaing inside predicate -->

<xsl:template match="doc">
	<out><xsl:apply-templates select="*"/></out>
</xsl:template>

<xsl:template match="text-elem[position() lt 2 and position() lt 3]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text-elem[not (xs:integer(@att) lt 4)]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>
