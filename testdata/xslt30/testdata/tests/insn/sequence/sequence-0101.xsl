<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- PURPOSE:   Test sequence construction in a variable -->
<?spec xslt#variable-values?>

<xsl:template match="doc">
  <xsl:variable name="q" as="item() *">
    <xsl:text>a</xsl:text>
    <xsl:text>b</xsl:text>
    <xsl:text>c</xsl:text>
  </xsl:variable>
  <z><xsl:value-of select="data($q)" separator=","/></z>
</xsl:template>
 
</xsl:stylesheet>
