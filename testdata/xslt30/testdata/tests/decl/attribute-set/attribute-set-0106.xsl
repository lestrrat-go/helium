<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: A terminating recursive attribute set. It's not entirely clear whether the spec allows this. -->
  <?spec xslt#attribute-sets?>

  <xsl:attribute-set name="set1">
    <xsl:attribute name="color">
          <xsl:for-each select="*">
            <xsl:variable name="x">
              <e xsl:use-attribute-sets="set1"/>
            </xsl:variable>
            <xsl:value-of select="string-join($x//@*, '|')"/>
          </xsl:for-each>
    </xsl:attribute>
    <xsl:attribute name="texture">matt</xsl:attribute>
  </xsl:attribute-set>

  <xsl:template match="/">
    <out>
      <test1 xsl:use-attribute-sets="set1"></test1>
    </out>
  </xsl:template>

</xsl:stylesheet>
