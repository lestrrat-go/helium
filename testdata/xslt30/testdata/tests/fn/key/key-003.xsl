<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test for xsl:key, where value of use is a string constant. -->

  <xsl:key name="mykey1" match="div" use="'foo'"/>
  <xsl:key name="mykey2" match="div" use="number(q)"/>

  <xsl:template match="doc">
    <out>
      <xsl:value-of select="key('mykey1','foo' )/p"/>
      <xsl:text>,</xsl:text>
      <xsl:value-of select="key('mykey2', 1 )/p"/>
      <xsl:text>,</xsl:text>
    </out>
  </xsl:template>

</xsl:stylesheet>
