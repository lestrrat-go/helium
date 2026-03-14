<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Basic test for mode="#all" (XSLT 2.0) -->

<xsl:template match="doc">
  <out>
    <m1><xsl:apply-templates select="*" mode="m1"/></m1>
    <m2><xsl:apply-templates select="*" mode="m2"/></m2>    
  </out>
</xsl:template>

<xsl:template match="a" mode="m1">
  <a mode="m1"/>
</xsl:template>

<xsl:template match="a" mode="m2">
  <a mode="m2"/>
</xsl:template>

<xsl:template match="b" mode="#all">
  <b mode="#all"/>
</xsl:template>

</xsl:stylesheet>
