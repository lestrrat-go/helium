<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="3.0">

  <xsl:accumulator name="section-nr" as="xs:integer*" initial-value="0">
    <xsl:accumulator-rule match="section" phase="start" 
      select="0, head($value)+1, tail($value)"/>
    <xsl:accumulator-rule match="section" phase="end" select="tail($value) (:pop:)"/>
  </xsl:accumulator>
  
  <xsl:mode use-accumulators="section-nr"/>

  <xsl:template match="node()|@*">
    <xsl:copy>
      <xsl:apply-templates select="node()|@*"/>
    </xsl:copy>
  </xsl:template>

  <xsl:template match="section">
    <p>
      <xsl:value-of select="reverse(tail(accumulator-before('section-nr')))" 
        separator="."/>
    </p>
    <xsl:apply-templates/>
  </xsl:template>

</xsl:stylesheet>