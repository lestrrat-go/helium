<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<?spec dm#CommentNode?>
<!-- BUG:  Handle a zero-length comment -->

<xsl:template match="/">
<out>
  <xsl:copy-of select="."/>
  <xsl:for-each select="//comment()">[<xsl:value-of select="."/>]</xsl:for-each>
</out>  
</xsl:template>

</xsl:stylesheet>
