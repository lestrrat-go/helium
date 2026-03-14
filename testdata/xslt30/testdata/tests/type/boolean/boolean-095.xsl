<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of '=' operator with one number with leading zero, one not. Reverse order of boolean14.
       Uses backwards compatibility mode -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="'001' = 1"/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
