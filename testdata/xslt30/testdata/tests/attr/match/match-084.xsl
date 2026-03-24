<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: Test match="$x" - not allowed in 2.0. -->
  
<xsl:variable name="x" select="/doc"/>  
<xsl:variable name="y">
    <YYY/>
</xsl:variable> 

<xsl:template match="$x">
  <out>
    <one/>
    <xsl:apply-templates select="$y"/>
  </out>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>