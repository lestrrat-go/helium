<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Set some attributes from an imported definition. -->

  <xsl:import href="attribute-set-1508a.xsl"/>
  <!-- defines colorset -->
  <xsl:import href="attribute-set-1508t.xsl"/>

  <xsl:template match="/">
    <out>
      <xsl:element name="test" use-attribute-sets="colorset decoset"/>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

  <xsl:attribute-set name="decoset">
    <xsl:attribute name="text-decoration">underline</xsl:attribute>
  </xsl:attribute-set>

</xsl:stylesheet>
