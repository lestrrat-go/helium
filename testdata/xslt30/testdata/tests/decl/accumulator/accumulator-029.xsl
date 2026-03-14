<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs" version="3.0"  declared-modes="no">

  <!-- Accumulator declared streamable, but accumulator-rule/@match is not motionless -->
  
  <xsl:mode streamable="yes"/>
  
  <xsl:accumulator name="a" as="xs:integer" initial-value="0"  streamable="yes">
    <xsl:accumulator-rule match="chap" select="0"/>
    <xsl:accumulator-rule match="fig[caption]" select="$value + 2"/>
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
</xsl:package>
