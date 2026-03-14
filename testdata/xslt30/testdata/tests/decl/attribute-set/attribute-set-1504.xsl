<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#attribute-sets?>
  <!-- Purpose: Use xsl:copy with multiple attribute sets in "merge" scenario. -->
  <!--  --><!-- simplified from ResultTree004 in NIST suite -->
  
  <!-- test corrected by MHK. Original uses xsl:copy when the context node is a document node;
       the spec says use-attribute-sets is used only when copying elements. -->

<xsl:template match="/*">
  <out>
    <xsl:copy use-attribute-sets="set1"/>
  </out>
</xsl:template>

<xsl:attribute-set name="set1">
  <xsl:attribute name="text-decoration">underline</xsl:attribute>
</xsl:attribute-set>

<xsl:attribute-set name="set1">
  <xsl:attribute name="color">black</xsl:attribute>
</xsl:attribute-set>

<xsl:attribute-set name="set1">
  <xsl:attribute name="font-size">14pt</xsl:attribute>
</xsl:attribute-set>

</xsl:stylesheet>