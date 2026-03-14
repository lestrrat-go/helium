<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Set attributes of an element created with xsl:copy from multiple attribute sets. -->
  
  <xsl:template match="foo">
    <out>
      <xsl:copy use-attribute-sets="set1 set2"/>
    </out>
  </xsl:template>
  
  <xsl:attribute-set name="set2">
    <xsl:attribute name="text-decoration">underline</xsl:attribute>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="set1">
    <xsl:attribute name="color">black</xsl:attribute>
  </xsl:attribute-set>
  
  <xsl:template match="text()"/>

</xsl:stylesheet>
