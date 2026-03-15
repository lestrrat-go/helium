<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


    <!-- Purpose: Check the namespace axis starting from an attribute; should be empty -->


<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>
  
<xsl:template match="doc">
    <xsl:apply-templates select="@*"/>
</xsl:template>

<xsl:template match="doc/@*">
  <ns-count attr="{name(.)}" count="{count(namespace::*)}"/>
</xsl:template>

<xsl:template match="text()"/><!-- To suppress empty lines -->

</xsl:stylesheet>