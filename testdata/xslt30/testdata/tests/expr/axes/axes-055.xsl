<?xml version="1.0"?> 

  <!-- PURPOSE: Test elimination of duplicates from result of path expression. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:template match="/">
    <out>
       <xsl:for-each select="//b/..">!</xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
