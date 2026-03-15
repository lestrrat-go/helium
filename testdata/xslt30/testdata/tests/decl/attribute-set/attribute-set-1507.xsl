<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#attribute-sets?>
  <!--  --><!-- ResultTree006 in NIST suite -->
  <!-- Purpose: Use xsl:element with multiple attribute sets that 
       inherit, but have overlapping attributes. -->

<xsl:template match="/">
  <out>
    <xsl:element name="element1" use-attribute-sets="set1"/>
  </out>
</xsl:template>

<xsl:attribute-set name="set2" use-attribute-sets="set3">
  <xsl:attribute name="color">blue</xsl:attribute>
</xsl:attribute-set>

<xsl:attribute-set name="set1" use-attribute-sets="set2">
  <xsl:attribute name="color">black</xsl:attribute>
</xsl:attribute-set>

<xsl:attribute-set name="set3">
  <xsl:attribute name="color">green</xsl:attribute>
</xsl:attribute-set>

</xsl:stylesheet>