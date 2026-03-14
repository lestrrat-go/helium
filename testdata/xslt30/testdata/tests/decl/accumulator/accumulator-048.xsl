<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
  name="http://www.w3.org/xslt30-test/accumulator/accumulator-001"
  package-version="1.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no">

  <!-- Accumulator values are copied by xsl:copy-of with copy-accumulators="yes" -->
  
  <xsl:param name="streamable" static="yes" select="'no'"/>
  <xsl:param name="copyAcc" static="yes" select="'yes'"/>

  <xsl:accumulator name="figNr" as="xs:integer" initial-value="0" _streamable="{$streamable}">
    <xsl:accumulator-rule match="fig" select="$value + 1"/>
  </xsl:accumulator>

  <xsl:mode _streamable="{$streamable}" on-no-match="shallow-skip" use-accumulators="figNr"/>
  
  <xsl:template match="fig">
    <xsl:apply-templates/>
    <p>Figure <xsl:value-of select="accumulator-before('figNr')"/>: <xsl:value-of select="@alt"/></p>
  </xsl:template>

  <xsl:template match="/">
    <figures>
      <xsl:variable name="second-chap" as="element(chap)">
        <xsl:copy-of select="/doc/chap[2]" _copy-accumulators="{$copyAcc}"/>
      </xsl:variable>  
      <xsl:apply-templates select="$second-chap"/>
    </figures>
  </xsl:template>
</xsl:package>
