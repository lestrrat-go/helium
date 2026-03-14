<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
  <!-- PURPOSE: Test of '=' operator with fractional number one with leading zero one without. -->
<xsl:template match="/">
  <out>
    <xsl:value-of select="0 = -0"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
