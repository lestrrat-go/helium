<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- PURPOSE: Test streamable="false" and "0" attribute -->
  <?spec xslt#attribute-sets?>
  
  <xsl:param name="color" select="'black'"/>
  <xsl:param name="texture" select="'matt'"/>

  <xsl:attribute-set name="set1" streamable="false">
    <xsl:attribute name="color" select="$color"/>
  </xsl:attribute-set>
  
  <xsl:attribute-set name="set2" streamable="0">
    <xsl:attribute name="texture" select="$texture"/>
  </xsl:attribute-set>

  <xsl:template match="/">
    <out>
      <test4 xsl:use-attribute-sets="set1 set2"></test4>
    </out>
  </xsl:template>

</xsl:stylesheet>
