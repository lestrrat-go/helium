<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position48 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 10 -->
    <!-- Purpose: Test position predicate on sort key. -->

<xsl:template match="doc">
  <out>Order should be cfdaegb...
    <xsl:for-each select="alpha">
      <xsl:sort select="z[1]" data-type="number"/>
      <xsl:value-of select="@key"/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>