<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: test with-param as string copied from global variable (an RTF) -->

<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:variable name="ax"><xsl:value-of select="//Test/@a"/>x</xsl:variable>
<xsl:variable name="by"><xsl:value-of select="//Test/@b"/>y</xsl:variable>

<xsl:template match="/">
  <out>
    <xsl:call-template name="preview"/>

    <xsl:call-template name="doNothing">
      <xsl:with-param name="dummy" select="string($by)"/>
    </xsl:call-template>

    <xsl:text>
</xsl:text>
    <after>ax="<xsl:value-of select='$ax'/>"</after>
  </out>
</xsl:template>

<xsl:template name="preview">
  <before>ax="<xsl:value-of select='$ax'/>"</before>
</xsl:template>

<xsl:template name="doNothing">
  <xsl:param name="dummy"/>
</xsl:template>

</xsl:stylesheet>