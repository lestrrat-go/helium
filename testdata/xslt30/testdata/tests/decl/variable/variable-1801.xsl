<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#variable-values?>
  <!-- Purpose: Test setting several parameters locally -->
  

<xsl:template match="doc">
  <xsl:param name="n1" select="1"/>
  <xsl:param name="n2" select="2"/>
  <xsl:param name="n3" select="3"/>
  <xsl:param name="n4" select="4"/>
  <xsl:param name="n5" select="5"/>
  <xsl:param name="n6" select="6"/>
  <xsl:param name="n7" select="7"/>
  <xsl:param name="n8" select="8"/>
  <xsl:param name="n9" select="9"/>
  <out>
    <xsl:value-of select="item[$n8]"/>
    <xsl:value-of select="item[$n3]"/>
    <xsl:value-of select="item[$n7]"/>
    <xsl:value-of select="item[$n4]"/>
    <xsl:value-of select="item[$n1]"/>
    <xsl:value-of select="item[$n6]"/>
    <xsl:value-of select="item[$n5]"/>
    <xsl:value-of select="item[$n9]"/>
    <xsl:value-of select="item[$n2]"/>
  </out>
</xsl:template>

</xsl:stylesheet>
