<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- FileName: r -->
<!-- Purpose: Used by impincl14. -->

<xsl:template match="title">
  <xsl:text>R-title shouldn't be used</xsl:text>
</xsl:template>

<xsl:template match="author">
  R-author:<xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>