<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <!-- Purpose: Test of patterns with operator "eq" appearing inside predicate filter -->

<xsl:template match="doc">
	<out><xsl:apply-templates select="*"/></out>
</xsl:template>

<xsl:template match="text-elem[position() eq 4]">
	<t>
		<xsl:value-of select="string(.)"/>
	</t>
</xsl:template>

<xsl:template match="text-elem[xs:integer(@att) eq 3]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text-elem[string(.) eq '11']">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()"/>


</xsl:stylesheet>
