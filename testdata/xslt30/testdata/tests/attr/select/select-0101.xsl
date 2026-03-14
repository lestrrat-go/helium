<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#for-each?>
    <!-- Purpose: Test for absolute path selection.-->

<xsl:template match="/">
  <out>
    <xsl:for-each select="doc/c/d/e">
      <xsl:value-of select="/doc/a/b/@attr"/>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
