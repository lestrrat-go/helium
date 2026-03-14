<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-if?>
  <!-- Purpose: Test xsl:if with simplistic boolean expression. -->

<xsl:template match="/">
  <out>
  <xsl:if test="2=2">
    <xsl:text>number </xsl:text>
  </xsl:if>
  <xsl:if test="'a'='a'">
    <xsl:text>string</xsl:text>
  </xsl:if>
  </out>
</xsl:template>

</xsl:stylesheet>
