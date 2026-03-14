<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:foo="http://foo.com"
                xmlns:moo="http://foo.com"
                exclude-result-prefixes="foo moo">

  <!-- FileName: MODES16 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 5.7 Modes -->
    <!-- Purpose: Test that qualified name of a mode is used in expanded form. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc/a" mode="foo:a"/>
  </out>
</xsl:template>

<xsl:template match="text()" mode="moo:a">
  <xsl:text>mode-moo:a, </xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="text()">
  <xsl:text>no-mode, </xsl:text><xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>
