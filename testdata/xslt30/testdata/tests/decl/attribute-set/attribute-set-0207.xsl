<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#attribute-sets?>
  <!-- PURPOSE: Set attributes of an element created with xsl:element, -->
  <!-- using attribute sets that inherit, -->
  <!-- plus add overlapping attribute with xsl:attribute. -->
  
  <xsl:template match="/">
    <out>
      <xsl:element name="test" use-attribute-sets="set1">
        <xsl:attribute name="text-decoration">none</xsl:attribute>
      </xsl:element>
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