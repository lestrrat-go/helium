<?xml version="1.0"?> 

<!-- non-conflicting use-acccumulator attribute between two xsl:mode with the same precedence -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:pppp="http://www.w3.org/acc-ns/p"
  xmlns:qqqq="http://www.w3.org/acc-ns/q"
  exclude-result-prefixes="xs">

  <xsl:include href="mode-1514a.xsl"/>
  
  <xsl:mode name="m" use-accumulators="qqqq:b pppp:a"/>
  
  <xsl:variable name="x" as="element()">
    <x/>
  </xsl:variable>
    
  <xsl:template name="main">
    <out>
      <xsl:value-of select="(//*)[last()]/accumulator-after('qqqq:b')"/>
    </out>
  </xsl:template>
  

  
    
</xsl:stylesheet>