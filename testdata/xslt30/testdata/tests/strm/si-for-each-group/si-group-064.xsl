<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <xsl:strip-space elements="*"/>
  <xsl:output indent="yes"/>
  
  <xsl:mode on-no-match="shallow-copy" streamable="yes"/>
  
  <xsl:template match="Member">
    <xsl:copy>
      <xsl:for-each-group select="*" group-adjacent="node-name()">
        <xsl:apply-templates select="current-group()!copy-of()" mode="add-index"/>
      </xsl:for-each-group>
    </xsl:copy>
  </xsl:template>
  
  <xsl:mode name="add-index" on-no-match="shallow-copy"/>
  
  <xsl:template match="*[not(parent::*)]" mode="add-index">
    <xsl:element name="{node-name()}{if (last() gt 1) then '_' || position() else ()}">
      <xsl:apply-templates mode="#current"/>
    </xsl:element>
  </xsl:template>
  
</xsl:stylesheet>