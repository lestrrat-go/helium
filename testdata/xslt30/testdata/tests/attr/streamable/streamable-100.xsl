<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
    <!-- within a streaming template, evaluate a "some" expression -->
  
    <!-- streamable in Saxon but not guaranteed-streamable according to W3C rules -->
      
     
    <xsl:mode name="s" streamable="yes"/>
    <xsl:strip-space elements="*"/>
          
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
      
    <xsl:template name="main" match="/">
      <out>
        <xsl:apply-templates select="document('transactions.xml')" mode="s"/>
      </out>
    </xsl:template>
    
    <xsl:template match="account" mode="s">
        <DeeplyInTheRed><xsl:value-of select="some $t in transaction satisfies xs:decimal($t/@value) lt -11110.0"/></DeeplyInTheRed>
    </xsl:template>
    
       
</xsl:transform>

