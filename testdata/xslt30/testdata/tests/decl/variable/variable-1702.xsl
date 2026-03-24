<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#local-variables?>
  <!-- Purpose: Verify that a variable in a for-each can take on a new value -->
  
  
  <!-- MHK: this is legal in 2.0 but not in 1.0 -->
  <?same-as-1.0 no?>

<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="doc">
  <xsl:variable name="var" select="'level1'"/>
  <xsl:text>begin doc template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
  <xsl:for-each select="a">
    <xsl:variable name="var" select="@pos"/>
    <xsl:text>inside for-each, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
  </xsl:for-each>
  <xsl:text>end doc template, $var=</xsl:text><xsl:value-of select="$var"/><xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="text()"/><!-- suppress empty lines -->

</xsl:stylesheet>