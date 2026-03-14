<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, use xsl:for-each that mixes element and text nodes -->
    
   
  <xsl:mode name="s" streamable="yes"/>
  <xsl:strip-space elements="*"/>
        
  <xsl:output method="xml" indent="no" encoding="UTF-8" />
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:apply-templates select="document('mixed.xml')" mode="s"/>
    </out>
  </xsl:template>
  
  <xsl:template match="v[../@nr='1']" mode="s">
      <v>
        <xsl:for-each select="node()">
            <node><xsl:value-of select="."/></node>
        </xsl:for-each>
      </v>
  </xsl:template>
  
  <xsl:template match="v" mode="s"/>
  
  <xsl:template match="text()" mode="s"/>
  
       
</xsl:transform>

