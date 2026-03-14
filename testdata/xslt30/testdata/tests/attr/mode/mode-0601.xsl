<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: modes13 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 5.7 -->
  <!-- Template003 in NIST suite -->
  <!-- Purpose: Test apply-templates for any node() with a mode
       and moded matching template. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="node()" mode="mode1"/>
  </out>
</xsl:template>

<xsl:template match="node()" mode="mode1">
  <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="node()">
  This test failed to execute properly.
</xsl:template>

</xsl:stylesheet>
