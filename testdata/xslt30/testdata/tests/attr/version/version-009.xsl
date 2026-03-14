<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.3">

<?spec xslt#forwards?>
  <!-- PURPOSE: Test that in forwards compatibility mode, unknown
  top-level XSL elements are ignored. -->
  
  <xsl:ignore-this/>
  
  <xsl:template match="/">
    <out>Success</out>
  </xsl:template>
   
</xsl:stylesheet>
