<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- PURPOSE: Test of '=' operator one true value compared against a non-zero number. -->
  <xsl:template match="/">
  <out>
    <xsl:value-of select="true()=boolean(2)"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
