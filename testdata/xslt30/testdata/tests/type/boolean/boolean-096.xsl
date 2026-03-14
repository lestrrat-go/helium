<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of '=' operator, false value compared against zero. Reverse order of boolean18. 
       Uses backwards compatibility mode -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="0=false()"/>
  </out>
</xsl:template>

</xsl:stylesheet>
