<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:template match="title">
  D-title:
</xsl:template> 

<xsl:template match="overview">
  D-overview: 
  <xsl:value-of select="."/>
</xsl:template>
   
</xsl:stylesheet>
