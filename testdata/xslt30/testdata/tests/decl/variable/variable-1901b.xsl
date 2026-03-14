<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: to be imported by impmid and others -->

  <xsl:param name="test" select="'test value set in impfinal'"/>

  <xsl:template match="/">
    <out>
      <final>
        <xsl:value-of select="$test"/>
      </final>
      <xsl:apply-templates/>
    </out>
  </xsl:template>

</xsl:stylesheet>
