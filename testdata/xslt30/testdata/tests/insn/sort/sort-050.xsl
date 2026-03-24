<?xml version="1.0"?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: SORT40 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 10 -->
    <!-- Purpose: Loop over a node-set in for-each and sort the nodes. -->

<xsl:template match="/">
  <xsl:variable name="numnodes" select="doc/num" />
  <out>
    Descending order....
    <xsl:for-each select="$numnodes">
      <xsl:sort data-type="number" order="descending"/>
      <xsl:value-of select="."/><xsl:text> </xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>