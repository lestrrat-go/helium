<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- PURPOSE:   Test that root() is OK when the context node is in a non-document tree -->


<xsl:template match="doc">
  <xsl:variable name="e" as="element()">
    <foo><bar/></foo>
  </xsl:variable>
  <xsl:for-each select="$e/bar">
    <xsl:copy-of select="root()"/>
  </xsl:for-each>
</xsl:template>
 
</xsl:stylesheet>
