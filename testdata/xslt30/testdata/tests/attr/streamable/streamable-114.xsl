<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, evaluate a FLWOR expression drilling down through the streamed input -->
  
  <!-- Not guaranteed streamable according to W3C rules -->
    
   
  <xsl:mode name="s" streamable="yes"/>
  <xsl:strip-space elements="*"/>
        
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="xmark1.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="item" mode="s">
      <item id="{@id}">
        <xsl:sequence select="for $x in .//parlist
                              return $x/listitem/text/tokenize(., '\s+')[string-length(.) gt 8]"/>
      </item>
  </xsl:template>
  
  <xsl:template match="text()"/>
  
       
</xsl:transform>

