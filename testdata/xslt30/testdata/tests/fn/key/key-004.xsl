<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test for xsl:key, where value of use is an integer. -->

  <xsl:key name="mykey" match="div" use="number(q)"/>

  <xsl:template match="doc">
    <out>
      <xsl:value-of select="key('mykey', 1 )/p"/>
      <xsl:text>,</xsl:text>
      <xsl:value-of select="key('mykey', 3.0 )/p"/>
      <xsl:text>,</xsl:text>
      <xsl:value-of select="key('mykey', 1+1 )/p"/>
      <xsl:text>,</xsl:text>
      <xsl:value-of select="key('mykey', 3.7 )/p"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
