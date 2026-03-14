<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

  <!-- Creator: Michael Kay -->
  <!-- Purpose: apply-imports with numeric pattern -->



<xsl:template match=".[. instance of xs:integer]">
  <second><xsl:apply-imports/></second>
</xsl:template>

</xsl:stylesheet>
