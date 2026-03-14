<?xml version="1.0"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position10 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test of position() function and sorting. -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="foo/a/num/@val">
      <xsl:value-of select="position()"/>_<xsl:value-of select="."/>
      <xsl:text> </xsl:text>
    </xsl:for-each>
  Now, sort the data...    
    <xsl:for-each select="foo/a/num/@val">
      <xsl:sort select="."/>
      <xsl:value-of select="position()"/>_<xsl:value-of select="."/>
      <xsl:text> </xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
