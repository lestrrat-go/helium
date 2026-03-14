<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:bdd="http://bdd.lotus.com"
                exclude-result-prefixes="bdd">

<?spec xpath#node-tests?>
    <!-- Purpose: Use a NameTest on the namespace axis. -->

<xsl:output method="xml" encoding="UTF-8" indent="no" />

<xsl:template match="/">
  <out>
    <xsl:text>&#10;</xsl:text>
    <xsl:apply-templates select="doc"/>
    <xsl:text>&#10;</xsl:text>
  </out>
</xsl:template>

<xsl:template match="doc">
  <xsl:for-each select="namespace::ped">
    <xsl:element name="{name(.)}"><xsl:value-of select="."/></xsl:element>
  </xsl:for-each>
  <xsl:text>&#10;</xsl:text>
  <!-- Do the same when the prefix is also declared here in the stylesheet. Shouldn't conflict. -->
  <xsl:for-each select="namespace::bdd">
    <xsl:element name="{name(.)}"><xsl:value-of select="."/></xsl:element>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>