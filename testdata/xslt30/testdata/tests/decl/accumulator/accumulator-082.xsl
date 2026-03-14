<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  expand-text="yes">
  
  <xsl:accumulator name="iterate-value-test" as="xs:integer" initial-value="0">
    <xsl:accumulator-rule match="foo">
      <xsl:iterate select="1">
        <xsl:param name="test" select="$value+1"/>
        <xsl:on-completion select="$test"/>
      </xsl:iterate>
    </xsl:accumulator-rule>
  </xsl:accumulator>
  
  <xsl:mode on-no-match="shallow-copy" use-accumulators="iterate-value-test"/>
  
  <xsl:template match="foo">
    <xsl:copy>
      <xsl:attribute name="accumulator-test" select="accumulator-before('iterate-value-test')"/>
    </xsl:copy>
  </xsl:template>
  

  
</xsl:stylesheet>