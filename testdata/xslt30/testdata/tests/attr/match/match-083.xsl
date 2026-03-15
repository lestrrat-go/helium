<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test match="$x//a" - disallowed in 2.0. -->
  
<xsl:variable name="x" select="/doc"/>  
<xsl:variable name="y">
    <xsl:copy-of select="."/>
</xsl:variable> 

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
    <xsl:apply-templates select="$y/doc/foo"/>
  </out>
</xsl:template>

<xsl:template match="$x//baz">
  <first><xsl:value-of select="name(.)"/> = <xsl:value-of select="@att1"/></first>
</xsl:template>

<xsl:template match="$y//baz">
  <second><xsl:value-of select="name(.)"/> = <xsl:value-of select="@att1"/></second>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>