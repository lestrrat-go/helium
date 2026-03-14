<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- FileName: imp26d -->
<!-- Purpose: Used indirectly by impincl26 -->
<!-- No template for middle -->

<xsl:template match="outer">
  <D>
    <xsl:value-of select="name(.)"/>
    <xsl:text> Switching to middle...
</xsl:text>
    <xsl:apply-templates select="middle"/>
  </D>
</xsl:template>

<xsl:template match="inner">
  <D>
    <xsl:value-of select="name(.)"/>
  </D>
</xsl:template>

</xsl:stylesheet>