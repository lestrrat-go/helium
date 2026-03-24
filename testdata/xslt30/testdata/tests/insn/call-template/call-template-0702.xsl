<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test for recursion with xsl:call-template. -->
  <!-- As -0701 but with namespace prefixes in variable names -->

  <xsl:template match="/doc" xmlns:P="http://0702.com/" exclude-result-prefixes="P">
    <out>
      <xsl:variable name="ResultTreeFragTest" select="name(.)"/>
      <xsl:call-template name="ntmp1">
        <xsl:with-param name="P:pvar1" select="$ResultTreeFragTest"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="ntmp1" xmlns:Q="http://0702.com/" exclude-result-prefixes="Q">
    <xsl:param name="Q:pvar1">def-text-1</xsl:param>
    <xsl:param name="Q:pvar2">def-text-2</xsl:param>
    <xsl:value-of select="$Q:pvar1"/>
    <xsl:text>,</xsl:text>
    <xsl:value-of select="$Q:pvar2"/>
    <xsl:text>,</xsl:text>
    <xsl:apply-templates select="*">
      <xsl:with-param name="Q:pvar1" select="$Q:pvar1"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="*" xmlns:R="http://0702.com/" exclude-result-prefixes="R">
    <xsl:param name="R:pvar1">def-text-1</xsl:param>
    <xsl:call-template name="ntmp1">
      <xsl:with-param name="R:pvar1" select="$R:pvar1"/>
      <xsl:with-param name="R:pvar2" select="name(.)"/>
    </xsl:call-template>
  </xsl:template>

</xsl:stylesheet>
