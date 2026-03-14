<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test for conflict Resolution - 2 non-simples (predicate and '/')
     1 conflict warning should be seen.
     Should say "Match of non-simple '/'" -->

<xsl:template match="doc">
    <out> 
      <xsl:apply-templates select="foo"/>
	</out>
</xsl:template>
  
<xsl:template match="foo[@test]">
    <xsl:text>Match-of-non-simple '[...]'</xsl:text>
</xsl:template>

<xsl:template match="doc/foo">
    <xsl:text>Match-of-non-simple '/'</xsl:text>
</xsl:template>

<xsl:template match="foo">
    <xsl:text>Match-of-qualified-name</xsl:text>
</xsl:template>

<xsl:template match="*">
    <xsl:text>Match-of-wildcard</xsl:text>
</xsl:template>

<xsl:template match="node()">
    <xsl:text>Match-of-node-type</xsl:text>
</xsl:template>

</xsl:stylesheet>
