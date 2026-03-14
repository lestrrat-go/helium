<?xml version="1.0"?> 
  <!-- PURPOSE: Test ancestor axis with a specific name. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:template match="/">
    <out>
       <xsl:for-each select="//c1">
         <xsl:if test="ancestor::b">
            <xsl:value-of select="ancestor::b/@att"/>
         </xsl:if>
        </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
