<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no">

  <!-- Dynamic call on accumulator-before - not streamed -->
  <!-- Context item for accumulator-before is bound within the function item -->
 

  <xsl:variable name="data">
    <doc><a/><b/><c/></doc>
  </xsl:variable>

  <xsl:accumulator name="f:elementNr" as="xs:integer" initial-value="0"
       streamable="false">
    <xsl:accumulator-rule match="*" select="$value + 1"/>
  </xsl:accumulator>

  <xsl:template match="*">
    <xsl:param name="aa" select="../accumulator-before#1"/>
    <xsl:copy>
      <xsl:attribute name="aa" select="$aa('f:elementNr')"/>
      <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template name="xsl:initial-template" visibility="public">
    <xsl:apply-templates select="$data"/>
  </xsl:template>

</xsl:package>
