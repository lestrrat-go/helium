<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#variable-values?>
  <!-- Purpose: Test for setting a param using a choose -->
  

<xsl:param name="chooseTest" select="'ABC'"/>

<xsl:template match="/">
  <out>
    <xsl:value-of select="$chooseTest"/><xsl:text>_</xsl:text>
    <xsl:apply-templates select="doc"/>
  </out>
</xsl:template>

<xsl:template match="doc">
  <xsl:param name="chooseTest">
    <xsl:choose>
      <xsl:when test="item='X'">24</xsl:when>
      <xsl:when test="item='Y'">25</xsl:when>
      <xsl:when test="item='Z'">26</xsl:when>
      <xsl:otherwise>32</xsl:otherwise>
    </xsl:choose>
  </xsl:param>
  <xsl:value-of select="$chooseTest"/>
</xsl:template>

</xsl:stylesheet>