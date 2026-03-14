<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: Test for node tests in select in for-each. -->

<xsl:template match="/doc">
  <out>
    <xsl:text>ATTRIBUTES:</xsl:text>
    <xsl:for-each select="@*">
      <xsl:text>A-</xsl:text><xsl:value-of select="name()"/>
    </xsl:for-each>
    <xsl:text>
TEXT:</xsl:text>
    <xsl:for-each select="text()">
      <xsl:text>T-</xsl:text><xsl:value-of select="."/>
    </xsl:for-each>
    <xsl:text>
COMMENTS:</xsl:text>
    <xsl:for-each select="comment()">
      <xsl:text>C-</xsl:text><xsl:value-of select="."/>
    </xsl:for-each>
    <xsl:text>
PIs:</xsl:text>
    <xsl:for-each select="processing-instruction()">
      <xsl:text>P-</xsl:text><xsl:value-of select="name()"/>
    </xsl:for-each>
    <xsl:text>
CHILDREN:</xsl:text>
    <xsl:for-each select="*">
      <xsl:text>E-</xsl:text><xsl:value-of select="name()"/><xsl:text>
</xsl:text>
      <xsl:apply-templates select="@*"/>
      <xsl:apply-templates/>
      <xsl:text>--End of child </xsl:text><xsl:value-of select="name()"/><xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*"><!-- for child elements -->
  <xsl:text>E:</xsl:text><xsl:value-of select="name()"/>
  <xsl:apply-templates select="@*"/>
  <xsl:apply-templates/>
  <xsl:text>
</xsl:text>
</xsl:template>

<xsl:template match="@*">
  <xsl:text>A:</xsl:text><xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="text()">
  <xsl:text>T:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="comment()">
  <xsl:text>C:</xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="processing-instruction()">
  <xsl:text>P:</xsl:text><xsl:value-of select="name()"/>
</xsl:template>

</xsl:stylesheet>
