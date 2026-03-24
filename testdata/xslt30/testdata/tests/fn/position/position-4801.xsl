<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position11 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() and last() in xsl:if test. -->

<xsl:template match="letters">
  <out>
    <xsl:apply-templates select="letter"/>
  </out>
</xsl:template>

<xsl:template match="letter">
  <xsl:value-of select="normalize-space(text())"/>
  <xsl:if test="not(position()=last())">
    <xsl:text>, </xsl:text>
  </xsl:if>
</xsl:template>

</xsl:stylesheet>
