<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position69 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
    <!-- Purpose: Test positional indexing with for-each loop and apply-templates. -->

<xsl:template match="metadata">
  <out>
    <xsl:text>&#010;</xsl:text>
    <xsl:for-each select="keyword[@tag='ticker']">
      <xsl:sort select="@value"/>
      <xsl:value-of select="@value"/>
      <xsl:value-of select="position()"/>
      <xsl:value-of select="last()"/>
      <xsl:if test="position()!=last()">,</xsl:if>
    </xsl:for-each>

    <xsl:text>&#010;</xsl:text>

    <xsl:apply-templates select="keyword[@tag='ticker']">
      <xsl:sort select="@value"/>
    </xsl:apply-templates>
    <xsl:text>&#010;</xsl:text>
  </out>
</xsl:template>

<xsl:template match="keyword">
  <xsl:value-of select="@value"/>
  <xsl:value-of select="position()"/>
  <xsl:value-of select="last()"/>
  <xsl:if test="position()!=last()">,</xsl:if>
</xsl:template>

</xsl:stylesheet>
