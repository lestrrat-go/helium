<?xml version="1.0" encoding="UTF-8" standalone="no"?>

<!-- show that appply-templates passes parameters through a built-in template
     in 2.0 mode, but not in 1.0 mode -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema">

<xsl:template match="doc">
  <out>
    <v2 xsl:version="2.0">
      <xsl:apply-templates>
        <xsl:with-param name="x" select="42"/>
      </xsl:apply-templates>
    </v2>
    <v1 xsl:version="1.0">
      <xsl:apply-templates>
        <xsl:with-param name="x" select="42"/>
      </xsl:apply-templates>
    </v1>
  </out>
</xsl:template>

<xsl:template match="para">
  <xsl:param name="x" select="0"/>
  <x><xsl:value-of select="$x"/></x>
</xsl:template>

</xsl:stylesheet>
