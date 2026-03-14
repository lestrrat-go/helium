<?xml version="1.0"?> 

<!-- Test xsl:mode on-no-match with namespace nodes; streaming= " yes ", "true", and "1" -->
<!-- on-no-match=various, namespace nodes, with streaming --> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:mode name="c" on-no-match="shallow-copy" streamable="true"/> 
  <xsl:mode name="d" on-no-match="shallow-skip" streamable="1"/>
  <xsl:mode name="s" on-no-match="text-only-copy" streamable=" yes "/>
  
  <xsl:variable name="temp" as="namespace-node()">
    <xsl:namespace name="a">http://abracadabra.com/</xsl:namespace>
  </xsl:variable>
  
  <xsl:template name="main">
    <out>
      <c><xsl:apply-templates select="$temp" mode="c"/></c>
      <d><xsl:apply-templates select="$temp" mode="d"/></d>
      <s><xsl:apply-templates select="$temp" mode="s"/></s>
    </out>
  </xsl:template>
  
    
</xsl:stylesheet>