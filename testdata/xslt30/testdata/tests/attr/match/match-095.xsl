<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <!-- Test of pattern matching with operator "le" apperaing inside predicate -->

<xsl:template match="doc">
	<out><xsl:apply-templates select="*"/></out>
</xsl:template>

<xsl:template match="text-elem[position() le 1 or position() le 3]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text-elem[not (xs:integer(@att) le 3)]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>
