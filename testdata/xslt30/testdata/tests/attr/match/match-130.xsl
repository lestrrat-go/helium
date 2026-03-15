<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

    <!-- Purpose: Pattern for matching atomic values with a predicate: new syntax-->

<xsl:template name="main">
 <out>
 	<xsl:apply-templates select="-2 to +2"/>
 </out>
</xsl:template>

<xsl:template match=".[ .=0 ]">
  <zero><xsl:value-of select="."/></zero>
</xsl:template>

<xsl:template match=".[. lt 0 ]">
  <negative><xsl:value-of select="."/></negative>
</xsl:template>

<xsl:template match=".[. gt 0]">
  <positive><xsl:value-of select="."/></positive>
</xsl:template>

</xsl:stylesheet>
