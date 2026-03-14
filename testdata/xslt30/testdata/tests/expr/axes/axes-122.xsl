<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#eval_context?>
    <!-- Purpose: Test for last() on two-step paths, contrasting descendant-or-self with ancestor-or-self. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:text>From root: </xsl:text>
    <xsl:value-of select="count(descendant-or-self::*/@att1)"/><xsl:text>&#10;</xsl:text>
    <xsl:apply-templates select="doc"/>
  </out>
</xsl:template>

<xsl:template match="doc">
  <xsl:text>From doc: </xsl:text>
  <xsl:value-of select="count(descendant-or-self::*/@att1)"/><xsl:text>= </xsl:text>
  <xsl:value-of select="(descendant-or-self::*/@att1[last()])[1]"/><xsl:text>, </xsl:text>
  <xsl:value-of select="descendant-or-self::*[last()]/@att1"/><xsl:text>, </xsl:text>
  <xsl:value-of select="(descendant-or-self::*/@att1)[last()]"/><xsl:text>; &#10;</xsl:text>
  <!-- Now reposition on the lowest node and look upward. -->
  <xsl:for-each select="//baz">
    <xsl:text>From baz: </xsl:text>
    <xsl:value-of select="count(ancestor-or-self::*/@att1)"/><xsl:text>= </xsl:text>
      <xsl:value-of select="(ancestor-or-self::*/@att1[last()])[1]"/><xsl:text>, </xsl:text>
      <xsl:value-of select="((ancestor-or-self::*)/@att1[last()])[1]"/><xsl:text>, </xsl:text>
      <xsl:value-of select="((ancestor-or-self::*/@att1)[last()])[1]"/><xsl:text>, </xsl:text>
      <xsl:value-of select="((ancestor::*|self::*)/@att1[last()])[1]"/><xsl:text>, </xsl:text>
      <xsl:value-of select="((ancestor::*|self::*)/@att1)[last()]"/><xsl:text>; &#10;</xsl:text>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>