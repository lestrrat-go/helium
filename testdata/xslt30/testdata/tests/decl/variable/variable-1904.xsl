<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Import precedence applies to top-level params
     (order of imports in this sheet matters) -->


  <xsl:import href="variable-1901b.xsl"/>
  <xsl:import href="variable-1904a.xsl"/>

  <xsl:template match="b">
    <main>
      <xsl:value-of select="$test"/>
    </main>
    <xsl:apply-templates/>
  </xsl:template>

</xsl:stylesheet>
