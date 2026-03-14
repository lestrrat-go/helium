<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no">

  <!-- Simple test of an accumulator function with streamable="No" (not allowed) -->
  
  <xsl:param name="streamable" static="yes" select="'No'"/>

  <xsl:accumulator name="figNr" as="xs:integer" initial-value="0" _streamable="{$streamable}">
    <xsl:accumulator-rule match="chap" select="0"/>
    <xsl:accumulator-rule match="fig" select="$value + 1"/>
  </xsl:accumulator>

  <xsl:mode _streamable="{$streamable}" on-no-match="shallow-skip"/>
  
  <xsl:template match="fig">
    <xsl:apply-templates/>
    <p>Figure <xsl:value-of select="accumulator-before('figNr')"/>: <xsl:value-of select="@alt"/></p>
  </xsl:template>

  <xsl:template match="/">
    <figures>
      <xsl:apply-templates/>
    </figures>
  </xsl:template>
</xsl:package>
