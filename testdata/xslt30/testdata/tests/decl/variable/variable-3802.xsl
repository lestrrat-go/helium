<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#with-param?>
  <!-- Purpose: Test for passing param containing 0-element node-set via apply-templates -->

  <xsl:output method="xml" encoding="UTF-8"/>

  <xsl:template match="doc">
    <out>
      <size>
        <xsl:value-of select="count(a/comment())"/>
      </size>
      <xsl:apply-templates select="a">
        <xsl:with-param name="comments" select="a/comment()"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>

  <xsl:template match="a">
    <xsl:param name="comments">Default text in param 1</xsl:param>
    <list>
      <xsl:for-each select="$comments">
        <xsl:value-of select="."/>
        <xsl:text>,</xsl:text>
      </xsl:for-each>
    </list>
  </xsl:template>

</xsl:stylesheet>
