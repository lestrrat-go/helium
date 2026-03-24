<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position85 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.1 -->
    <!-- Purpose: Test that () grouping can be applied multiple times. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//baz">
      <xsl:value-of select="ancestor::foo[1]/@att1"/><xsl:text>, </xsl:text>
      <xsl:value-of select="(ancestor::foo[1])/@att1"/><xsl:text>, </xsl:text>
      <xsl:value-of select="(ancestor::foo)[1]/@att1"/><xsl:text>, </xsl:text>
      <xsl:value-of select="((ancestor::foo))[1]/@att1"/><xsl:text>, </xsl:text>
      <xsl:value-of select="(((ancestor::foo)[1])/@att1)"/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
