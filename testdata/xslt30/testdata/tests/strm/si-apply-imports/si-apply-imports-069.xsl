<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
    
  <xsl:import href="si-apply-imports-069a.xsl"/>  
       
  
  <!-- within a streaming template, use xsl:apply-imports with tunnel and non-tunnel parameters -->
  
  <xsl:mode streamable="yes"/>
         
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />

  <xsl:strip-space elements="*"/>
    
  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/ot.xml">
        <xsl:apply-templates select=".">
          <xsl:with-param name="p" select="17" tunnel="yes"/> 
        </xsl:apply-templates>
      </xsl:source-document>  
    </out>
  </xsl:template>
  
  <xsl:template match="/">
      <xsl:apply-templates select="outermost(.//book)">
        <xsl:with-param name="q" select="23" tunnel="yes"/> 
      </xsl:apply-templates>
  </xsl:template>
  
  <xsl:template match="bookcoll | tstmt | book">
    <xsl:apply-templates select="outermost(.//bktshort)"/>
  </xsl:template>
  
  <xsl:template match="bktshort">
    <xsl:param name="p" tunnel="yes" required="yes"/>
    <xsl:param name="q" tunnel="yes" required="yes"/>
    <prio5 p="{$p}" q="{$q}">
      <xsl:apply-imports>
         <xsl:with-param name="p" select="$p+1" tunnel="yes"/> 
         <xsl:with-param name="r" select="42"/>
      </xsl:apply-imports>
    </prio5>
  </xsl:template>   
  

  
  <xsl:template match="node()"/>
   
 
    
</xsl:transform>

