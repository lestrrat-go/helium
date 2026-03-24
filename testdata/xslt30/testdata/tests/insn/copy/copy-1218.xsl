<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- test xsl:copy on-empty - empty attribute node -->

    
<xsl:template match="/">
  <xsl:variable name="var1" as="node()"><a x="" y="3"/></xsl:variable>
  <out>
    <xsl:copy select="$var1//@x"/>
    <xsl:on-empty select="$var1//a"/>
  </out>  
</xsl:template>

</xsl:stylesheet>