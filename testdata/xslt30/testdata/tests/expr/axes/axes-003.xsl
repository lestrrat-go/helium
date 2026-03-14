<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <!-- PURPOSE: Test for 'ancestor::' Axis Identifier with index. -->
  <xsl:template match="/">
        <out>
          <xsl:for-each select="//center">
            <xsl:apply-templates select="ancestor::*[3]"/>
          </xsl:for-each>
  	  </out>
  </xsl:template>
  
  <xsl:template match="*">
     <xsl:value-of select="name(.)"/>
  </xsl:template>
 
</xsl:stylesheet>