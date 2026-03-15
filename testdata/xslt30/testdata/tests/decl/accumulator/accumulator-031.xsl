<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs" version="3.0"  declared-modes="no">

  <!-- Accumulator has circular dependency on global variable -->
  
  <xsl:mode streamable="yes"/>
  
  <xsl:global-context-item use="absent"/>
  
  <xsl:accumulator name="a" as="xs:integer" initial-value="0" streamable="yes">
    <xsl:accumulator-rule match="*" select="$value + $count"/>
  </xsl:accumulator>

  <xsl:variable name="count" as="xs:integer">
    <xsl:source-document streamable="true" href="accumulator-001.xml" use-accumulators="#all">
      <xsl:for-each select="*"/>
      <xsl:value-of select="accumulator-after('a')"/>
    </xsl:source-document>
  </xsl:variable>
  
  <xsl:template name="main" visibility="public">
    <o><xsl:value-of select="$count"/></o>
  </xsl:template>
</xsl:package>
