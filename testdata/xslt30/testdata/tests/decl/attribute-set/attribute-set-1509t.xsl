<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Holds template and attribute-set definition to be included. -->

<xsl:template match="doc">
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="foo">
  <xsl:element name="foocopy" use-attribute-sets="colorset decoset fontset">
    <xsl:value-of select="."/>
  </xsl:element>
</xsl:template>

<xsl:attribute-set name="fontset">
  <xsl:attribute name="font-size">14pt</xsl:attribute>
</xsl:attribute-set>

</xsl:stylesheet>
