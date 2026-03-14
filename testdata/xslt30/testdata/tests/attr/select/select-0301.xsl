<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#current-function?>
    <!-- Purpose: Test for select in for-each and current(). -->

<xsl:template match="doc">
  <out>
    <xsl:for-each select="foo">
      <xsl:copy-of select="//item[@type=current()/@name]"/><xsl:text> </xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
