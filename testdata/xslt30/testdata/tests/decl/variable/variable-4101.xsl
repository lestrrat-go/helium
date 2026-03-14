<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#with-param?>
    <!-- Purpose: Test for passing param containing number via apply-templates -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="a">
      <xsl:with-param name="nums" select="a/text()"/>
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="a">
  <xsl:param name="nums">Default text in param 1</xsl:param>
  <list>
    <xsl:for-each select="$nums">
      <xsl:value-of select=".+6"/>
      <xsl:text>,</xsl:text>
    </xsl:for-each>
  </list>
</xsl:template>

</xsl:stylesheet>