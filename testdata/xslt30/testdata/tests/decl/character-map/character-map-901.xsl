<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  expand-text="yes">
  
  <xsl:mode on-no-match="shallow-copy"/>
  
  <xsl:output use-character-maps="m1"/>
  
  <xsl:character-map name="m1">
    <xsl:output-character char="A" string="&amp;nbsp;"/>
  </xsl:character-map>
  
  <xsl:template name="xsl:initial-template">
    <xsl:text>AAABAAA</xsl:text>
  </xsl:template>
  
</xsl:stylesheet>