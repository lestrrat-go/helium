<?xml version="1.0"?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: SORT11 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 10 -->
    <!-- Purpose: Sort strings that all can be numbers without specifying data-type or order. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="item">
      <xsl:sort/>
      <xsl:copy-of select="."/><xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
