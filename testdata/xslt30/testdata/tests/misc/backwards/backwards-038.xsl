<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">
  
  <!-- PURPOSE: effect of BC on keys -->
  
  <xsl:key name="k" match="item" use="."/>
  
  <xsl:template match="/">
    <out>
      <xsl:for-each select="key('k', 1.0)">
        <k><xsl:copy-of select="."/></k>
      </xsl:for-each>
    </out>
  </xsl:template>
  
</xsl:stylesheet>
