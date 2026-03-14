<?xml version="1.0"?> 

<!-- mode-0016 -->
<!-- Michael Kay -->
<!-- on-no-match=various, elements having namespace nodes --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  exclude-result-prefixes="xs xsi">

  <xsl:mode name="c" on-no-match="shallow-copy"/> 
  <xsl:mode name="d" on-no-match="shallow-skip"/>
  <xsl:mode name="s" on-no-match="text-only-copy"/>
  
  <xsl:variable name="temp" as="element()">
    <x:foo bar="42" xsi:type="xs:integer" xmlns:x="http://ns.x/" xmlns:y="http://ns.y/">3</x:foo>
  </xsl:variable>
  
  <xsl:template name="main">
    <out>
      <c><xsl:apply-templates select="$temp" mode="c"/></c>
      <d><xsl:apply-templates select="$temp" mode="d"/></d>
      <s><xsl:apply-templates select="$temp" mode="s"/></s>
    </out>
  </xsl:template>
  
  <xsl:template match="@*" mode="#all">
    <matched attribute="{local-name()}"/>
  </xsl:template>
  
    
</xsl:stylesheet>