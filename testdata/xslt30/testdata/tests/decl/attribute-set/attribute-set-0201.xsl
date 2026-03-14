<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#attribute-sets?>
  <!-- PURPOSE: Set attributes of a LRE from multiple attribute sets. -->
  
  <xsl:template match="/">
    <out>
      <test1 xsl:use-attribute-sets="set1 set2"></test1>
    </out>
  </xsl:template>

  <xsl:attribute-set name="set2">
    <xsl:attribute name="text-decoration">underline</xsl:attribute>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="set1">
    <xsl:attribute name="color">black</xsl:attribute>
  </xsl:attribute-set>

</xsl:stylesheet>