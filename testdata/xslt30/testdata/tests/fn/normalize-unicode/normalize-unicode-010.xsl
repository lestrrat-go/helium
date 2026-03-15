<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:strip-space elements="doc"/>
  
  <!-- trivial test of normalized output (NFC) -->
  
  <xsl:output indent="no" encoding="iso-8859-1"/>
  <xsl:output method="xml" normalization-form="NFC"/>
  
  <xsl:template match="/">
    <xsl:copy-of select="."/>
  </xsl:template>
    

</xsl:stylesheet>
