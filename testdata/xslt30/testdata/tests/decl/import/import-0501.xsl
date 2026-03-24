<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#include?>
  <!-- Purpose: Two includes (see below), each of which has an import. -->

<xsl:template match="/">
  <out><xsl:apply-templates select="doc/*" /></out>
</xsl:template>

<xsl:template match="author">
  MAIN author: <xsl:text>How did THIS get called?</xsl:text>
</xsl:template>

<xsl:template match="chapter"><!-- This has import precedence over e.xsl -->
    MAIN chapter <xsl:value-of select="@num"/><xsl:text>: </xsl:text>
  <xsl:value-of select="."/>
</xsl:template>

<!-- INCLUDES can be anywhere and are positionally significant -->
<xsl:include href="b.xsl"/>
<xsl:include href="c.xsl"/>

<xsl:template match="title"><!-- This has import precedence over d.xsl, plus it's last -->
  MAIN title matched<xsl:text>...</xsl:text>
</xsl:template>

</xsl:stylesheet>