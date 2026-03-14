<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <!-- Test of pattern matching with operator "gt" apperaing inside filter -->

<xsl:template match="doc">
	<out>
		<xsl:apply-templates select="*"/>
	</out>
</xsl:template>

<xsl:template match="text-elem[position() gt 1 and position() gt 2]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text-elem[not (xs:integer(@att) gt 2)]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>
