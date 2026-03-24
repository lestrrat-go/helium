<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- PURPOSE: Use EQNames for attribute set names. -->
  <?spec xslt#attribute-sets?>

  <xsl:attribute-set name="Q{http://example.com/ns}set1">
    <xsl:attribute name="color">black</xsl:attribute>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="Q{http://example.com/ns}set2">
    <xsl:attribute name="texture">matt</xsl:attribute>
  </xsl:attribute-set>

  <xsl:template match="/">
    <out>
      <test1 xsl:use-attribute-sets="Q{http://example.com/ns}set1 Q{http://example.com/ns}set2"></test1>
    </out>
  </xsl:template>

</xsl:stylesheet>
