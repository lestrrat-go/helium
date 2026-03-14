<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-if?>
    <!-- Purpose: Test xsl:if with test involving current node value. -->

<xsl:template match="letters">
  <out>
    <xsl:apply-templates select="letter"/>
  </out>
</xsl:template>

<xsl:template match="letter">
  <xsl:if test=".='b'">
    <xsl:text>Found b  </xsl:text>
  </xsl:if>
</xsl:template>

</xsl:stylesheet>
