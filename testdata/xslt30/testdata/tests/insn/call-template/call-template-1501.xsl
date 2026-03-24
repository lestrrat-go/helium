<?xml version="1.0"?>
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: namedtemplate13 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.6 -->
    <!-- Purpose: Ensure that we can use the default parameter value on some calls -->

<xsl:template match="/doc">
  <out>
    <xsl:text>Looking for doc...</xsl:text>
    <xsl:choose>
      <xsl:when test="//doc">
        <xsl:call-template name="status">
          <xsl:with-param name="X1">hasDocBelow</xsl:with-param>
        </xsl:call-template>
      </xsl:when>
      <xsl:otherwise>
        <xsl:call-template name="status"/>
      </xsl:otherwise>
    </xsl:choose>
    <xsl:text>&#010;Looking for croc...</xsl:text>
    <xsl:choose>
      <xsl:when test="//croc">
        <xsl:call-template name="status">
          <xsl:with-param name="X1">hasCrocBelow</xsl:with-param>
        </xsl:call-template>
      </xsl:when>
      <xsl:otherwise>
        <xsl:call-template name="status"/>
      </xsl:otherwise>
    </xsl:choose>
    <xsl:text>&#010;Looking for bloc...</xsl:text>
    <xsl:choose>
      <xsl:when test="//bloc">
        <xsl:call-template name="status">
          <xsl:with-param name="X1">hasBlocBelow</xsl:with-param>
        </xsl:call-template>
      </xsl:when>
      <xsl:otherwise>
        <xsl:call-template name="status"/>
      </xsl:otherwise>
    </xsl:choose>
  </out>
</xsl:template>

<xsl:template name="status">
  <xsl:param name="X1">noLowerNode</xsl:param>
  <xsl:text>X1=</xsl:text><xsl:value-of select="$X1"/>
</xsl:template>

</xsl:stylesheet>
