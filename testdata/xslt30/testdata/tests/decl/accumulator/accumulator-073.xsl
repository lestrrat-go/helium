<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://accum001/"
  exclude-result-prefixes="xs f" version="3.0"  declared-modes="no" expand-text="yes">

  <!-- Test xsl:copy-of using copy-accumulators=true and validation=strict -->
 
  <xsl:import-schema schema-location="books.xsd"/>

  <xsl:accumulator name="f:total-price" as="xs:decimal" initial-value="0.0"
       streamable="false">
    <xsl:accumulator-rule match="PRICE" select="$value + ."/>
  </xsl:accumulator>
  
  <xsl:mode on-no-match="shallow-copy"/>
  
  <xsl:template match="BOOKLIST">
    <xsl:variable name="last3" as="element()*">
      <xsl:copy-of select="subsequence(BOOKS/ITEM, 4)" copy-accumulators="yes" validation="lax"/>
    </xsl:variable>
    <xsl:for-each select="$last3">
      <book title="{TITLE}" typed="{PRICE instance of element(*, moneyType)}">{accumulator-before('f:total-price')}</book>
    </xsl:for-each>
  </xsl:template>
  
  <xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

</xsl:package>
