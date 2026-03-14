<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
       
  
    <!-- within a streaming template, apply-templates with tunnel parameters -->
     
    <xsl:mode streamable="yes"/>
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
  
    <xsl:strip-space elements="*"/>
      
    <xsl:template name="main">
      <out>
       <xsl:source-document streamable="true" href="mixed.xml">
         <xsl:apply-templates select=".">
           <xsl:with-param name="p" select="17" tunnel="yes"/> 
         </xsl:apply-templates>
       </xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="/">
        <xsl:apply-templates>
          <xsl:with-param name="q" select="23" tunnel="yes"/> 
        </xsl:apply-templates>
    </xsl:template>  
    
    
    <xsl:template match="text()"/>
     
    <xsl:template match="v">
      <xsl:param name="p" tunnel="yes" required="yes"/>
      <xsl:param name="q" tunnel="yes" required="yes"/>
      <v p="{$p}" q="{$q}"><xsl:value-of select="."/></v>
    </xsl:template>
   
    
</xsl:transform>

