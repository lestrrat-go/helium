<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#global-variables?>
  <!-- Purpose: Set cascaded top-level variables in arbitrary order. -->
  

<xsl:variable name="b" select="$a"/>

<xsl:variable name="e" select="$d"/>

<xsl:variable name="a" select="/doc/mid/inner"/>

<xsl:variable name="d" select="$c"/>

<xsl:variable name="c" select="$b"/>

<xsl:template match="/">
  <out>
    <xsl:text>
a= </xsl:text><xsl:value-of select="$a"/>
    <xsl:text>
b= </xsl:text><xsl:value-of select="$b"/>
    <xsl:text>
c= </xsl:text><xsl:value-of select="$c"/>
    <xsl:text>
d= </xsl:text><xsl:value-of select="$d"/>
    <xsl:text>
e= </xsl:text><xsl:value-of select="$e"/>
  </out>
</xsl:template>

</xsl:stylesheet>
