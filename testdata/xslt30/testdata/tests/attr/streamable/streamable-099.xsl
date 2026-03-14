<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, evaluate a general comparison on streamed nodes -->
    
   
  <xsl:mode name="s" streamable="yes"/>
  <xsl:strip-space elements="*"/>
  
  <xsl:param name="p1" select="-1000"/>
  <xsl:param name="p2" select="-1100"/>
  <xsl:param name="p3" select="-1200"/>
       
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:apply-templates select="document('transactions.xml')" mode="s"/>
    </out>
  </xsl:template>
  
  <xsl:template match="account" mode="s">
      <DeeplyInTheRed><xsl:value-of select="transaction/@value/xs:decimal(.) &lt;= ($p1, $p2, $p3)"/></DeeplyInTheRed>
  </xsl:template>
  
  <xsl:template match="text()" mode="#all"/>
       
</xsl:transform>

