<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs" version="3.0"  declared-modes="no">

  <!-- Local variables in initial-value and select expressions -->
  
  <xsl:mode streamable="yes" on-no-match="shallow-skip" use-accumulators="a"/>
  
  <xsl:accumulator name="a" as="xs:integer" initial-value="min(for $i in $zero to 5 return $i*2)"
  			streamable="yes">
    <xsl:accumulator-rule match="chap" select="0"/>
    <xsl:accumulator-rule match="fig" select="$value + min(for $i in ($zero+1) to 5 return $i*2)"/>
  </xsl:accumulator>
  
  <xsl:template match="fig">
    <xsl:apply-templates/>
    <p>Figure <xsl:value-of select="accumulator-before('a')"/>: <xsl:value-of select="@alt"/></p>
  </xsl:template>

  <xsl:template match="/">
    <figures>
      <xsl:apply-templates/>
    </figures>
  </xsl:template>
  
  <xsl:param name="zero" select="0"/>
  <xsl:global-context-item/>
</xsl:package>
