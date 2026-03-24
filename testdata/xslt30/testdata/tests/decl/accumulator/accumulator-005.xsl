<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no">

  <!-- Get post-descent value of accumulator -->
  
  <xsl:param name="streamable" static="yes" select="'no'"/>


  <xsl:accumulator name="f:figNr" as="xs:decimal" initial-value="0"
    _streamable="{$streamable}">
    <xsl:accumulator-rule match="chap" select="0"/>
    <xsl:accumulator-rule match="fig" select="$value + 1"/>
    <xsl:accumulator-rule match="diag" select="$value + 0.5"/>
  </xsl:accumulator>

  <xsl:mode _streamable="{$streamable}" on-no-match="shallow-skip" use-accumulators="f:figNr"/>
  <xsl:template match="fig">
    <pix>
      <p>Figure <xsl:value-of select="accumulator-before('f:figNr')"/> start</p>
      <xsl:apply-templates/>
      <p>Figure <xsl:value-of select="accumulator-after('f:figNr')"/> end</p>
    </pix>
  </xsl:template>


  <xsl:template match="/">
    <figures>
      <xsl:apply-templates/>
    </figures>
  </xsl:template>
</xsl:package>
