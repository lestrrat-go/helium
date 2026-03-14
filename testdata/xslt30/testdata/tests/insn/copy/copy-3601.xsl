<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: copy52 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 7.5 -->
    <!-- Purpose: Make each PI of a certain name be the current node, and copy it. -->

<xsl:template match="/">
  <out>
    <xsl:for-each select="//processing-instruction('b-pi')">
      <xsl:copy/><xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>