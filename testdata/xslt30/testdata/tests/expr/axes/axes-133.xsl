<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: Traverse the ancestor::* axis -->
  <!-- Elaboration: First, jump to the center (via for-each so that we don't have interfering
    templates, which is less of a problem for ancestor than other axes), then apply-templates
    on all members of the axis under test. Also, demonstrate the statement in XSLT 5.4 that
    "The selected set of nodes is processed in document order, unless a sorting specification
    is present." -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center">
      <xsl:apply-templates select="ancestor::*"/>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="name(.)"/><xsl:text> </xsl:text>
</xsl:template>

</xsl:stylesheet>
