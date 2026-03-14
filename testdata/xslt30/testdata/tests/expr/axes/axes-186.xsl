<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
  <!-- Purpose: Compound test going "vertically" in the tree. -->
    <!-- The expression selects the set of all grandparents.
       "//*[count(ancestor::*) &gt;= 2]" is the set of all grandchildren. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:apply-templates select="//*[count(ancestor::*) &gt;= 2]/../parent::*"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:text> </xsl:text><xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>
