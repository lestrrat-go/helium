<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  expand-text="yes">
  
  <xsl:strip-space elements="*"/>
  
  <xsl:output method="text"/>
  
  <xsl:mode streamable="yes"/>
  
  <xsl:template match="/" name="xsl:initial-template">
    <out>
      <xsl:for-each select="root/record">
        <row><xsl:value-of select="data(@att1), data(@att2), if (normalize-space(@att3)) then @att3 else 'default'" separator=","/></row>
      </xsl:for-each>
    </out>
  </xsl:template>
  
</xsl:stylesheet>