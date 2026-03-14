<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xpath#axes?>
<?error XPDY0050?>
<!-- PURPOSE:   Test that '/' is an error when the context node is in a non-document tree -->

<xsl:output indent="yes"/>

<xsl:template match="doc">
  <xsl:variable name="e" as="element()">
    <foo/>
  </xsl:variable>
  <xsl:for-each select="$e">
    <xsl:copy-of select="/"/>
  </xsl:for-each>
</xsl:template>
 
</xsl:stylesheet>
