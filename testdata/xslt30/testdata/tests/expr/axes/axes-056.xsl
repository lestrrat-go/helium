<?xml version="1.0"?> 

<!-- Test elimination of duplicates from result of path expression. -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:template match="b">
    <out>
       <attributes>
       <xsl:for-each select="preceding-sibling::c/@x"><xsl:value-of select="."/>,</xsl:for-each>
       </attributes>
       <elements>
       <xsl:for-each select="preceding-sibling::c/x"><xsl:value-of select="."/>,</xsl:for-each>
       </elements>
    </out>
  </xsl:template>
  
  <xsl:template match="c"/>
  
  <xsl:template match="text()"/>
  
</xsl:stylesheet>
