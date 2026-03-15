<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-if?>
    <!-- Purpose: Test xsl:if with boolean function and test of current node. -->

<xsl:template match="letters">
  <out>
    <xsl:apply-templates select="letter"/>
  </out>
</xsl:template>

<xsl:template match="letter">
  <xsl:if test="not(.='b')">
    <xsl:text>not_b </xsl:text>
  </xsl:if>
</xsl:template>

</xsl:stylesheet>
