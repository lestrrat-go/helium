<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: idkey24 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test for id(string), where string is a whitespace-separated list of values. -->

<xsl:template match="/">
  <n>
    <xsl:for-each select="id('d b c')">
      <xsl:value-of select="./@id"/>
    </xsl:for-each>
  </n>
</xsl:template>

</xsl:stylesheet>
