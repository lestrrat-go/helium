<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test position() when template is imported. -->

  <xsl:import href="position-4101a.xsl"/>

  <xsl:output method="xml" indent="no" encoding="UTF-8"/>

  <xsl:template match="doc">
    <out>
      <xsl:apply-templates select="a"/>
      <xsl:apply-templates select="b"/>
      <xsl:apply-templates select="c"/>
    </out>
  </xsl:template>

  <xsl:template match="a">
    <xsl:text>&#xa;</xsl:text>
    <local>
      <xsl:text>Item </xsl:text>
      <xsl:value-of select="@mark"/>
      <xsl:text> is in position </xsl:text>
      <xsl:value-of select="position()"/>
    </local>
  </xsl:template>

  <xsl:template match="c">
    <xsl:text>&#xa;</xsl:text>
    <apply level="main">
      <xsl:text>Item </xsl:text>
      <xsl:value-of select="@mark"/>
      <xsl:text> is in position </xsl:text>
      <xsl:value-of select="position()"/>
    </apply>
    <xsl:apply-imports/>
  </xsl:template>

</xsl:stylesheet>
