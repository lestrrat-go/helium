<?xml version="1.0"?> 

<!-- mode-1420 -->
<!-- Michael Kay -->
<!-- on-no-match="shallow-skip" with streaming --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="s" on-no-match="shallow-skip" streamable="yes"/> 
  
  <xsl:template match="/*" mode="s">
    <out>
      <xsl:next-match/>
    </out>
  </xsl:template>
  
  <xsl:template match="bktshort/text()" mode="s">
    <xsl:value-of select="."/>
  </xsl:template>
  
  <xsl:template match="@nr" mode="s">
    <nr><xsl:value-of select="."/></nr>
  </xsl:template>
  

    
</xsl:stylesheet>