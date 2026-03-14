<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test match="doc(X)//a" - disallowed in 2.0. -->
  
<xsl:strip-space elements="*"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="doc('match02.xml')"/>
  </out>
</xsl:template>

<xsl:template match="doc('match02.xml')">
  <first><xsl:apply-templates/></first>
</xsl:template>

<xsl:template match="doc('match02.xml')/doc/foo/a">
  <two><xsl:copy-of select="."/></two>
</xsl:template>


</xsl:stylesheet>