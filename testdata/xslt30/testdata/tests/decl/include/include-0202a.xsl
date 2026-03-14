<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  
  <xsl:template match="doc">
    <xsl:param name="magic"/>
    <in>
     <xsl:attribute name="magic"><xsl:value-of select="$magic"/></xsl:attribute>
    </in>
  </xsl:template>
 
</xsl:stylesheet>
