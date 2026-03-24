<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math"
  version="3.0">
  
  <xsl:mode on-no-match="shallow-copy" streamable="yes" use-accumulators="w"/>

  
  <xsl:accumulator name="w" initial-value="0" streamable="true" as="xs:integer">
    <xsl:accumulator-rule match="text()" select="$value + count(tokenize(.))"/>
  </xsl:accumulator>
  
  <xsl:template match="/*">
    <xsl:copy>
      <xsl:apply-templates/>
      <result>Total count of words in document : <xsl:value-of select="accumulator-after('w')"/></result>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="section">
    <xsl:copy>
      <xsl:apply-templates select="@*"/>
      <xsl:apply-templates/>
      <p>(words: <xsl:value-of select="accumulator-after('w') - accumulator-before('w')"/>)</p>
    </xsl:copy>
  </xsl:template>
  
</xsl:stylesheet>