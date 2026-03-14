<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: to be imported -->

  <xsl:import href="variable-1903b.xsl"/>

  <xsl:param name="test" select="'test value set in mid'"/>

  <xsl:template match="/">
    <out>
      <middle>
        <xsl:value-of select="$test"/>
      </middle>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

</xsl:stylesheet>
