<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-true?>
  <!-- PURPOSE: Test of true() -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="true()"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
