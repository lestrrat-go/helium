<?xml version="1.0"?> 

<!-- mode-onnomatch013s -->
<!-- Michael Kay -->
<!-- on-no-match=various, attribute nodes, with streaming --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="c" on-no-match="shallow-copy" streamable="yes"/> 
  <xsl:mode name="d" on-no-match="shallow-skip" streamable="yes"/>
  <xsl:mode name="s" on-no-match="text-only-copy" streamable="yes"/>
  
  <xsl:variable name="temp" as="attribute()">
    <xsl:attribute name="a">abracadabra</xsl:attribute>
  </xsl:variable>
  
  <xsl:template name="main">
    <out>
      <c><xsl:apply-templates select="$temp" mode="c"/></c>
      <d><xsl:apply-templates select="$temp" mode="d"/></d>
      <s><xsl:apply-templates select="$temp" mode="s"/></s>
    </out>
  </xsl:template>
  
    
</xsl:stylesheet>