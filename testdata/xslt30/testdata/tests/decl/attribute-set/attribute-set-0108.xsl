<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Invoking an attribute set does not change the current template rule -->
  <?spec xslt#attribute-sets?>

  <xsl:attribute-set name="set1">
    <xsl:attribute name="foo">
      <xsl:next-match/>
    </xsl:attribute>
  </xsl:attribute-set>

  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="doc"/>  
    </out>
  </xsl:template>
  
  <xsl:template match="doc" priority="2">
    <in xsl:use-attribute-sets="set1"/>
  </xsl:template>
  
  <xsl:template match="doc" priority="1">
    <xsl:value-of select="'baa'"/>
  </xsl:template>

</xsl:stylesheet>
