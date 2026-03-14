<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:foo="http://foo.test.com"
    xmlns:joes="http://joes.com"
    xmlns:huh="http://unknown.com">

  <!-- FileName: copy46 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.2 -->
    <!-- Purpose: Introduce namespace nodes through copy-of where select ignores namespace. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc"/>
  </out>
</xsl:template>

<xsl:template match="doc">
  <star>
    <xsl:copy-of select="*[local-name()='bar']" />
  </star>
</xsl:template>

</xsl:stylesheet>