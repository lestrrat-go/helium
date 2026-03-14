<?xml version="1.0"?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: SORT27 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 10 -->
    <!-- Purpose: Test for xsl:sort on apply-templates that will get text and element children. -->

<xsl:template match="main">
  <out>
    <xsl:apply-templates>
      <xsl:sort select="normalize-space(.)"/>
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="sub | text()">
  <xsl:value-of select="."/><xsl:text>-</xsl:text>
</xsl:template>

</xsl:stylesheet>
