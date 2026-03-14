<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- test xsl:copy on-empty - non-empty comment node -->

    
<xsl:template match="/">
  <xsl:variable name="var1" as="node()"><xsl:comment>Boo!</xsl:comment></xsl:variable>
  <out>
    <xsl:where-populated>
      <xsl:copy select="$var1"/>
    </xsl:where-populated>  
  </out>  
</xsl:template>

</xsl:stylesheet>