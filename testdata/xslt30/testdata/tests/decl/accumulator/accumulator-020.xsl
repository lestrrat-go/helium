<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:f="http://accum001/"
  xmlns:f2="http://accum001/"
  xmlns:g="http://accum002/"
    declared-modes="no"
  exclude-result-prefixes="xs f f2 g" version="3.0">

  <!-- Test valid/invalid accumulator names -->
  
  <xsl:param name="one" static="yes" select="'one'"/>
  <xsl:param name="two" static="yes" select="'two'"/>

  <xsl:accumulator _name="{$one}" as="xs:integer" initial-value="0">
    <xsl:accumulator-rule match="chap" select="0"/>
    <xsl:accumulator-rule match="fig" select="$value + 1"/>
  </xsl:accumulator>
  
  <xsl:accumulator _name="{$two}" as="xs:integer" initial-value="0">
    <xsl:accumulator-rule match="chap" select="0"/>
    <xsl:accumulator-rule match="fig" select="$value + 1"/>
  </xsl:accumulator>

  <xsl:mode on-no-match="shallow-skip" _use-accumulators="{$one} {$two}"/>
  
  <xsl:template match="fig">
    <xsl:apply-templates/>
    <p>Figure <xsl:value-of select="accumulator-before($one)"/>: <xsl:value-of select="@alt"/></p>
  </xsl:template>

  <xsl:template match="/">
    <figures>
      <xsl:apply-templates/>
    </figures>
  </xsl:template>
</xsl:package>
