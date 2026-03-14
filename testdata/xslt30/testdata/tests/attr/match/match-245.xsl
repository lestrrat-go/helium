<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
  <!-- Purpose: Test of local variables in match pattern with mode="#all". -->
  
<xsl:param name="target" select="'b'"/>
<xsl:strip-space elements="*"/>  

<xsl:template match="letters">
  <out>
    <xsl:apply-templates mode="m1"/>
    <xsl:apply-templates mode="m2"/>
    <xsl:apply-templates mode="m3"/>
  </out>
</xsl:template>

<xsl:template match="letter[some $x in tokenize(., '-') satisfies $x = $target]" mode="#all">
  <in><xsl:value-of select="."/></in>
</xsl:template>

<xsl:template match="letter" mode="#all"/>

</xsl:stylesheet>
