<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test of three predicates, two being positional.
     First predicate reduces the set to {a,c,e,g,i,k}.
     Second predicate further reduces the set to {c}.
     Third predicate, taken alone, reduces the set to {a,b,c,d,e,f,g,h,i}. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="root/x"/>
  </out>
</xsl:template>

<xsl:template match="x[(position() mod 2)=1][2][@num &lt; 10]">
  <xsl:value-of select="@spot"/>
  <xsl:text> </xsl:text>
</xsl:template>

</xsl:stylesheet>
