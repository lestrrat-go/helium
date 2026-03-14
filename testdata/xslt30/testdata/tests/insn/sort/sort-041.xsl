<?xml version="1.0"?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: SORT32 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 10 -->
    <!-- Purpose: Try to set data-type from a variable. -->

<xsl:template match="doc">
  <xsl:param name="typer" select="'number'"/>
  <out>
    Ascending order....
    <xsl:for-each select="t">
      <xsl:sort data-type="{$typer}"/>
      <xsl:value-of select="."/><xsl:text>|</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
