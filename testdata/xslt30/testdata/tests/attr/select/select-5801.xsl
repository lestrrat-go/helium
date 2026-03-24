<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#node-tests?>
    <!-- Purpose: Test whether null string as attribute value causes selection problem. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="a">
  <xsl:value-of select="."/><xsl:text>-</xsl:text>
  <xsl:for-each select="attribute::*">
    <xsl:sort select="name(.)"/>
    <xsl:value-of select="."/><xsl:text>|</xsl:text>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
