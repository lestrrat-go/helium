<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
   <!-- within a streaming template, evaluate a quantified expression on streamed nodes -->
  
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
       <SlightlyInTheRed><xsl:value-of select="some $t in transaction satisfies xs:decimal($t/@value) lt -200.0"/></SlightlyInTheRed>
   </xsl:template>
  
       
</xsl:transform>

