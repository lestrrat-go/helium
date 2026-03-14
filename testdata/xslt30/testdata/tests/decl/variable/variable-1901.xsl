<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Import precedence applies to top-level params
     (testing references from various import levels) -->

  <xsl:import href="variable-1901a.xsl"/>
  
  <xsl:param name="test" select="'main stylesheet, should have highest precedence'"/>

  <xsl:template match="b">
    <main>
      <xsl:value-of select="$test"/>
    </main>
  </xsl:template>

</xsl:stylesheet>
