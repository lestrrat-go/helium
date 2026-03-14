<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:strip-space elements="*"/>
  
  <xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>
  
  <xsl:template match="Family">
    <xsl:for-each select=".//Child">
      <a name="{Name/@First}">
        <xsl:number value="position()" format="1.1"/>
      </a>
    </xsl:for-each>
  </xsl:template>
</xsl:stylesheet>
