<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- Test for recursion with xsl:call-template. -->
  <!-- As -0701 but with EQNames as variable names -->

  <xsl:template match="/doc">
    <out>
      <xsl:variable name="ResultTreeFragTest" select="name(.)"/>
      <xsl:call-template name="ntmp1">
        <xsl:with-param name="Q{http://0702.com/}pvar1" select="$ResultTreeFragTest"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="ntmp1" xmlns:Q="http://0702.com/" exclude-result-prefixes="Q">
    <xsl:param name="Q{http://0702.com/}pvar1">def-text-1</xsl:param>
    <xsl:param name="Q{}pvar2">def-text-2</xsl:param>
    <xsl:value-of select="$Q:pvar1" xmlns:Q="http://0702.com/"/>
    <xsl:text>,</xsl:text>
    <xsl:value-of select="$pvar2"/>
    <xsl:text>,</xsl:text>
    <xsl:apply-templates select="*">
      <xsl:with-param name="Q{http://0702.com/}pvar1" select="$Q{http://0702.com/}pvar1"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="*" xmlns:R="http://0702.com/" exclude-result-prefixes="R">
    <xsl:param name="R:pvar1" xmlns:R="http://0702.com/">def-text-1</xsl:param>
    <xsl:call-template name="ntmp1">
      <xsl:with-param name="S:pvar1" select="$S:pvar1"  xmlns:S="http://0702.com/"/>
      <xsl:with-param name="pvar2" select="name(.)"/>
    </xsl:call-template>
  </xsl:template>

</xsl:stylesheet>
