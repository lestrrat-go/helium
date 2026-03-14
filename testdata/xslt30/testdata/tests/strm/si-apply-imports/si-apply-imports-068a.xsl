<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
       
  
  <!-- within a streaming template, use xsl:apply-imports with tunnel and non-tunnel parameters -->
   
  <xsl:mode streamable="yes"/>

  
  <xsl:template match="bktshort">
    <xsl:param name="p" tunnel="yes" required="yes"/>
    <xsl:param name="q" tunnel="yes" required="yes"/>
    <xsl:param name="r" tunnel="no" required="yes"/>
    <prio4 p="{$p}" q="{$q}" r="{$r}">
      <xsl:value-of select="."/>
    </prio4>
  </xsl:template>     
  

    
</xsl:transform>

