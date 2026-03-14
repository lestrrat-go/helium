<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- test xsl:copy on-empty - select empty sequence-->

    
<xsl:template match="/">
  <xsl:variable name="var1"><rtf>abc</rtf></xsl:variable>
  <out>
    <xsl:copy select="//foo"/>
    <xsl:on-empty select="$var1/*"/>
  </out>  
</xsl:template>

</xsl:stylesheet>