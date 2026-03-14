<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#creating-attributes?>
  <!-- PURPOSE: xsl:attribute with content and separators (XSLT 2.0). -->
  <?same-as-1.0 no?>
  <xsl:attribute-set name="set1">
    <xsl:attribute name="color" separator="|">
      <xsl:value-of>black</xsl:value-of>
      <xsl:value-of>blue</xsl:value-of>
      <xsl:value-of>green</xsl:value-of>
    </xsl:attribute>
  </xsl:attribute-set>

  <xsl:template match="/">
    <out>
      <test1 xsl:use-attribute-sets="set1"></test1>
    </out>
  </xsl:template>

</xsl:stylesheet>