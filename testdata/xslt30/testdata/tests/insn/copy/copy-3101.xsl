<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:foo="http://foo.test.com"
    xmlns:joes="http://joes.com">

  <!-- FileName: copy44 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.3 -->
    <!-- Purpose: Introduce namespace nodes through copy-of (i.e., no earlier reference). -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc"/>
  </out>
</xsl:template>

<xsl:template match="doc">
  <union>
    <xsl:text>&#10;</xsl:text>
    <xsl:copy-of select="bar | joes:bar" />
  </union>
</xsl:template>

</xsl:stylesheet>