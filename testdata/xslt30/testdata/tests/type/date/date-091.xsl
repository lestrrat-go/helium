<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec fo#casting-to-string?>
 <!-- PURPOSE: test casting xs:duration to string -->
  
  <xsl:template match="/">
     <out>
     <time>
      <a><xsl:value-of select="xs:duration('P24M')"/></a> 
      <a><xsl:value-of select="xs:duration('P0M')"/></a>     
      <a><xsl:value-of select="xs:duration('P2Y18M')"/></a> 
      <a><xsl:value-of select="xs:duration('P1M60D')"/></a>                  
      <a><xsl:value-of select="xs:duration('P60DT48H')"/></a>    
      <a><xsl:value-of select="xs:duration('P1Y1D')"/></a>    
      <a><xsl:value-of select="xs:duration('PT0M')"/></a>    
     </time>                                        
     </out>     
  </xsl:template>

</xsl:stylesheet>
