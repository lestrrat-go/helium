<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- test xsl:copy on-empty; ensure the on-empty node is copied -->

    
<xsl:template match="/">
  <xsl:variable name="var1"><rtf>abc</rtf></xsl:variable>
  <xsl:variable name="var2" as="element()">
    <xsl:copy select="doc">
      <xsl:copy-of select="//foo"/>
      <xsl:on-empty select="$var1/*"/>
    </xsl:copy>
  </xsl:variable>  
  <out copied="{not($var2 is $var1/*)}" equal="{deep-equal($var2, $var1/*)}"/>
</xsl:template>

</xsl:stylesheet>