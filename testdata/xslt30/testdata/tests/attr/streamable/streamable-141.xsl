<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:strip-space elements="chapter"/>

  <!-- §3.9.1: an instruction processed in 1.0 compatibility mode is roaming and free-ranging -->
  
  <xsl:mode name="s" streamable="yes"/>
  
  <xsl:template name="xsl:initial-template">
      <xsl:source-document streamable="true" href="ot.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
  </xsl:template>
  
  <xsl:template match="*" mode="s">
    <xsl:copy>
      <xsl:apply-templates mode="s" version="1.0"/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="/" mode="s">
    <xsl:copy>
      <xsl:apply-templates mode="s"/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="v" mode="s"/>
  
</xsl:stylesheet>