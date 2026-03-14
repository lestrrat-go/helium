<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test of large rule-set with similar predicates. -->

<xsl:template match="*[. = '123']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '124']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '125']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '126']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '127']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '128']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '129']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '130']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '131']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '132']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '133']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '134']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '135']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '136']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '137']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '138']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '139']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '140']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '141']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*[. = '142']">
  <xsl:value-of select="name()"/>
</xsl:template>

<xsl:template match="*"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="*/*"/>
  </out>  
</xsl:template>


</xsl:stylesheet>
