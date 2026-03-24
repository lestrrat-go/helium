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
        <xsl:for-each select="current-group()!copy-of()">
          <xsl:element name="{node-name()}{if (last() gt 1) then '_' || position() else ()}">
            <xsl:copy-of select="child::node()"/>
          </xsl:element>
        </xsl:for-each>
      </xsl:for-each-group>
    </xsl:copy>
  </xsl:template>
  

  
</xsl:stylesheet>