<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: To be imported  -->

  <xsl:template match="b">
    <xsl:text>&#xa;</xsl:text>
    <direct>
      <xsl:text>Item </xsl:text>
      <xsl:value-of select="@mark"/>
      <xsl:text> is in position </xsl:text>
      <xsl:value-of select="position()"/>
    </direct>
  </xsl:template>

  <xsl:template match="c">
    <xsl:text>&#xa;</xsl:text>
    <apply level="import">
      <xsl:text>Item </xsl:text>
      <xsl:value-of select="@mark"/>
      <xsl:text> is in position </xsl:text>
      <xsl:value-of select="position()"/>
    </apply>
  </xsl:template>

</xsl:stylesheet>
