<?xml version="1.0"?> 

<!-- non-conflicting use-acccumulator attribute between two xsl:mode with the same precedence -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/acc-ns/pppp"
  xmlns:q="http://www.w3.org/acc-ns/qqqq"
  exclude-result-prefixes="xs">

  <xsl:include href="mode-1514a.xsl"/>
  
  <xsl:accumulator name="p:a" initial-value="0" as="xs:integer">
    <xsl:accumulator-rule match="*" select="$value+5"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="q:b" initial-value="0" as="xs:integer">
    <xsl:accumulator-rule match="*" select="$value+6"/>
  </xsl:accumulator>
  
  <xsl:mode name="m" use-accumulators="p:a q:b"/>
  
  <xsl:variable name="x" as="element()">
    <x/>
  </xsl:variable>
    
  <xsl:template name="main">
    <out>
      <xsl:value-of select="(//*)[last()]/accumulator-after('q:b')"/>
    </out>
  </xsl:template>
  

  
    
</xsl:stylesheet>