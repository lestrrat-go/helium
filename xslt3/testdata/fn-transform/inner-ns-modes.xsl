<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:m="http://example.com/modes">
  <xsl:mode name="m:highlight"/>
  <xsl:template match="/" mode="m:highlight">
    <out>ns-mode-highlight</out>
  </xsl:template>
  <xsl:template match="/">
    <out>default-mode</out>
  </xsl:template>
</xsl:stylesheet>
