<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-arithmetic?>
    <!-- Purpose: Test that * has precedence over + and -. -->

<xsl:variable name="anum" select="10"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="3*2+5*4-4*2-1"/><xsl:text>,</xsl:text>
    <xsl:value-of select="n6*5-8*n2+5*2"/><xsl:text>,</xsl:text>
    <xsl:value-of select="$anum*5-4*n2+n6*n1 -n3*3"/>
  </out>
</xsl:template>

</xsl:stylesheet>
