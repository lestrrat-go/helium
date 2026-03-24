<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:my="my" 
    exclude-result-prefixes="xs my" 
    version="3.0">
    
  <!-- test that the priority of a pattern with a predicate is 0.5 -->  
    
  <xsl:variable name="x" as="element()">
    <test class="content_body"/>
  </xsl:variable>
  
  <xsl:template name="main">
    <xsl:apply-templates select="$x"/>
  </xsl:template>    

  <xsl:template match="*[contains(@class,'content_body')]" priority="0.501">
    <one><xsl:next-match/></one>
  </xsl:template>
  
  <xsl:template match="*[contains(@class,'content_body')]">
    <two><xsl:next-match/></two>
  </xsl:template> 

  <xsl:template match="*[contains(@class,'content_body')]" priority="0.499">
    <three/>
  </xsl:template> 
  
   

</xsl:stylesheet>