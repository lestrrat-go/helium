<?xml version="1.0"?> 

<!-- conflicting attributes between two xsl:mode with the same precedence, resolved by 
     another declaration at higher import precedence -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:import href="mode-1505a.xsl"/>
  
  <xsl:mode streamable="no" on-no-match="deep-copy"/>
  
  <xsl:variable name="x" as="element()">
    <x/>
  </xsl:variable>
    
  <xsl:template name="main">
    <out>
      <xsl:apply-templates select="$x"/>
    </out>
  </xsl:template>
  
  <xsl:template match="unstreamable">
    <xsl:value-of select="a+b+c"/>
  </xsl:template>
  

  
    
</xsl:stylesheet>