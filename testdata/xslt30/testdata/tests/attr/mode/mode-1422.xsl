<?xml version="1.0"?> 

<!-- mode-onnomatch009s -->
<!-- Michael Kay -->
<!-- on-no-match="shallow-skip" with streaming --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="s" on-no-match="shallow-copy" streamable="yes"/> 
  <xsl:mode name="t" on-no-match="shallow-skip" streamable="yes"/>
  
  <xsl:template match="/*" mode="s">
    <out>
      <xsl:next-match/>
    </out>
  </xsl:template>
  
  <xsl:template match="v" mode="s">
    <v><xsl:apply-templates mode="t"/></v>
  </xsl:template>
  

    
</xsl:stylesheet>