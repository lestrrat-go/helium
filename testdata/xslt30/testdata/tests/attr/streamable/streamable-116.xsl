<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <xsl:output indent="yes"/>
  <xsl:strip-space elements="chapter"/>

  <!-- disallow context-dependent global variables if principal document is streamed -->
  
  <xsl:mode name="s" streamable="yes"/>
   
  <xsl:template match="*" mode="s">
    <xsl:copy>
      <xsl:attribute name="count" select="$size"/>
      <xsl:apply-templates mode="s"/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="/" mode="s">
    <xsl:copy>
      <xsl:apply-templates mode="s"/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="v" mode="s"/>
  
  <xsl:variable name="size" select="count(//*)"/>
  
</xsl:stylesheet>