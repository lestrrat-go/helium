<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:strip-space elements="foo"/>

<?spec xslt#patterns?>
    <!-- Purpose: Test for *:local-name match pattern (XSLT 2.0). -->

<xsl:template match="doc">
 <out>
 	<xsl:apply-templates/>
 </out>
</xsl:template>

<xsl:template match="*:element">
  [<xsl:value-of select="name()"/>}
  <xsl:apply-templates select="@*"/>
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="@*:attribute">
  (@<xsl:value-of select="name()"/>)
</xsl:template>

</xsl:stylesheet>
