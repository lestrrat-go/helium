<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of '=' operator, false value compared against zero. Reverse order of boolean18. -->
  <!-- Error: cannot compare boolean to number -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="0=false()"/>
  </out>
</xsl:template>

</xsl:stylesheet>
