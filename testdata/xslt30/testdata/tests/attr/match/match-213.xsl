<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: match="/.." is a syntax error -->

  <xsl:template name="main">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:template match="/..">
    <xsl:value-of select="@att1"/>
    <xsl:apply-templates/>
  </xsl:template>

</xsl:stylesheet>
