<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum005/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no">

  <!-- Use accumulators to compute hierarchic section numbers -->
  
  <xsl:param name="streamable" static="yes" select="'no'"/>

  <xsl:accumulator name="f:section-nr" as="xs:integer*" initial-value="0" _streamable="{$streamable}">
    <xsl:accumulator-rule match="section" phase="start" select="0, head($value)+1, tail($value)"/>
    <xsl:accumulator-rule match="section" phase="end" select="tail($value) (:pop:)"/>
  </xsl:accumulator>


  <xsl:mode _streamable="{$streamable}" on-no-match="shallow-copy" use-accumulators="#all"/>

  <xsl:template match="section">
    <span>
      <xsl:value-of select="reverse(tail(accumulator-before('f:section-nr')))" separator="."/>
    </span>
    <xsl:apply-templates/>
  </xsl:template>

  <xsl:template match="head">
    <span>
      <xsl:value-of select="."/>
    </span>
  </xsl:template>

  <xsl:strip-space elements="*"/>

</xsl:package>
