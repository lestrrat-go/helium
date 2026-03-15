<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test for sorted results with combination of axes -->
  <xsl:template match="/">
    <out>
      <ordered>
          <xsl:for-each select="//e">
            <xsl:apply-templates select="following-sibling::*/../child::*"/>
          </xsl:for-each>
      </ordered>
  	</out>
  </xsl:template>
    
  <xsl:template match="*">
     <xsl:value-of select="name(.)"/><xsl:text> </xsl:text>
  </xsl:template>
 
</xsl:stylesheet>
