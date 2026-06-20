<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:mode name="special"/>
  <xsl:template match="item" mode="special">special:<xsl:value-of select="."/></xsl:template>
  <xsl:template match="item">default:<xsl:value-of select="."/></xsl:template>
</xsl:stylesheet>
