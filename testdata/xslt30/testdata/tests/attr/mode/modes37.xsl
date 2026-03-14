<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test that mode="#current" in a global variable uses the unnamed mode (XSLT 2.0, clarified by erratum XT.E19) -->

<xsl:template match="book" mode="#all">
  <out>
    <xsl:apply-templates select="chapter" mode="m"/>
  </out>
</xsl:template>

<xsl:template match="chapter" mode="m">
  <chapter a="{$glob}"/>
</xsl:template>

<xsl:template match="chtitle" mode="#default">
  <xsl:text>#default:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="chtitle" mode="m">
  <xsl:text>m:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:variable name="glob">
  <xsl:apply-templates select="(//chtitle)[1]" mode="#current"/>
</xsl:variable>


</xsl:stylesheet>
