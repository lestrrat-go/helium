<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#node-tests?>
  <!-- Purpose: Step through names on the namespace axis. Ensure attributes aren't counted. -->

<xsl:output method="xml" encoding="UTF-8" indent="no" />

<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="doc">
  <xsl:text>doc #</xsl:text>
  <xsl:number level="single" count="doc" from="docs"/>
  <xsl:text>: &#010;</xsl:text>
  <xsl:for-each select="namespace::*">
    <xsl:sort select="name(.)"/>
    <xsl:element name="{name(.)}"><xsl:value-of select="."/></xsl:element>
  </xsl:for-each>
  <xsl:text>&#010;</xsl:text>
</xsl:template>

<xsl:template match="text()"/><!-- To suppress empty lines -->

</xsl:stylesheet>
