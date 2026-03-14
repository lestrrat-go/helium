<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- Purpose: Try to select a non-existent attribute out of a node-set variable -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/page">
  <out><xsl:text>calling...</xsl:text>
    <xsl:call-template name="BrandHeader">
      <xsl:with-param name="hotelnode" select="/page/contents/avail/hotel"/>
    </xsl:call-template>
  </out>
</xsl:template>

<xsl:template name="BrandHeader">
  <xsl:param name="hotelnode"/>
  <xsl:value-of select="$hotelnode/location/@country"/>
</xsl:template>

</xsl:stylesheet>