<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Import precedence applies to top-level params
     (first template from middle, then up/down/up) -->

  <xsl:import href="variable-1903a.xsl"/>
  
  <xsl:param name="test" select="'main stylesheet, should have highest precedence'"/>

  <xsl:template match="a">
    <main>
      <xsl:value-of select="$test"/>
    </main>
    <xsl:apply-templates/>
  </xsl:template>

  <xsl:template match="c">
    <main-again>
      <xsl:value-of select="$test"/>
    </main-again>
  </xsl:template>

</xsl:stylesheet>
