<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position27 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of positional capabilities with choose. -->

<xsl:template match="letters">
  <out>
    <xsl:apply-templates select="letter"/>
  </out>
</xsl:template>

<xsl:template match="letter">
  <xsl:choose>
    <xsl:when test="position()=1">
      <xsl:text>First: </xsl:text><xsl:value-of select="."/><xsl:text>+</xsl:text>
    </xsl:when>
    <xsl:when test="position()=last()">
      <xsl:text>+Last: </xsl:text><xsl:value-of select="."/>
    </xsl:when>
    <xsl:when test="position()=ceiling(last() div 2)">
      <xsl:text>-Middle: </xsl:text><xsl:value-of select="."/><xsl:text>-</xsl:text>
    </xsl:when>
    <xsl:otherwise>
      <xsl:value-of select="."/>
    </xsl:otherwise>
  </xsl:choose>
</xsl:template>

</xsl:stylesheet>
