<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-false?>
  <!-- PURPOSE: Test of false() function -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="false()"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
