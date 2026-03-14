<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test of three predicates, two being positional. Use last() for one.
     First predicate reduces the set to {b,e,h,k}.
     Second predicate further reduces the set to {h,k}. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="root/x"/>
  </out>
</xsl:template>

<xsl:template match="x[(@num mod 3)=2][position() &gt; 2][last()]">
  <xsl:value-of select="@spot"/>
  <xsl:text> </xsl:text>
</xsl:template>

</xsl:stylesheet>
