<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-arithmetic?>
    <!-- Purpose: Test that mod has precedence over + and -. -->

<xsl:variable name="anum" select="10"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="48 mod 17 - 2 mod 9 + 13 mod 5"/><xsl:text>,</xsl:text>
    <xsl:value-of select="56 mod round(n5*2+1.444) - n6 mod 4 + 7 mod n4"/><xsl:text>,</xsl:text>
    <xsl:value-of select="(77 mod $anum + n5 mod 8) mod $anum"/>
  </out>
</xsl:template>

</xsl:stylesheet>
