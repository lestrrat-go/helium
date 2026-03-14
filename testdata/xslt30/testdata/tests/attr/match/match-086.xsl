<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test of id('literal', $DOC) as match pattern, new in XSLT 2.1. -->
  
<xsl:variable name="x" select="doc('match62.xml')"/>  

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="$x//a"/>
  </out>
</xsl:template>

<xsl:template match="id('c', $x)">
  <xsl:value-of select="name(.)"/> = <xsl:value-of select="."/>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>