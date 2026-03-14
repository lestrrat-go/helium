<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#with-param?>
  <!-- Purpose: Test how big a string can be passed to a template. -->
  

<!-- Set upper limit here -->
<xsl:variable name="max" select="256" />

<xsl:template match="/doc">
  <out>
    <xsl:call-template name="looper">
      <xsl:with-param name="str" select="'....5....|'" />
    </xsl:call-template>
  </out>
</xsl:template>

<xsl:template name="looper">
  <xsl:param name="str" select="0000000000" />
  <!-- put out one iteration of the name and a trailing space -->
  <xsl:value-of select="string-length($str)"/><xsl:text>: </xsl:text>
  <xsl:value-of select="$str"/><xsl:text>
</xsl:text>
  <!-- here's the recursion -->
  <xsl:if test="string-length($str) &lt; $max">
    <xsl:call-template name="looper">
      <xsl:with-param name="str">
        <xsl:value-of select="concat($str,'....5....|')"/>
      </xsl:with-param>
    </xsl:call-template>
  </xsl:if>
</xsl:template>

</xsl:stylesheet>
