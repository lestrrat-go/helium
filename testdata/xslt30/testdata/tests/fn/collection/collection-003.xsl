<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
<!-- PURPOSE: Get the default collection (by default this is an empty collection). -->

<xsl:template match="/">
  <result><xsl:copy-of select="collection()"/></result>
</xsl:template>
       
</xsl:stylesheet>
