<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-lang?>
  <!-- PURPOSE: Test of lang() function -->
<xsl:template match="/">
	<out><xsl:apply-templates select="doc/p"/></out>
</xsl:template>

<xsl:template match="p">
	<xsl:value-of select="lang('en')"/><xsl:text>, </xsl:text>
</xsl:template>
 
</xsl:stylesheet>
