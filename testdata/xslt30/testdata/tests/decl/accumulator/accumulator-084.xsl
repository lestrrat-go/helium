<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  expand-text="yes">
  
  <xsl:accumulator name="iterate-value-test" as="xs:integer" initial-value="0">
    <xsl:accumulator-rule match="*:foo">
      <xsl:sequence select="let $temp := namespace-uri() return if ($temp = '') then -1 else string-length($temp)"/>
    </xsl:accumulator-rule>
  </xsl:accumulator>
  
  <xsl:mode on-no-match="shallow-copy" use-accumulators="iterate-value-test"/>
  
  <xsl:template match="foo">
    <xsl:copy>
      <xsl:attribute name="accumulator-test" select="accumulator-before('iterate-value-test')"/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="/" name="xsl:initial-template">
    <xsl:next-match/>
    <xsl:comment xmlns:saxon="http://saxon.sf.net/">Run with {system-property('xsl:product-name')} {system-property('xsl:product-version')} {system-property('Q{http://saxon.sf.net/}platform')}</xsl:comment>
  </xsl:template>
  
</xsl:stylesheet>