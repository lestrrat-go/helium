<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-sum?>
    <!-- Purpose: Test of sum(). -->

<xsl:template match="doc">
  <out><xsl:text>&#10;</xsl:text>
    <xsl:variable name="rtf" select="av//h"/>
	<xsl:copy-of select="$rtf"/><xsl:text>&#10;</xsl:text>
    <xsl:value-of select="sum($rtf)"/><xsl:text>&#10;</xsl:text>
	<xsl:value-of select="$rtf[1]"/><xsl:text>&#10;</xsl:text>
  </out>

</xsl:template>

</xsl:stylesheet>
