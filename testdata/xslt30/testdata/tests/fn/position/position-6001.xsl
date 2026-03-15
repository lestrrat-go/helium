<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position68 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
    <!-- Purpose: Test positional indexing with for-each loop and all-but-last xsl:if for comma. -->

<xsl:template match="metadata">
  <out>
    <xsl:for-each select="keyword[@tag='ticker']">
      <xsl:value-of select="@value"/>
      <xsl:value-of select="position()"/>
      <xsl:value-of select="last()"/>
      <xsl:if test="position()!=last()">,</xsl:if>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
