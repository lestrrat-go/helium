<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
       
  
    <!-- within a streaming template, simple use of xsl:sequence -->
    
     
    <xsl:mode name="s" streamable="yes"/>
    <xsl:strip-space elements="*"/>
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
      
    <xsl:template name="main" match="/">
      <out>
        <xsl:source-document streamable="true" href="loans.xml"><xsl:apply-templates select="." mode="s"/></xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="Property" mode="s">
      <Property>
        <xsl:sequence select="Address"/>
      </Property>
    </xsl:template>
    
    <xsl:template match="text()" mode="#all"/>
         
</xsl:transform>

