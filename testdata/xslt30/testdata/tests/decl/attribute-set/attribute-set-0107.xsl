<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: A focus-dependent attribute set. (Fails in Saxon because of incorrect loop-lifting) -->
  <?spec xslt#attribute-sets?>

  <xsl:attribute-set name="set1">
    <xsl:attribute name="class" select="name()"/>
    <xsl:attribute name="position" select="position()"/>
  </xsl:attribute-set>

  <xsl:template match="/">
    <out>
      <xsl:for-each select="doc//*">
        <xsl:copy use-attribute-sets="set1"/>
      </xsl:for-each>  
    </out>
  </xsl:template>

</xsl:stylesheet>
