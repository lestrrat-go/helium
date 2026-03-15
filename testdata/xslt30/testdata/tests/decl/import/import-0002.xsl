<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://impincl33/ns" exclude-result-prefixes="f">

  <!-- Purpose: XSLT 3.0 removes the rule that "xsl:import must come first"  -->

  <xsl:variable name="v" select="1"/>

  <xsl:template name="t">
    <xsl:value-of select="1"/>
  </xsl:template>

  <xsl:function name="f:f">
    <xsl:value-of select="1"/>
  </xsl:function>

  <xsl:template match="/">
    <xsl:value-of select="1"/>
  </xsl:template>

  <xsl:decimal-format minus-sign="~"/>

  <xsl:namespace-alias stylesheet-prefix="f" result-prefix="xsl"/>

  <xsl:attribute-set name="a">
    <xsl:attribute name="a" select="1"/>
  </xsl:attribute-set>

  <xsl:import href="import-0002a.xsl"/>

  <xsl:variable name="dummy">
    <a/>
  </xsl:variable>

  <xsl:template name="main">
    <out>
      <variable>
        <xsl:value-of select="$v"/>
      </variable>
      <call-template>
        <xsl:call-template name="t"/>
      </call-template>
      <function>
        <xsl:value-of select="f:f()"/>
      </function>
      <apply-template>
        <xsl:apply-templates select="$dummy"/>
      </apply-template>
      <decimal-format>
        <xsl:value-of select="format-number(-10000093.7, '0.00')"/>
      </decimal-format>
      <f:transform xsl:use-attribute-sets="a"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
