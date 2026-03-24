<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no">

  <!-- Dynamic call on accumulator-after - not streamable! -->
 

  <xsl:param name="aa" select="accumulator-after#1"/>

  <xsl:accumulator name="f:figNr" as="xs:decimal" initial-value="0"
       streamable="true">
    <xsl:accumulator-rule match="chap" select="0"/>
    <xsl:accumulator-rule match="fig" select="$value + 1"/>
    <xsl:accumulator-rule match="diag" select="$value + 0.5"/>
  </xsl:accumulator>

  <xsl:mode streamable="true" on-no-match="shallow-skip" use-accumulators="f:figNr"/>
  <xsl:template match="fig">
    <pix>
      <p>Figure <xsl:value-of select="accumulator-before('f:figNr')"/> start</p>
      <xsl:apply-templates/>
      <p>Figure <xsl:value-of select="../$aa('f:figNr')"/> end</p>
    </pix>
  </xsl:template>


  <xsl:template match="/">
    <figures>
      <xsl:apply-templates/>
    </figures>
  </xsl:template>
</xsl:package>
