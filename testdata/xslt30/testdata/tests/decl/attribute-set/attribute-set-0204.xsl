<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#attribute-sets?>
  <!-- PURPOSE: Set attributes of a LRE using attribute sets that inherit. -->
  
  <xsl:template match="/">
    <out>
      <test1 xsl:use-attribute-sets="set1"></test1>
    </out>
  </xsl:template>

  <xsl:attribute-set name="set2"  use-attribute-sets="set3">
    <xsl:attribute name="text-decoration">underline</xsl:attribute>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="set1" use-attribute-sets="set2">
    <xsl:attribute name="color">black</xsl:attribute>
  </xsl:attribute-set>

  <xsl:attribute-set name="set3">
    <xsl:attribute name="font-size">14pt</xsl:attribute>
  </xsl:attribute-set>

</xsl:stylesheet>