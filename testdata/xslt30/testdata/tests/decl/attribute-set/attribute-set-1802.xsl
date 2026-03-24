<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Only top-level variables and params are visible within
     the declaration of an attribute set. -->


  <xsl:variable name="foo" select="'correct'"/>

  <xsl:template match="/">
    <xsl:variable name="foo" select="'incorrect'"/>
    <out xsl:use-attribute-sets="attrs"/>
  </xsl:template>

  <xsl:attribute-set name="attrs">
    <xsl:attribute name="test">
      <xsl:value-of select="$foo"/>
    </xsl:attribute>
  </xsl:attribute-set>

</xsl:stylesheet>
