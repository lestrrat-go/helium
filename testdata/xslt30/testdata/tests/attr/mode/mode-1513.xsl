<?xml version="1.0"?> 

<!-- non-conflicting use-accumulator attribute between two xsl:mode with differing precedence -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:import href="mode-1510.xsl"/>
  
  <xsl:mode name="m" use-accumulators="b a c"/>
  
  <xsl:accumulator name="c" initial-value="0" as="xs:integer">
    <xsl:accumulator-rule match="*" select="$value+3"/>
  </xsl:accumulator>
  
  <xsl:variable name="x" as="element()">
    <x/>
  </xsl:variable>
    
  <xsl:template name="main">
    <out>
      <xsl:value-of select="(//*)[last()]/accumulator-after('b')"/>
    </out>
  </xsl:template>
  
 

  
    
</xsl:stylesheet>