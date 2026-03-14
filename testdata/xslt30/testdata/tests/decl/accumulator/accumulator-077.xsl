<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  expand-text="yes"
  exclude-result-prefixes="#all">
  
  <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
  
  <xsl:mode _streamable="{$STREAMABLE}" use-accumulators="#all"/>
  
  <xsl:strip-space elements="*"/>
  
  <xsl:accumulator name="header-map" _streamable="{$STREAMABLE}" as="map(xs:integer, xs:string)" initial-value="map{}">
    <xsl:accumulator-rule match="header-item"
      phase="end"
      select="map:put($value, accumulator-after('header-id'), accumulator-after('header-value'))"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="header-id" _streamable="{$STREAMABLE}" as="xs:integer?" initial-value="()">
    <xsl:accumulator-rule match="header-item/id/text()" select="xs:integer(.)"/>
  </xsl:accumulator>
  
  <xsl:accumulator name="header-value" _streamable="{$STREAMABLE}" as="xs:string?" initial-value="()">
    <xsl:accumulator-rule match="header-item/value/text()" select="string()"/>
  </xsl:accumulator>
  
  
  
  <xsl:template match="header"/>
  
  <xsl:template match="item/header-ref">
    <header-value>{accumulator-before('header-map')(xs:integer(@idref))}</header-value>
  </xsl:template>
  
  <xsl:template match="id"/>
  
</xsl:stylesheet>