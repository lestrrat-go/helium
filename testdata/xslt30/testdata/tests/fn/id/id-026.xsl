<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test id() filtered by a predicate, in a match pattern. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="tee/s">
      <xsl:apply-templates select="r"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="id('d')/*[@size &gt; 17]" priority="2">
  <xsl:text>&#10;</xsl:text>
  <bigD><xsl:value-of select="../@id"/></bigD>
</xsl:template>

<xsl:template match="r[@size &gt; 17]">
  <xsl:text>&#10;</xsl:text>
  <big><xsl:value-of select="../@id"/></big>
</xsl:template>

<xsl:template match="r">
  <xsl:text>&#10;</xsl:text>
  <other><xsl:value-of select="../@id"/></other>
</xsl:template>

</xsl:stylesheet>
