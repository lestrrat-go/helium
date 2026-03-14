<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- test xsl:copy on-empty -->

    
<xsl:template match="/">
  <xsl:variable name="var1"><rtf>abc</rtf></xsl:variable>
  <out>
    <xsl:copy select="doc">
      <xsl:copy-of select="//foo"/>
      <xsl:on-empty select="$var1/*"/>
    </xsl:copy>
  </out>
</xsl:template>

</xsl:stylesheet>