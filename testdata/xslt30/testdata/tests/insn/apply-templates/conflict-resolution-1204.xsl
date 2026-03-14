<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Testing xsl:next-match using two templates with different import precedence -->

<xsl:import href="conflict-resolution-1204a.xsl"/>

<xsl:template match="doc" priority="10">
   <out>
      <xsl:apply-templates select="foo"/>
   </out>
</xsl:template>
  
<xsl:template match="foo" priority="5">
    <xsl:text>(5)</xsl:text>
    <xsl:next-match/>
</xsl:template>

<xsl:template match="foo" priority="2">
    <xsl:text>(2)</xsl:text>
    <xsl:next-match/>
</xsl:template>

<xsl:template match="*" priority="3">
    <xsl:text>(3)</xsl:text>
    <xsl:next-match/>
</xsl:template>

<xsl:template match="node()" priority="4">
    <xsl:text>(4)</xsl:text>
    <xsl:next-match/>
</xsl:template>

<xsl:template match="text()" priority="10">
 
</xsl:template>

</xsl:stylesheet>
