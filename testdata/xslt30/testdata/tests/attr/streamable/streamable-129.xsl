<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, bind a variable to a streamed node and then
       do illicit navigation starting from the variable -->
    
   
  <xsl:mode name="s" streamable="yes"/>
  <xsl:strip-space elements="*"/>
        
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:apply-templates select="document('transactions.xml')" mode="s"/>
    </out>
  </xsl:template>
  
  <xsl:template match="account" mode="s">
      <DeeplyInTheRed><xsl:value-of select="some $t in transaction satisfies xs:decimal($t/preceding-sibling::*[1]/@value) lt 0.0"/></DeeplyInTheRed>
  </xsl:template>
  
       
</xsl:transform>

