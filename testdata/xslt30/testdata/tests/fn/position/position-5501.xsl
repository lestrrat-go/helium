<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position31 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() with a sibling axis. -->

<xsl:template match="items">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="sfdhd">
  <xsl:text>Pos</xsl:text><xsl:value-of select="position()"/><xsl:text> G</xsl:text><xsl:value-of select="@group"/><xsl:text>- </xsl:text><xsl:value-of select="."/>
</xsl:template>

<xsl:template match="bracket|button|dial|light">
  <xsl:choose>
    <xsl:when test="position()=2"><!-- Have to skip interstitial text -->
      <xsl:text>Group: </xsl:text><xsl:value-of select="@group"/><xsl:text>
     </xsl:text><xsl:value-of select="."/><xsl:text> </xsl:text><xsl:value-of select="name(.)"/>
    </xsl:when>
    <xsl:when test="@group != preceding-sibling::*[1]/@group">
      <xsl:text>Group: </xsl:text><xsl:value-of select="@group"/><xsl:text>
     </xsl:text><xsl:value-of select="."/><xsl:text> </xsl:text><xsl:value-of select="name(.)"/>
    </xsl:when>
    <xsl:otherwise>
      <xsl:text>   </xsl:text><xsl:value-of select="."/><xsl:text> </xsl:text><xsl:value-of select="name(.)"/>
    </xsl:otherwise>
  </xsl:choose>
</xsl:template>

</xsl:stylesheet>
