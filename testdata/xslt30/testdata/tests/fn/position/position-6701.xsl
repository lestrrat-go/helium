<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position77 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 5.4 -->
  <!-- AdditionalSpec: 4, 10 and "current node list" in 1 -->
    <!-- Purpose: Test positional indexing in current node list passed
       via apply-templates with select that has predicate. -->

<xsl:template match="metadata">
  <out>
    <xsl:apply-templates select="CLASSIFICATION"/>
    <xsl:text>&#10;====== tickers ======&#10;</xsl:text>
    <xsl:apply-templates select="keyword[@tag='ticker']"/>
    <xsl:text>&#10;====== sectors ======&#10;</xsl:text>
    <xsl:apply-templates select="keyword[@tag='sector']"/>
  </out>
</xsl:template>

<xsl:template match="keyword">
  <xsl:value-of select="@value"/>
  <xsl:value-of select="position()"/>
  <xsl:value-of select="last()"/>
  <xsl:if test="position()!=last()">,</xsl:if>
</xsl:template>

<xsl:template match="*">
  <xsl:value-of select="@value"/>
  <xsl:value-of select="position()"/>
  <xsl:value-of select="last()"/>
  <xsl:if test="position()!=last()">,</xsl:if>
</xsl:template>

</xsl:stylesheet>
