<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode name="special"/>
  <xsl:template match="/" mode="special">
    <out>special-mode</out>
  </xsl:template>
  <xsl:template match="/">
    <out>default-mode</out>
  </xsl:template>
</xsl:stylesheet>
