<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Set attributes of an element created with xsl:copy, -->
  <!-- using attribute sets that inherit, -->
  <!-- plus add overlapping attribute with xsl:attribute. -->
  
  <xsl:template match="foo">
    <out>
      <xsl:copy use-attribute-sets="set1">
        <xsl:attribute name="text-decoration">none</xsl:attribute>
      </xsl:copy>
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
  
  <xsl:template match="text()"/>

</xsl:stylesheet>