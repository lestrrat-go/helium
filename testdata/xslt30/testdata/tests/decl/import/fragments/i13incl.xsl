<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<xsl:template match="title">
  <xsl:text>Template for title in included file should NOT be used.</xsl:text>
</xsl:template>

<xsl:template match="overview">
  <!-- This one should have highest import precedence, since it's included into main. -->
  <xsl:element name="{@x}">
    <xsl:apply-imports/><!-- go get next-highest-precedence overview template -->
  </xsl:element>
</xsl:template>

</xsl:stylesheet>