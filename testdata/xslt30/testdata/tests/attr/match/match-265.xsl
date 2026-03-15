<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs"
  expand-text="yes">
   
 
   <xsl:variable name="nodes" as="element()*">
     <x>
       <a>23</a>
       <b>25</b>
     </x>
   </xsl:variable>
  
  
   <xsl:template name="xsl:initial-template">
        <xsl:apply-templates select="$nodes"/>    
   </xsl:template>
  
   <xsl:template match="x">
      <X><xsl:apply-templates/></X>
   </xsl:template>
  
  <xsl:template match="x/(a|b)">
    <AB><xsl:value-of select="."/></AB>
  </xsl:template>
  
  
</xsl:transform>
