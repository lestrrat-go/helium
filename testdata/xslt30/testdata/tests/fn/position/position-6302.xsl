<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position73 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
    <!-- Purpose: Test of position() and for-each resetting the frame of reference of a node-set. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select=".//f"/>
  </out>
</xsl:template>

<xsl:template match="f">
  <xsl:for-each select="ancestor-or-self::*">
    <xsl:value-of select="position()"/>
    <xsl:text>. </xsl:text>
    <xsl:choose>
      <xsl:when test="count(./*)=0">
        <xsl:value-of select="."/>
      </xsl:when>
      <xsl:otherwise>
        <xsl:value-of select="text()[1]"/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
