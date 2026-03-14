<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of '!=' operator on two strings that are equal, so false should result. -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="'ace' != 'ace'"/>
  </out>
</xsl:template>

</xsl:stylesheet>