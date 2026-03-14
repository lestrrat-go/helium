<?xml version="1.0"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:s="http://s.com/" exclude-result-prefixes="s">

<!-- PURPOSE:   Test pattern matching on parentless elements -->


<xsl:template match="/">
  <xsl:variable name="x" as="node()*">
    <xsl:call-template name="make-elements"/>
  </xsl:variable>
  <do>
   <xsl:apply-templates select="$x" mode="m"/>
  </do>
</xsl:template>

<xsl:template name="make-elements" as="node()*">
  <aaa>1</aaa>
  <bbb>2</bbb>
  <ccc>3</ccc>
  <xsl:document><ddd>4</ddd></xsl:document>
</xsl:template>

<xsl:template match="aaa | bbb | ccc | ddd" mode="m">
  <xsl:copy-of select="."/>
</xsl:template>

<xsl:template match="/bbb" priority="10" mode="m">
  <error>/bbb shouldn't match</error>
</xsl:template>

<xsl:template match="//ccc" priority="10" mode="m">
  <error>//ccc shouldn't match</error>
</xsl:template>

<xsl:template match="//ddd" priority="10" mode="m">
  <ddd-got-it/>
</xsl:template>

</xsl:stylesheet>
