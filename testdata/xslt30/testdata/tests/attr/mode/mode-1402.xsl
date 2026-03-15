<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test that mode="#current" in an attribute set retains the current mode (XSLT 2.0, clarfied by erratum XT.E19) -->

<xsl:template match="book">
  <out>
    <xsl:apply-templates select="chapter" mode="m"/>
  </out>
</xsl:template>

<xsl:template match="chapter" mode="m">
  <chapter xsl:use-attribute-sets="a"/>
</xsl:template>

<xsl:attribute-set name="a">
  <xsl:attribute name="a">
    <xsl:apply-templates select="chtitle" mode="#current"/>
  </xsl:attribute>
</xsl:attribute-set>

<xsl:template match="chtitle" mode="#default">
  <xsl:text>#default:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="chtitle" mode="m">
  <xsl:text>m:</xsl:text><xsl:value-of select="."/>
</xsl:template>




</xsl:stylesheet>
