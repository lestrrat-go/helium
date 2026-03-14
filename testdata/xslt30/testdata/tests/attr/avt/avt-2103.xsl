<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="3.0">
  
  <xsl:output method="xml"/>
  
  <xsl:template match="/" name="xsl:initial-template">
    <test>
      <xsl:variable name="element-name" select="'foo'"/>
      <xsl:element name="{$element-name}" namespace="{}">test</xsl:element>
    </test>
  </xsl:template>
  
</xsl:stylesheet>