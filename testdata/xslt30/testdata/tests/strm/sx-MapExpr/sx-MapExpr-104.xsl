<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" 
  exclude-result-prefixes="xs math map" 
  version="3.0">
  
  <xsl:param name="STREAMABLE" as="xs:boolean" static="yes" select="true()"/>
  
  <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-copy"/>
  
  <xsl:output indent="yes"/>
  <xsl:strip-space elements="*"/>
  
  <xsl:template match="section">
    <xsl:copy>
      <xsl:for-each-group select="*" group-starting-with="h1">
        <xsl:variable name="head-and-tail" as="map(xs:integer, element()*)" 
          select="map { 1 : copy-of(current-group()[self::h1]), 2 : copy-of(current-group()[self::p]) }"/>
        <xsl:choose>
          <xsl:when test="$head-and-tail?2">
            <div>
              <xsl:apply-templates select="$head-and-tail?*"/>
            </div>
          </xsl:when>
          <xsl:otherwise>
            <xsl:apply-templates select="$head-and-tail?*"/>
          </xsl:otherwise>
        </xsl:choose>
      </xsl:for-each-group>
    </xsl:copy>
  </xsl:template>
  
</xsl:stylesheet>