<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:strip-space elements="chapter"/>

  <!-- copy the verses, lose the rest of the markup. -->
  
  <xsl:mode name="s" streamable="yes"/>
  
  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="true" href="ot.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>
   
  <xsl:template match="v" mode="s">
    <xsl:copy>
      <xsl:value-of select="."/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="text()" mode="s"/>
  
</xsl:stylesheet>