<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:foo="http://foo.test.com"
    xmlns:huh="http://unknown.com"
    exclude-result-prefixes="huh">

  <!-- FileName: copy47 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 7.1.1 (and 11.3) -->
    <!-- Purpose: Show that exclude-result-prefixes doesn't affect copy-of. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc"/>
  </out>
</xsl:template>

<xsl:template match="doc">
  <union>
    <xsl:text>&#10;</xsl:text>
    <xsl:copy-of select="bar | foo:bar" />
  </union>
</xsl:template>

</xsl:stylesheet>