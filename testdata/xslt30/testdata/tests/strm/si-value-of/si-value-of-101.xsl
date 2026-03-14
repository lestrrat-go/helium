<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  version="3.0"
  exclude-result-prefixes="#all">
  
  <xsl:mode streamable="yes"/>
  
  <xsl:output method="text"/>
  <xsl:strip-space elements="*"/>
  
  <xsl:template match="/*">
    <out>
      <xsl:value-of separator="&#10;">
        <xsl:apply-templates select="outermost(descendant::record)"/>
      </xsl:value-of>
    </out>
  </xsl:template>
  
  <xsl:template match="record" as="xs:string">
    <xsl:value-of separator=",">
      <xsl:apply-templates/>  
    </xsl:value-of>
  </xsl:template>
  
  <xsl:template match="record/*">
    <xsl:sequence select="string()"/>
  </xsl:template>
  
  <xsl:template match="record/*[contains(., ',') or contains(., '&quot;')]" priority="5">
    <xsl:sequence select="concat('&quot;', replace(., '&quot;', '&quot;&quot;'), '&quot;')"/>
  </xsl:template>
  
</xsl:stylesheet>