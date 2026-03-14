<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
       
  
    <!-- within a streaming template, a more elaborate expression using the result of the streamed value -->
     
    <xsl:mode streamable="yes"/>
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
  
    <xsl:strip-space elements="*"/>
      
    <xsl:template name="main">
      <out>
        <xsl:source-document streamable="true" href="mixed.xml"><xsl:apply-templates select="."/></xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="book">
      <xsl:apply-templates select="chapter/chtitle"/>
    </xsl:template>
    
    <xsl:template match="chtitle">
      <title chapter="{concat('#', if (upper-case(.) = 'CHAPTER 1') then 'one' else 'two')}"/>
    </xsl:template>
    
      
</xsl:transform>

