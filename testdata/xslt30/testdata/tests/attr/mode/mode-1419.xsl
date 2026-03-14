<?xml version="1.0"?> 

<!-- mode-onnomatch008 -->
<!-- Michael Kay -->
<!-- on-no-match="shallow-skip" --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="s" on-no-match="shallow-skip"/> 
  
  <xsl:template match="/*" mode="s">
    <out>
      <xsl:next-match/>
    </out>
  </xsl:template>
  
  <xsl:template match="bktshort/text()" mode="s">
    <xsl:value-of select="."/>
  </xsl:template>
  

    
</xsl:stylesheet>