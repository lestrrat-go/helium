<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Purpose: To be imported by imp27b. -->
  <xsl:template match="doc">
    <xsl:copy-of select="num" />
  </xsl:template>

</xsl:stylesheet>