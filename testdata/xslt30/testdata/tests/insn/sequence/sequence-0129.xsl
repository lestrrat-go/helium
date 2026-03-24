<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#constructing-sequences?>
<?error XTTE0570?>
<!-- PURPOSE:   Test sequence construction where a constructed item has the wrong cardinality -->

<xsl:template match="doc">
  <xsl:variable name="q" as="text() ?">
    <xsl:text>a</xsl:text>
    <xsl:text>b</xsl:text>
    <xsl:text>c</xsl:text>
  </xsl:variable>
  <z><xsl:value-of select="$q" separator=","/></z>
</xsl:template>
 
</xsl:stylesheet>
