<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: copy42 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.3 -->
    <!-- Purpose: Demonstrate sorting of tree fragments -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="doc//a3[../@pick='yes']">
      <xsl:sort select="@ordering" data-type="number"/>
      <xsl:copy-of select="."/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>