<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs" version="3.0"  declared-modes="no">

  <!-- Accumulator pattern has dependency on global variable and local variable -->
  
  <xsl:param name="streamable" static="yes" required="yes"/>
  
  <xsl:mode _streamable="{$streamable}" on-no-match="shallow-skip" use-accumulators="#all"/>
  
  <xsl:global-context-item/>
  
  <xsl:accumulator name="a" as="xs:integer" initial-value="0" _streamable="{$streamable}">
    <xsl:accumulator-rule match="chap[not(@nr = $seven)]" select="0"/>
    <xsl:accumulator-rule match="fig[every $n in data(@*) satisfies $n = '83']" select="$value + 2"/>
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
  
  <xsl:variable name="seven" select="7"/>
</xsl:package>
