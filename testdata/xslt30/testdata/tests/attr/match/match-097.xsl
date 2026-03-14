<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Test of pattern matching with operator "ge" apperaing inside filter -->

<xsl:template match="doc">
	<out>
		<xsl:apply-templates select="*"/>
	</out>
</xsl:template>

<xsl:template match="text-elem[position() ge 1 or position() ge 2]">
  <t>
	<xsl:value-of select="string(.)"/>
  </t>
</xsl:template>

<xsl:template match="text()"/>


</xsl:stylesheet>
