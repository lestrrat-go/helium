<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Import precedence applies to top-level params
     (start at bottom; all use variable set in middle) -->


  <xsl:import href="variable-1905a.xsl"/>

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
