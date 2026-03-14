<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: copy54 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 7.5 -->
    <!-- Purpose: Copy some PIs with a multiply-filtered select. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//center//processing-instruction('b-pi')">
      <xsl:copy/><xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>