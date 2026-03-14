<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- Purpose: Test for id() in complex structure. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="doc">
  Test for id selection and pattern matching...
  Next line should read: -*id17*-    <!-- No id() either end -->
  <xsl:apply-templates select="c/b"/>
  Next line should read: (*id14*)    <!-- id() below but not here -->
  <xsl:apply-templates select="b/c"/>
  Next line should read: -*id4*-     <!-- id() here on apply but not below -->
  <xsl:apply-templates select="id('id4')"/>
  Next line should read: +*id9*+     <!-- id() same on both ends -->
  <xsl:apply-templates select="id('id9')"/>
  Next line should read: (*id13*)    <!-- simple id() here, path off top-level id() below -->
  <xsl:apply-templates select="id('id13')"/>
  Next line should read: -*id6*-     <!-- path off top-level id() here but not below -->
  <xsl:apply-templates select="id('id0')/a/b"/>
  Next line should read: @*id19*@    <!-- same path off top-level id() here and below -->
  <xsl:apply-templates select="id('id0')/c/c/a"/>
  Next line should read: %*id12*%    <!-- different paths off id() here and below -->
  <xsl:apply-templates select="id('id8')/b/b"/>
  Next line should read: !*id11*!    <!-- different paths off id(), middle here and top-level below -->
  <xsl:apply-templates select="id('id10')/a"/>
  Next line should read: [*id3*] \*id5*\ =*id16*=  <!-- union of id()-based paths -->
  <xsl:apply-templates select="id('id2')/a | id('id5') | id('id15')/a"/>
</xsl:template>

<!-- Now the templates for the specific cases -->

<xsl:template match="c">
  Error if this (c) template fires!
</xsl:template>

<xsl:template match="b">
  -<xsl:apply-templates/>-
</xsl:template>

<xsl:template match="id('id5')">
  \<xsl:apply-templates/>\
</xsl:template>

<xsl:template match="id('id9')">
  +<xsl:apply-templates/>+
</xsl:template>

<xsl:template match="id('id16')">
  =<xsl:apply-templates/>=
</xsl:template>

<xsl:template match="id('id0')/a/a/a">
  [<xsl:apply-templates/>]
</xsl:template>

<xsl:template match="id('id0')/b/b/a">
  !<xsl:apply-templates/>!
</xsl:template>

<xsl:template match="id('id0')/c/c/a">
  @<xsl:apply-templates/>@
</xsl:template>

<xsl:template match="id('id8')//b" priority="1">
  %<xsl:apply-templates/>%
</xsl:template>

<xsl:template match="id('id8')//c" priority="2">
  (<xsl:apply-templates/>)
</xsl:template>

</xsl:stylesheet>