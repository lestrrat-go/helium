<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- FileName: e -->
<!-- Purpose: Used by c.xsl, which is used by several tests. -->

<xsl:template match="title">
  E-title: <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="chapter">
    E-chapter <xsl:value-of select="@num"/><xsl:text>: </xsl:text>
  <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>