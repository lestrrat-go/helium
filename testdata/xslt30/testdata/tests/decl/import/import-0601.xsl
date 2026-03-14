<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#import?>
  <!-- Purpose: Show the significance of the order of xsl:import declarations. -->

<xsl:import href="c.xsl"/>
<xsl:import href="b.xsl"/>

<xsl:template match="/">
  <out><xsl:apply-templates select="doc/*" /></out>
</xsl:template>

<xsl:template match="title"><!-- This has import precedence over all others -->
  MAIN title matched<xsl:text>...</xsl:text>
  <xsl:apply-imports/>
</xsl:template>

</xsl:stylesheet>