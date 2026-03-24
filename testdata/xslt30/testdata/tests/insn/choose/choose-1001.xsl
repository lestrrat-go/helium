<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
    <!-- Purpose: Test that only the content of the first matching xsl:when is instantiated. -->

<xsl:template match="doc">
  <out>
    <xsl:choose>
      <xsl:when test="foo &lt; 2">1</xsl:when>
      <xsl:when test="foo &lt; 4">2</xsl:when>
      <xsl:when test="foo &lt; 8">3</xsl:when>
      <xsl:when test="foo &lt; 16">4</xsl:when>
      <xsl:when test="foo &lt; 32">5</xsl:when>
      <xsl:otherwise>0</xsl:otherwise>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>
