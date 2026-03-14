<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-number?>
    <!-- Purpose: Test of number() conversion function for small decimal numbers. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
      <xsl:value-of select="number(1.75)"/>
      <xsl:text>|</xsl:text>
      <xsl:value-of select="number(7 div 4)"/>
      <xsl:text>|</xsl:text>
      <xsl:value-of select="(number(1.75) = (7 div 4))"/>
      <xsl:text>|&#10;</xsl:text>
      <xsl:value-of select="number(0.109375 * 16)"/>
      <xsl:text>|</xsl:text>
      <xsl:value-of select="(number(1.75) = (0.109375 * 16))"/>
      <xsl:text>|&#10;</xsl:text>
      <xsl:value-of select="number(k)"/>
      <xsl:text>|</xsl:text>
      <xsl:value-of select="number(4 div 10000)"/>
      <xsl:text>|</xsl:text>
      <xsl:value-of select="(number(k) = (4 div 10000))"/>
      <xsl:text>|&#10;</xsl:text>
      <xsl:value-of select="number(0.0001 * 4)"/>
      <xsl:text>|</xsl:text>
      <xsl:value-of select="(number(k) = (0.0001 * 4))"/>
      <xsl:text>|&#10;</xsl:text>
  </out>
</xsl:template>

</xsl:stylesheet>
