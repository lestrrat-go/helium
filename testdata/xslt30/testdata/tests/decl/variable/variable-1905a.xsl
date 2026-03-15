<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: to be imported  -->

  <xsl:import href="variable-1901b.xsl"/>

  <xsl:param name="test" select="'value set in var30mid, should have highest precedence'"/>

  <xsl:template match="b">
    <middle>
      <xsl:value-of select="$test"/>
    </middle>
    <xsl:apply-templates/>
  </xsl:template>

</xsl:stylesheet>
