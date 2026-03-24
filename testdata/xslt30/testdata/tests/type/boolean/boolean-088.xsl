<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of '!=' operator on two booleans, false first. -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="false()!=true()"/>
  </out>
</xsl:template>

</xsl:stylesheet>