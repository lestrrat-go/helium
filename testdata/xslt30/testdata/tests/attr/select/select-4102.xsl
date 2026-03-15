<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#combining_seq?>
  <!-- Purpose: NodeSet union using the descendant-or-self axis -->
  <!-- Expression010 in NIST suite -->

<xsl:template match="/doc">
  <out>
    <xsl:apply-templates select="descendant-or-self::sub1|descendant-or-self::sub2"/>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select = "."/>
</xsl:template>

</xsl:stylesheet>

