<?xml version="1.0"?> 

<!-- conflicting use-acccumulator attribute between two xsl:mode with the same precedence -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:include href="mode-1510a.xsl"/>
  
  <xsl:mode name="m" use-accumulators="a"/>
  
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