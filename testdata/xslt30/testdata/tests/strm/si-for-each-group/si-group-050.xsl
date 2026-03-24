<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
  exclude-result-prefixes="xs math" 
  version="3.0">
  
  <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
  
  <xsl:mode _streamable="{$STREAMABLE}"/>
  
  <xsl:output indent="no"/>
  <xsl:strip-space elements="*"/>
  
  <xsl:template match="Orders">
    <xsl:copy>
      <xsl:for-each-group select="copy-of(Order)" group-adjacent="OrderNumber">
        <xsl:variable name="highest-date" select="max(current-group()/Date)"/>
        <xsl:for-each select="current-group()">
          <xsl:copy>
            <xsl:copy-of select="OrderNumber"/>
            <Date>
              <xsl:value-of select="$highest-date"/>
            </Date>
          </xsl:copy>
        </xsl:for-each>
      </xsl:for-each-group>
    </xsl:copy>
  </xsl:template>
  
</xsl:stylesheet>