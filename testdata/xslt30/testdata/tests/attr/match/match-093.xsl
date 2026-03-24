<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs">

  <!-- Test of pattern with operator "ne" apperaing inside predicate filter -->

<xsl:template match="doc">
	<out><xsl:apply-templates select="*"/></out>
</xsl:template>

<xsl:template match="text-elem[position() ne 4 and position() ne 3 and position() ne 2]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text-elem[xs:integer(@att) ne 1]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()"/>


</xsl:stylesheet>
