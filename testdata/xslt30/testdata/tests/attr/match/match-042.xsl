<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <xsl:template match="*">
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="COORDINATES_ITEM[position() = /SHAPE/ELEM_INFO/ELEM_INFO_ITEM[position() mod 3 = 1]]">
    <xsl:copy>
      <xsl:text>d</xsl:text>
      <xsl:value-of select="."/>
    </xsl:copy>
  </xsl:template>
</xsl:stylesheet>
