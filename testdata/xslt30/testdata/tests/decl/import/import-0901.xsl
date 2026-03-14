<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#apply-imports?>
    <!-- Purpose: Show that apply-imports really means imports, not includes -->

<xsl:template match="/">
  <out><xsl:apply-templates select="doc/*" /></out>
</xsl:template>

<xsl:template match="author">
  MAIN author: <xsl:value-of select="."/>
</xsl:template>

<!-- INCLUDES can be anywhere and are positionally significant -->
<xsl:include href="c.xsl"/>

<xsl:template match="title"><!-- This has import precedence over e.xsl, plus it's last -->
  MAIN title: <xsl:value-of select="."/>
  <xsl:apply-imports/>
</xsl:template>

</xsl:stylesheet>