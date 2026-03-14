<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Set attribute of a LRE from single attribute set. -->
  <?spec xslt#attribute-sets?>

  <xsl:attribute-set name="set1">
    <xsl:attribute name="color">black</xsl:attribute>
  </xsl:attribute-set>

  <xsl:template match="/">
    <out>
      <test1 xsl:use-attribute-sets="set1"></test1>
    </out>
  </xsl:template>

</xsl:stylesheet>
