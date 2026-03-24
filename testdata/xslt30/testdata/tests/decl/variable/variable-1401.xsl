<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#local-variables?>
  <!-- Purpose: Test for-each inside xsl:variable -->
  

<xsl:template match="docs">
  <out>
    <xsl:variable name="all">
      <xsl:for-each select="a">
        <xsl:value-of select="."/>
      </xsl:for-each>
    </xsl:variable>
    <xsl:text>$all contains </xsl:text>
    <xsl:value-of select="$all"/>
  </out>
</xsl:template>

</xsl:stylesheet>
