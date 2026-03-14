<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test xsl:message with terminate as an AVT (XSLT 2.0). -->
  <?spec xslt#message?>
  <xsl:template match="/">
    <out>
        <xsl:message terminate="{$terminate}">Terminating as requested: <xsl:value-of select="$terminate"/>&#xa;</xsl:message>
        <still-here/>
    </out>
  </xsl:template>
  
  <xsl:param name="terminate">yes</xsl:param>
   
</xsl:stylesheet>
