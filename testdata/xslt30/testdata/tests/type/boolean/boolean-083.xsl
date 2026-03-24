<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of '=' operator, boolean to string, backwards compatibility mode. Reverse order of boolean15. -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="'0'=true()"/>
  </out>
</xsl:template>

</xsl:stylesheet>
