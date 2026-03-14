<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#with-param?>
  <!-- Purpose: Test for passing param containing 4-element node-set via apply-templates -->

  <xsl:output method="xml" encoding="UTF-8"/>

  <xsl:template match="doc">
    <out>
      <size>
        <xsl:value-of select="count(a-set/a/text())"/>
      </size>
      <xsl:apply-templates select="a-set">
        <xsl:with-param name="texts" select="a-set/a/text()"/>
      </xsl:apply-templates>
    </out>
  </xsl:template>

  <xsl:template match="a-set">
    <xsl:param name="texts">Default text in param 1</xsl:param>
    <list>
      <xsl:for-each select="$texts">
        <xsl:value-of select="."/>
        <xsl:text>,</xsl:text>
      </xsl:for-each>
    </list>
  </xsl:template>

</xsl:stylesheet>
