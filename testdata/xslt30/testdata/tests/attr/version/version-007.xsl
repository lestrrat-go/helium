<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="8.5">

  <!-- PURPOSE: In forwards compatible mode, unknown attributes on XSL elements should be accepted. -->
  <?spec xslt#forwards?>
  <xsl:template match="/">
    <out xsl:version="32.0">
        <xsl:variable name="pigs" select="21" with-jam-on-top="yes"/>
        <xsl:value-of select="$pigs"/>
    </out>
  </xsl:template>
   
</xsl:stylesheet>
