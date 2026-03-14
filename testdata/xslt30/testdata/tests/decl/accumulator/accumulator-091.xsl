<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  expand-text="yes">
  
  <xsl:param name="limit" as="xs:integer" select="5"/>
  
  <xsl:accumulator name="node-count" as="xs:integer" initial-value="0">
    <xsl:accumulator-rule match="node()[$value le $limit or $value gt $limit]" select="$value + 1"/>
  </xsl:accumulator>
  
  <xsl:mode on-no-match="shallow-copy" use-accumulators="#all"/>
  
  <xsl:template match="node()">
    <xsl:message select="accumulator-before('node-count')"/>
    <xsl:next-match/>
  </xsl:template>
  
  <xsl:template match="/" name="xsl:initial-template">
    <xsl:copy>
      <xsl:apply-templates/>
      <xsl:comment>Run with {system-property('xsl:product-name')} {system-property('xsl:product-version')} at {current-dateTime()}</xsl:comment>
    </xsl:copy>
  </xsl:template>
  
</xsl:stylesheet>