<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:x="http://example.org/x?subject="
  xmlns:y="http://example.org/x?subject">

  <!-- Purpose: Namespace prefix fn is not allowed unless declared -->


  
  <xsl:template match="/">
   <out at="{fn:current-dateTime()}"/>
  </xsl:template>
  
 
</xsl:stylesheet>
